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
	"runtime"

	"golang.org/x/sys/unix"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/thediveo/caps"
	. "github.com/thediveo/success"
)

var _ = Describe("transient namespaces", Ordered, func() {

	BeforeAll(func() {
		if os.Getuid() != 0 {
			Skip("needs root")
		}
	})

	It("cannot return to its original time namespace when having multiple threads", func() {
		// This test solely exists to document the expected and currently
		// observed Linux kernel behavior where unshare(2)'ing a new time
		// namespace is allowed, but returning to it using setns(2) isn't.
		done := make(chan struct{})
		go func() {
			defer close(done)
			defer GinkgoRecover()

			runtime.LockOSThread()
			origtime := Successful(unix.Open("/proc/thread-self/ns/time", os.O_RDONLY, 0))
			defer func() { _ = unix.Close(origtime) }()
			Expect(unix.Unshare(unix.CLONE_NEWTIME)).To(Succeed())
			Expect(unix.Setns(origtime, unix.CLONE_NEWTIME)).To(
				MatchError(ContainSubstring("too many users")))
		}()
		Eventually(done).Should(BeClosed())
	})

	When("creating and entering in a single step", func() {

		DescribeTable("no entry",
			func(typ int) {
				Expect(InterceptGomegaFailure(func() {
					_ = EnterTransient(typ)
				})).To(MatchError(ContainSubstring("unsupported type " + Name(typ))))
			},
			Entry("mount", unix.CLONE_NEWNS),
			Entry("user", unix.CLONE_NEWUSER),
			Entry("time", unix.CLONE_NEWTIME),
		)

		DescribeTable("enter and leave",
			func(typ int) {
				runtime.LockOSThread()
				defer runtime.UnlockOSThread()

				origIno := Ino(Current(typ), typ)

				cleanup := EnterTransient(typ)
				Expect(cleanup).NotTo(BeNil())
				Expect(Ino(Current(typ), typ)).NotTo(Equal(origIno),
					"failed to enter "+Name(typ))

				cleanup()
				Expect(Ino(Current(typ), typ)).To(Equal(origIno),
					"failed to leave"+Name(typ))
			},
			Entry("cgroup", unix.CLONE_NEWCGROUP),
			Entry("ipc", unix.CLONE_NEWIPC),
			Entry("net", unix.CLONE_NEWNET),
			Entry("uts", unix.CLONE_NEWUTS),
		)

		It("panics when unable to restore the previously attached namespace", func() {
			// To 100% coverage and beyond...!!!

			runtime.LockOSThread() // this thread will be tainted and must be dropped at the end.

			cleanup := EnterTransient(unix.CLONE_NEWNET)
			caps.SetForThisTask(caps.TaskCapabilities{})
			Expect(cleanup).To(PanicWith(
				ContainSubstring("cannot restore original net namespace")))
		})

	})

	When("only creating", func() {

		DescribeTable("rejecting creation",
			func(typ int) {
				Expect(InterceptGomegaFailure(func() {
					_ = NewTransient(typ)
				})).To(MatchError(ContainSubstring("unsupported type " + Name(typ))))
			},
			Entry("mount", unix.CLONE_NEWNS),
			Entry("user", unix.CLONE_NEWUSER),
			Entry("time", unix.CLONE_NEWTIME),
		)

		DescribeTable("successful creation",
			func(typ int) {
				runtime.LockOSThread()
				defer runtime.UnlockOSThread()

				origIno := Ino(Current(typ), typ)
				newns := NewTransient(typ)
				Expect(Ino(Current(typ), typ)).To(Equal(origIno), "didn't switch back")
				Expect(Ino(newns, typ)).NotTo(Equal(origIno), "didn't create new namespace")
			},
			Entry("cgroup", unix.CLONE_NEWCGROUP),
			Entry("ipc", unix.CLONE_NEWIPC),
			Entry("net", unix.CLONE_NEWNET),
			Entry("uts", unix.CLONE_NEWUTS),
		)

	})

})
