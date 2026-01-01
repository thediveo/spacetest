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

package spacer

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/thediveo/ioctl"
	"github.com/thediveo/safe"
	"github.com/thediveo/spacetest"
	"golang.org/x/sys/unix"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gleak"
	. "github.com/thediveo/fdooze"
	. "github.com/thediveo/success"
)

var _ = Describe("spacer client", func() {

	When("working with the spacer service", func() {

		BeforeEach(func() {
			if os.Getuid() != 0 {
				Skip("needs root")
			}

			goodfds := Filedescriptors()
			goodgos := Goroutines()
			DeferCleanup(func() {
				Eventually(Goroutines).Within(2 * time.Second).ProbeEvery(100 * time.Millisecond).
					ShouldNot(HaveLeaked(goodgos))
				Expect(Filedescriptors()).NotTo(HaveLeakedFds(goodfds))
			})
		})

		It("starts a spacer and creates a subspace", func(ctx context.Context) {
			var out safe.Buffer
			w := io.MultiWriter(&out, GinkgoWriter)
			cl := New(ctx, WithOut(w), WithErr(w))
			defer cl.Close()

			subcl, spc := cl.Subspace(true, true)

			Expect(subcl).NotTo(BeNil())
			Expect(spc.PID).To(BeNumerically(">", 0))
			Expect(spc.User).To(BeNumerically(">", 0))
		})

		It("starts a spacer and creates namespaces", func(ctx context.Context) {
			var out safe.Buffer
			w := io.MultiWriter(&out, GinkgoWriter)
			cl := New(ctx, WithOut(w), WithErr(w))
			defer cl.Close()

			rooms := cl.Rooms(true, true, true, true, true, true)
			Eventually(out.String()).Within(2 * time.Second).To(
				MatchRegexp(`"serving request" .* service=\*api.RoomsRequest`))

			Expect(rooms.Cgroup).To(BeNumerically(">", 0))
			Expect(rooms.IPC).To(BeNumerically(">", 0))
			Expect(rooms.Mnt).To(BeNumerically(">", 0))
			Expect(rooms.Net).To(BeNumerically(">", 0))
			Expect(rooms.Time).To(BeNumerically(">", 0))
			Expect(rooms.UTS).To(BeNumerically(">", 0))

			subcl, _ := cl.Subspace(true, true)

			rooms = subcl.Rooms(true, true, true, true, true, true)

			Expect(rooms.Cgroup).To(BeNumerically(">", 0))
			Expect(rooms.IPC).To(BeNumerically(">", 0))
			Expect(rooms.Mnt).To(BeNumerically(">", 0))
			Expect(rooms.Net).To(BeNumerically(">", 0))
			Expect(rooms.Time).To(BeNumerically(">", 0))
			Expect(rooms.UTS).To(BeNumerically(">", 0))
		})

		DescribeTable("creating transient namespaces",
			func(ctx context.Context, typ int) {
				var out safe.Buffer
				w := io.MultiWriter(&out, GinkgoWriter)
				cl := New(ctx, WithOut(w), WithErr(w))
				defer cl.Close()
				nsfd := cl.NewTransient(typ)
				Expect(nsfd).To(BeNumerically(">", 0))
			},
			Entry("cgroup", unix.CLONE_NEWCGROUP),
			Entry("ipc", unix.CLONE_NEWIPC),
			Entry("mnt", unix.CLONE_NEWNS),
			Entry("net", unix.CLONE_NEWNET),
			Entry("time", unix.CLONE_NEWTIME),
			Entry("uts", unix.CLONE_NEWUTS),
		)

	})

	It("augments namespace clone flags", func() {
		flags := namespaces(0).ifrequested(true, unix.CLONE_NEWNET)
		Expect(flags).To(Equal(namespaces(unix.CLONE_NEWNET)))
		flags = flags.ifrequested(false, unix.CLONE_NEWUTS)
		Expect(flags).To(Equal(namespaces(unix.CLONE_NEWNET)))
		flags = flags.ifrequested(true, unix.CLONE_NEWIPC)
		Expect(flags).To(Equal(namespaces(unix.CLONE_NEWNET | unix.CLONE_NEWIPC)))
	})

	// The following unit test sets up a network namespace in a child user
	// namespace via a subspacer process, but then this subspacer process drops
	// the reference to the network namespace as it passes the referencing fd to
	// the client, our unit tests process. This test establishes that the
	// network namespace is in fact still owned by the child user namespace and
	// this can be seen from out perspective of the (parent) unit tests process.
	//
	// Note: this test does not actually test the spacer but instead the
	// expected Linux kernel behavior; being kind of a meta test.
	Context("namespace madness", func() {

		var childusernsfd, netnsfd int

		BeforeEach(func() {
			if os.Getuid() != 0 {
				Skip("needs root")
			}

			By("creating a primary spacer client")
			ctx, cancel := context.WithCancel(context.Background())
			clnt := New(ctx, WithErr(GinkgoWriter))
			DeferCleanup(func() {
				By("cancelling context and closing primary spacer client")
				cancel()
				clnt.Close()
			})

			By("creating a child user namespace")
			subclnt, spc := clnt.Subspace(true, false)
			DeferCleanup(func() {
				subclnt.Close()
			})
			childusernsfd = spc.User

			By("creating a network namespace belonging to the child user namespace")
			netnsfd = subclnt.NewTransient(unix.CLONE_NEWNET)
			DeferCleanup(func() {
				Expect(unix.Close(netnsfd)).To(Succeed())
			})
		})

		It("correctly creates a network namespace owned by a child user namespace", func() {
			Expect(spacetest.Type(netnsfd)).To(Equal(unix.CLONE_NEWNET))
			ownerfd := Successful(ioctl.RetFd(netnsfd, NS_GET_USERNS))
			defer func() {
				Expect(unix.Close(ownerfd)).To(Succeed())
			}()
			Expect(spacetest.Ino(ownerfd, unix.CLONE_NEWUSER)).To(
				Equal(spacetest.Ino(childusernsfd, unix.CLONE_NEWUSER)))
			parentfd := Successful(ioctl.RetFd(ownerfd, NS_GET_PARENT))
			defer func() {
				Expect(unix.Close(parentfd)).To(Succeed())
			}()
			Expect(spacetest.Ino(parentfd, unix.CLONE_NEWUSER)).To(
				Equal(spacetest.CurrentIno(unix.CLONE_NEWUSER)))
		})

	})

})

// Linux kernel [ioctl(2)] command for [namespace relationship queries].
//
// [ioctl(2)]: https://man7.org/linux/man-pages/man2/ioctl.2.html
// [namespace relationship queries]: https://elixir.bootlin.com/linux/v6.2.11/source/include/uapi/linux/nsfs.h
const _NSIO = 0xb7

var (
	NS_GET_USERNS = ioctl.IO(_NSIO, 0x1)
	NS_GET_PARENT = ioctl.IO(_NSIO, 0x2)
)
