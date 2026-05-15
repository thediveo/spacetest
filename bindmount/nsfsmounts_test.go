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
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gleak"
	. "github.com/thediveo/fdooze"
)

var _ = Describe("listing nsfs type mounts", func() {

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

	It("fails the spec if the path is not readable", func() {
		Expect(InterceptGomegaFailure(func() {
			for range AllNsfsMountsPath("/foobar") {
			}
		})).To(MatchError(ContainSubstring("no such file or directory")))
	})

	It("produces only correct nsfs bind mounts", func() {
		Expect(AllNsfsMountsPath("./_testdata/mountinfo")).To(ConsistOf("/tmp/bindmount"))
	})

	It("aborts the iterator", func() {
		Expect(AllNsfsMountsPath("./_testdata/mountinfo-multi")).To(HaveLen(2))
		count := 0
		for range AllNsfsMountsPath("./_testdata/mountinfo-multi") {
			count++
			break
		}
		Expect(count).To(Equal(1))
	})

	It("iterates or not", func() {
		for range AllNsfsMounts() {
		}
	})

})
