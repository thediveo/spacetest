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

package mntns

import (
	"os"
	"time"

	"github.com/thediveo/spacetest/netns"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gleak"
	. "github.com/thediveo/fdooze"
	. "github.com/thediveo/success"
)

var _ = Describe("sysfs", func() {

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

	It("rejects mounting sysfs in the original mount namespace", func() {
		Expect(InterceptGomegaFailure(func() {
			// Here goes nothing ... hope and pray that we fail! At least with
			// devcontainers we would trash only the non-critical part of the
			// devcontainer's filesystem within the container's mount namespace,
			// so nothing a fresh start of the container wouldn't correct.
			MountSysfsRO()
		})).To(MatchError(
			ContainSubstring("current mount namespace must not be the process's original mount namespace")))
	})

	It("mounts a fresh sysfs (RO) in a transient mount namespace", func() {
		defer netns.EnterTransient()()
		Expect(len(Successful(os.ReadDir("/sys/class/net")))).To(
			BeNumerically(">", 1), "expecting lo and more, like eth0")

		defer EnterTransient()() // well, only for symmetry
		MountSysfsRO()

		Expect(Successful(os.ReadDir("/sys/class/net"))).To(
			ConsistOf(HaveField("Name()", "lo")))
	})

})
