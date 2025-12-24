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
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gleak"
	. "github.com/thediveo/fdooze"
	"golang.org/x/sys/unix"
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
			cl := New(ctx)
			defer cl.Close()

			subcl, spc := cl.Subspace(true, true)

			Expect(subcl).NotTo(BeNil())
			Expect(spc.PID).To(BeNumerically(">", 0))
			Expect(spc.User).To(BeNumerically(">", 0))
		})

		It("starts a spacer and creates namespaces", func(ctx context.Context) {
			cl := New(ctx)
			defer cl.Close()

			rooms := cl.Rooms(true, true, true, true, true, true)

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
				cl := New(ctx)
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

})
