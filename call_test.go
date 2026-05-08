// Copyright 2025 Harald Albrecht.
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

package spacetest

import (
	"os"
	"time"

	"golang.org/x/sys/unix"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gleak"
	. "github.com/thediveo/fdooze"
)

var _ = Describe("calling things with results in different namespaces", Ordered, func() {

	BeforeAll(func() {
		if os.Getuid() != 0 {
			Skip("needs root")
		}
	})

	BeforeEach(func() {
		goodfds := Filedescriptors()
		goodgos := Goroutines()
		DeferCleanup(func() {
			Eventually(Goroutines).Within(2 * time.Second).ProbeEvery(100 * time.Millisecond).
				ShouldNot(HaveLeaked(goodgos))
			Expect(Filedescriptors()).NotTo(HaveLeakedFds(goodfds))
		})
	})

	It("returns the result of calling a fn in a different net and uts namespace", func() {
		netns := NewTransient(unix.CLONE_NEWNET)
		utsns := NewTransient(unix.CLONE_NEWUTS)
		Expect(Call(func() uint64 {
			return Ino(utsns, unix.CLONE_NEWUTS)
		}, netns, utsns)).To(Equal(Ino(utsns, unix.CLONE_NEWUTS)))
	})

})
