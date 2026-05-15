// Copyright 2026 Harald Albrecht.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package bindmount

import (
	"iter"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/thediveo/spacetest/netns"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gleak"
	. "github.com/thediveo/fdooze"
)

const (
	mountIDField = iota
	parentIDField
	stdevField
	rootField
	mountpointField
)

const (
	fsTypeField = iota
)

func allNsfsMounts() iter.Seq[string] {
	return func(yield func(string) bool) {
		mntinfo, err := os.ReadFile("/proc/thread-self/mountinfo")
		if err != nil {
			return
		}

		for line := range strings.Lines(string(mntinfo)) {
			block1, block2, ok := strings.Cut(line, " - ")
			if !ok {
				continue
			}

			beforeSep := strings.Fields(block1)
			if len(beforeSep) < 5 {
				continue
			}

			afterSep := strings.Fields(block2)
			if len(afterSep) < 1 || afterSep[fsTypeField] != "nsfs" {
				continue
			}

			if !yield(beforeSep[mountpointField]) {
				return
			}
		}
	}
}

var _ = Describe("bind-mounting namespaces", func() {

	BeforeEach(func() {
		if os.Getuid() != 0 {
			Skip("needs root")
		}
		goodfds := Filedescriptors()
		goodgos := Goroutines()
		DeferCleanup(func() {
			Eventually(Goroutines).Within(2 * time.Second).ProbeEvery(250 * time.Millisecond).
				ShouldNot(HaveLeaked(goodgos))
			Expect(Filedescriptors()).NotTo(HaveLeakedFds(goodfds))
		})
	})

	It("bind-mounts a namespace fd", func() {
		netnsfd := netns.NewTransient()
		bm := NewTransient(netnsfd)
		Expect(allNsfsMounts()).To(ContainElement(bm))
		Expect(netns.Ino(bm)).To(Equal(netns.Ino(netnsfd)))
	})

	It("bind-mounts a namespace path", func() {
		netnsfd := netns.NewTransient()
		ref := "/proc/thread-self/fd/" + strconv.FormatInt(int64(netnsfd), 10)
		bm := NewTransient(ref)
		Expect(allNsfsMounts()).To(ContainElement(bm))
		Expect(netns.Ino(bm)).To(Equal(netns.Ino(netnsfd)))
	})

	It("doesn't bind mount an unsuitable fd reference", func() {
		Expect(InterceptGomegaFailure(func() {
			_ = NewTransient(0)
		})).To(MatchError(MatchRegexp(`(?s)cannot determine type .* inappropriate ioctl`)))
	})

	It("doesn't bind mount an invalid string reference", func() {
		Expect(InterceptGomegaFailure(func() {
			_ = NewTransient("/proc/self")
		})).To(MatchError(MatchRegexp(`(?s)cannot determine type .* inappropriate ioctl`)))
	})

})
