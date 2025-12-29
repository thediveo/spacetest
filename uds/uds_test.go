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

package uds

import (
	"io"
	"os"
	"time"

	"golang.org/x/sys/unix"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/thediveo/fdooze"
	. "github.com/thediveo/success"
)

var _ = Describe("unix domain sockets (UDS's)", func() {

	BeforeEach(func() {
		goodfds := Filedescriptors()
		DeferCleanup(func() {
			Eventually(Filedescriptors).Within(2 * time.Second).ProbeEvery(100 * time.Millisecond).
				ShouldNot(HaveLeakedFds(goodfds))
		})
	})

	When("transforming a file descriptor into a UnixConn", func() {

		It("returns a UnixConn without leaking fds", func() {
			goodfds := Filedescriptors()
			DeferCleanup(func() {
				// yes, we are checking only once, so it better be correct.
				Expect(Filedescriptors()).NotTo(HaveLeakedFds(goodfds))
			})

			fdpair := Successful(unix.Socketpair(unix.AF_UNIX, unix.SOCK_DGRAM, 0))
			Expect(unix.Close(fdpair[1])).To(Succeed())
			unixconn := Successful(NewUnixConn(fdpair[0], "dupond"))
			Expect(unixconn.Close()).To(Succeed())
		})

		It("returns an error when the passed fd is bonkers", func() {
			Expect(NewUnixConn(-1, "nada")).Error().To(MatchError(
				ContainSubstring("not a file descriptor")))

			Expect(NewUnixConn(0, "nada")).Error().To(MatchError(
				ContainSubstring("socket operation on non-socket")))

			udpsockfd := Successful(unix.Socket(unix.AF_INET, unix.SOCK_DGRAM, 0))
			defer func() { _ = unix.Close(udpsockfd) }()
			Expect(NewUnixConn(udpsockfd, "nada")).Error().To(MatchError(
				ContainSubstring("not a unix domain socket")))
		})

	})

	When("transferring open file descriptors", func() {

		It("succeeds", func() {
			dupond, dupont := Successful2R(NewPair())
			defer func() {
				_ = dupond.Close()
				_ = dupont.Close()
			}()
			Expect(dupond).NotTo(BeNil())
			Expect(dupont).NotTo(BeNil())

			canaryfd := Successful(unix.Open("./_testdata/canary.dat", unix.O_RDONLY, 0))
			defer func() { _ = unix.Close(canaryfd) }()
			go func() {
				defer GinkgoRecover()
				Expect(dupond.SendWithFds(nil, canaryfd)).Error().NotTo(HaveOccurred())
			}()

			Expect(dupont.SetReadDeadline(time.Now().Add(2 * time.Second))).To(Succeed())
			_, fds := Successful2R(dupont.ReceiveWithFds(nil, 16))
			defer func() {
				for _, fd := range fds {
					_ = unix.Close(fd)
				}
			}()
			Expect(fds).To(HaveLen(1))
			fd := fds[0]
			Expect(fd).NotTo(Equal(canaryfd))

			// note: the received file descriptor has the same underlying read
			// position, so we need to rewind after reading it once.
			canary := Successful(io.ReadAll(os.NewFile(uintptr(canaryfd), "canary.dat")))
			Expect(canary).NotTo(BeEmpty())
			Expect(unix.Seek(fd, 0, unix.SEEK_SET)).Error().NotTo(HaveOccurred())
			received := Successful(io.ReadAll(os.NewFile(uintptr(fd), "received")))
			Expect(canary).To(Equal(received))
		})

		It("returns an error when there is no control message sent", func() {
			dupond, dupont := Successful2R(NewPair())
			defer func() {
				_ = dupond.Close()
				_ = dupont.Close()
			}()

			Expect(dupont.SetReadDeadline(time.Now().Add(1 * time.Second))).To(Succeed())
			Expect(dupont.ReceiveWithFds(nil, 1)).Error().To(MatchError(
				ContainSubstring("i/o timeout")))
		})

		It("skips other control messages", func() {
			dupond, dupont := Successful2R(NewPair())
			defer func() {
				_ = dupond.Close()
				_ = dupont.Close()
			}()

			// Now that is getting into overachiever territory: in order to
			// receive SCM_CREDENTIALS control messages, we must first enable
			// receiption on the particular file descriptor. Since the fd is
			// wrapped deeply inside a unix.UnixConn, this needs a
			// SyscallConn/Control dance; many thanks to
			// https://github.com/golang/go/issues/36293 for charting the lie of
			// the land.
			Expect(Successful(dupont.SyscallConn()).Control(func(fd uintptr) {
				Expect(unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_PASSCRED, 1)).To(Succeed())
			})).To(Succeed())
			go func() {
				defer GinkgoRecover()
				oob := unix.UnixCredentials(&unix.Ucred{
					Pid: int32(os.Getpid()),
					Uid: uint32(os.Getuid()),
					Gid: uint32(os.Getgid()),
				})
				_, noob, err := dupond.WriteMsgUnix(nil, oob, nil)
				Expect(noob).To(Equal(len(oob)))
				Expect(err).NotTo(HaveOccurred())
			}()

			//Expect(dupont.SetReadDeadline(time.Now().Add(1 * time.Second))).To(Succeed())
			_, fds, err := dupont.ReceiveWithFds(nil, 42)
			Expect(err).NotTo(HaveOccurred())
			Expect(fds).To(BeNil())
		})

	})

})
