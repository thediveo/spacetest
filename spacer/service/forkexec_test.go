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

package service

import (
	"io"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/thediveo/safe"
	"golang.org/x/sys/unix"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gleak"
)

// Sometimes, you have to (unit) test the expected Linux system behavior, as
// opposed to testing your own code. This serves both as system behavior
// documentation, as well as repeatable system behavior checking.
var _ = Describe("Linux namespace unsharing when forking/executing", func() {

	BeforeEach(func() {
		goodgos := Goroutines()
		DeferCleanup(func() {
			Eventually(Goroutines).Within(2 * time.Second).ProbeEvery(100 * time.Millisecond).
				ShouldNot(HaveLeaked(goodgos))
		})
	})

	When("not being root", func() {

		BeforeEach(func() {
			if os.Getuid() == 0 {
				Skip("needs non-root")
			}
		})

		It("denies mapping root uid and gid", func() {
			cmd := exec.Command("/bin/true")
			cmd.Stdout = GinkgoWriter
			cmd.Stderr = GinkgoWriter
			cmd.SysProcAttr = &syscall.SysProcAttr{
				Cloneflags: unix.CLONE_NEWUSER,
				UidMappings: []syscall.SysProcIDMap{
					{
						HostID:      0,
						ContainerID: 0,
						Size:        1,
					},
				},
				GidMappings: []syscall.SysProcIDMap{
					{
						HostID:      0,
						ContainerID: 0,
						Size:        1,
					},
				},
			}
			Expect(cmd.Run()).To(MatchError(
				ContainSubstring("fork/exec /bin/true: operation not permitted")))
		})

		It("allows mapping the current user to root inside a child user namespace, then creating a network namespace", func() {
			cmd := exec.Command("unshare", "-n")
			cmd.Stdout = GinkgoWriter
			cmd.Stderr = GinkgoWriter
			cmd.SysProcAttr = &syscall.SysProcAttr{
				Cloneflags: unix.CLONE_NEWUSER,
				UidMappings: []syscall.SysProcIDMap{
					{
						HostID:      os.Getuid(),
						ContainerID: 0,
						Size:        1,
					},
				},
				GidMappings: []syscall.SysProcIDMap{
					{
						HostID:      os.Getgid(),
						ContainerID: 0,
						Size:        1,
					},
				},
			}
			Expect(cmd.Run()).To(Succeed())
		})

		It("allows user namespace unsharing when not mapping any uids and gids, but not creating a new network namespace", func() {
			var out safe.Buffer

			cmd := exec.Command("unshare", "-n")
			cmd.Stdout = io.MultiWriter(&out, GinkgoWriter)
			cmd.Stderr = cmd.Stdout
			cmd.SysProcAttr = &syscall.SysProcAttr{
				Cloneflags: unix.CLONE_NEWUSER,
			}
			Expect(cmd.Start()).To(Succeed(),
				"expected unshare to successfully start before failing")
			Expect(cmd.Wait()).To(MatchError(ContainSubstring("exit status 1")))
			Expect(out.String()).To(Equal("unshare: unshare failed: Operation not permitted\n"))
		})

	})

})
