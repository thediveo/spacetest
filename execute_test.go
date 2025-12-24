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
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"syscall"
	"time"

	"github.com/thediveo/caps"
	"golang.org/x/sys/unix"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gleak"
	. "github.com/thediveo/fdooze"
	. "github.com/thediveo/success"
)

var _ = Describe("doing things in different namespaces", Ordered, func() {

	BeforeAll(func() {
		if os.Getuid() != 0 {
			Skip("needs root")
		}
	})

	// Nota bene: we cannot use a top-level BeforeEach() to check for go routine
	// and file descriptor leaks, because some tests further down would trigger
	// false positives. So we're using multiple BeforeEach()es on the next
	// level. Also, make sure to correctly Wait() for the idler processes to
	// have finally gone.

	When("switching namespaces on the current go routine", func() {

		BeforeEach(func() {
			goodfds := Filedescriptors()
			goodgos := Goroutines()
			DeferCleanup(func() {
				Eventually(Goroutines).Within(2 * time.Second).ProbeEvery(100 * time.Millisecond).
					ShouldNot(HaveLeaked(goodgos))
				Expect(Filedescriptors()).NotTo(HaveLeakedFds(goodfds))
			})
		})

		It("succeeds switching in and out a net and uts namespace", func() {
			origNetns := Current(unix.CLONE_NEWNET)
			CurrUtsns := Current(unix.CLONE_NEWUTS)

			netns := NewTransient(unix.CLONE_NEWNET)
			utsns := NewTransient(unix.CLONE_NEWUTS)

			count := 0
			goInAndOut(func() {
				count++
				Expect(Ino("/proc/thread-self/ns/net", unix.CLONE_NEWNET)).To(
					Equal(Ino(netns, unix.CLONE_NEWNET)), "net switch failed")
				Expect(Ino("/proc/thread-self/ns/uts", unix.CLONE_NEWUTS)).To(
					Equal(Ino(utsns, unix.CLONE_NEWUTS)), "uts switch failed")
			}, netns, utsns)
			Expect(count).To(Equal(1), "fn wasn't called")

			Expect(Ino("/proc/thread-self/ns/net", unix.CLONE_NEWNET)).To(
				Equal(Ino(origNetns, unix.CLONE_NEWNET)), "net restore failed")
			Expect(Ino("/proc/thread-self/ns/uts", unix.CLONE_NEWUTS)).To(
				Equal(Ino(CurrUtsns, unix.CLONE_NEWUTS)), "uts restore failed")
		})

		It("fails when switch fails due to invalid namespace reference", func() {
			Expect(InterceptGomegaFailure(func() {
				goInAndOut(func() {}, -1)
			})).To(MatchError(ContainSubstring("cannot determine type of namespace")))
		})

		It("fails when switch fails due to not being allowed to switch", func() {
			runtime.LockOSThread() // throw away

			// Creating new user namespace in a Go program doesn't seem possible
			// because unshare(2) only allows so when the caller is
			// single-threaded. We thus run a full idle process ("/bin/sleep")
			// and tell exec to plant it into a new user namespace, which we
			// then pick up from the process file system.
			sleep := exec.Command("/bin/sleep", "1h")
			sleep.SysProcAttr = &syscall.SysProcAttr{
				Cloneflags: unix.CLONE_NEWUSER,
				UidMappings: []syscall.SysProcIDMap{
					{
						ContainerID: 0,
						HostID:      os.Getuid(),
						Size:        1,
					},
				},
				GidMappings: []syscall.SysProcIDMap{
					{
						ContainerID: 0,
						HostID:      os.Getgid(),
						Size:        1,
					},
				},
			}
			Expect(sleep.Start()).To(Succeed())
			defer func() {
				if err := sleep.Process.Kill(); err == nil {
					_ = sleep.Wait()
				}
			}()
			usernsfd := Successful(
				unix.Open(fmt.Sprintf("/proc/%d/ns/user", sleep.Process.Pid),
					os.O_RDONLY, 0))
			defer func() { _ = unix.Close(usernsfd) }()

			Expect(Ino(usernsfd, unix.CLONE_NEWUSER)).NotTo(BeZero())
			Expect(Ino(usernsfd, unix.CLONE_NEWUSER)).NotTo(
				Equal(CurrentIno(unix.CLONE_NEWUSER)))

			Expect(InterceptGomegaFailure(func() {
				// Ironically, user namespaces are so zupa zekure that we can't
				// switch to them because we're multi-threaded at this time.
				goInAndOut(func() {}, usernsfd)
			})).To(MatchError(ContainSubstring("cannot switch into user namespace")))
		})

		It("fails correctly when unable to switch back", func() {
			// This unit test goes some way to test every last bit of code by
			// manipulating the thread that executes the callback function in
			// such a way that restoring the network namespace becomes
			// impossible; for this we drop our capabilities.

			runtime.LockOSThread() // this thread will be tainted

			netns := NewTransient(unix.CLONE_NEWNET)

			count := 0
			Expect(InterceptGomegaFailure(func() {
				goInAndOut(func() {
					count++
					Expect(caps.SetForThisTask(caps.TaskCapabilities{})).To(Succeed())
				}, netns)
			})).To(MatchError(ContainSubstring("cannot restore net namespace")))
			Expect(count).To(Equal(1))
		})

	})

	When("switching namespaces on a transient go routine", func() {

		var mntnsfd int

		BeforeAll(func() {
			sleep := exec.Command("/bin/sleep", "1h")
			sleep.SysProcAttr = &syscall.SysProcAttr{
				Cloneflags: unix.CLONE_NEWNS,
			}
			Expect(sleep.Start()).To(Succeed())
			DeferCleanup(func() {
				_ = sleep.Process.Kill()
			})
			mntnsfd = Successful(
				unix.Open(fmt.Sprintf("/proc/%d/ns/mnt", sleep.Process.Pid),
					os.O_RDONLY, 0))
			DeferCleanup(func() {
				_ = unix.Close(mntnsfd)
			})
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

		It("successfully runs on a different go routine with different mount+net namespaces", func() {
			runtime.LockOSThread() // throw away

			Expect(Ino(mntnsfd, unix.CLONE_NEWNS)).NotTo(BeZero())
			Expect(Ino(mntnsfd, unix.CLONE_NEWNS)).NotTo(
				Equal(CurrentIno(unix.CLONE_NEWNS)))

			netnsfd := NewTransient(unix.CLONE_NEWNET)

			tid := unix.Gettid()
			count := 0
			goSeparate(func() {
				defer GinkgoRecover()
				count++

				Expect(unix.Gettid()).NotTo(Equal(tid), "not on a different thread")
				Expect(Ino("/proc/thread-self/ns/mnt", unix.CLONE_NEWNS)).To(
					Equal(Ino(mntnsfd, syscall.CLONE_NEWNS)))
				Expect(Ino("/proc/thread-self/ns/net", unix.CLONE_NEWNET)).To(
					Equal(Ino(netnsfd, syscall.CLONE_NEWNET)))
			}, mntnsfd, netnsfd)
			Expect(count).To(Equal(1), "fn wasn't called")
		})

		It("fails when switch fails due to invalid namespace reference", func() {
			Expect(InterceptGomegaFailure(func() {
				goSeparate(func() {}, 0)
			})).To(MatchError(ContainSubstring("cannot switch into mnt namespace")))
		})

	})

	When("directly executing in different namespaces", func() {

		BeforeEach(func() {
			goodfds := Filedescriptors()
			goodgos := Goroutines()
			DeferCleanup(func() {
				Eventually(Goroutines).Within(2 * time.Second).ProbeEvery(100 * time.Millisecond).
					ShouldNot(HaveLeaked(goodgos))
				Expect(Filedescriptors()).NotTo(HaveLeakedFds(goodfds))
			})
		})

		It("switches to a different namespace while picking up other namespaces", func() {
			defer EnterTransient(unix.CLONE_NEWIPC)()
			ipcns := Current(unix.CLONE_NEWIPC)

			netns := NewTransient(unix.CLONE_NEWNET)

			// We do a "hard" fd leak test here to make sure that Execute does
			// not do any fd cleanup at the unit test level, because that could
			// cause problems where API users need to use Execute in a
			// DeferCleanup() fn.
			goodFds := Filedescriptors()

			count := 0
			Execute(func() {
				count++
				Expect(CurrentIno(unix.CLONE_NEWIPC)).To(
					Equal(Ino(ipcns, unix.CLONE_NEWIPC)), "didn't brought over the ipc namespace")
				Expect(CurrentIno(unix.CLONE_NEWNET)).To(
					Equal(Ino(netns, unix.CLONE_NEWNET)), "didn't switch the net namespace")
			}, netns)
			Expect(count).To(Equal(1), "didn't call fn")

			Expect(Filedescriptors()).NotTo(HaveLeakedFds(goodFds))
		})

		It("switches the mount namespace successfully", func() {
			sleep := exec.Command("/bin/sleep", "1h")
			sleep.SysProcAttr = &syscall.SysProcAttr{
				Cloneflags: unix.CLONE_NEWNS,
			}
			Expect(sleep.Start()).To(Succeed())
			DeferCleanup(func() {
				if err := sleep.Process.Kill(); err == nil {
					_ = sleep.Wait()
				}
			})
			mntns := Successful(
				unix.Open(fmt.Sprintf("/proc/%d/ns/mnt", sleep.Process.Pid),
					os.O_RDONLY, 0))
			DeferCleanup(func() {
				_ = unix.Close(mntns)
			})

			defer EnterTransient(unix.CLONE_NEWIPC)()
			ipcns := Current(unix.CLONE_NEWIPC)

			netns := NewTransient(unix.CLONE_NEWNET)

			count := 0
			Execute(func() {
				count++

				Expect(CurrentIno(unix.CLONE_NEWNS)).To(
					Equal(Ino(mntns, unix.CLONE_NEWNS)), "didn't switch the mnt namespace")
				Expect(CurrentIno(unix.CLONE_NEWIPC)).To(
					Equal(Ino(ipcns, unix.CLONE_NEWIPC)), "didn't brought over the ipc namespace")
				Expect(CurrentIno(unix.CLONE_NEWNET)).To(
					Equal(Ino(netns, unix.CLONE_NEWNET)), "didn't switch the net namespace")
			}, mntns, netns)
			Expect(count).To(Equal(1), "didn't call fn")
		})

		It("fails when the separate fn go routine fails switching", func() {
			Expect(InterceptGomegaFailure(func() {
				Execute(func() {}, Current(unix.CLONE_NEWNS), -1)
			})).To(MatchError(ContainSubstring("cannot determine type of namespace")))
		})

		It("rejects to switch user namespaces", func() {
			Expect(InterceptGomegaFailure(func() {
				Execute(func() {}, Current(unix.CLONE_NEWUSER))
			})).To(MatchError(ContainSubstring("cannot Execute() in different user namespace")))
		})

		Specify("Execute can be used in a DeferCleanup func", func() {
			// NewTransient schedules a DeferCleanup for closing the namespace
			// fd it allocated.
			netns := NewTransient(unix.CLONE_NEWNET)

			// Run the next DeferCleanup before closing the namespace,
			// exercising Execute.
			DeferCleanup(func() {
				count := 0
				Execute(func() {
					count++
				}, netns)
				Expect(count).To(Equal(1), "didn't call fn")
			})
		})

	})

})
