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
	"runtime"

	"golang.org/x/sys/unix"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/thediveo/success"
)

const minNamespaceIno = 0xf000000

var _ = Describe("retrieving properties of name spaces", func() {

	When("determining the type of namespace", func() {

		It("accepts a VFS path", func() {
			Expect(Type("/proc/self/ns/mnt")).To(Equal(unix.CLONE_NEWNS))
		})

		It("rejects an invalid VFS path", func() {
			Expect(InterceptGomegaFailure(func() {
				_ = Type("/proc/me,myself,I")
			})).To(MatchError(ContainSubstring("cannot determine type of namespace referenced as")))
		})

		It("accepts an open file descriptor", func() {
			fd := Successful(unix.Open("/proc/thread-self/ns/net", unix.O_RDONLY, 0))
			defer func() { _ = unix.Close(fd) }()
			Expect(Type(fd)).To(Equal(unix.CLONE_NEWNET))
		})

		It("rejects an invalid file descriptor", func() {
			Expect(InterceptGomegaFailure(func() {
				_ = Type(0)
			})).To(MatchError(ContainSubstring("cannot determine type of namespace")))
		})

	})

	When("determining the id/inode no of a namespace", func() {

		It("accepts a VFS path", func() {
			Expect(Ino("/proc/self/ns/mnt", unix.CLONE_NEWNS)).To(
				BeNumerically(">=", minNamespaceIno))
		})

		It("rejects an invalid VFS path", func() {
			Expect(InterceptGomegaFailure(func() {
				_ = Ino("/proc/me,myself,I", unix.CLONE_NEWPID)
			})).To(MatchError(ContainSubstring("cannot stat pid namespace reference")))
		})

		It("rejects the wrong type of namespace", func() {
			Expect(InterceptGomegaFailure(func() {
				_ = Ino("/proc/self/ns/mnt", unix.CLONE_NEWUTS)
			})).To(MatchError(ContainSubstring("not a uts namespace")))
		})

		It("accepts an open file descriptor", func() {
			fd := Successful(unix.Open("/proc/thread-self/ns/net", unix.O_RDONLY, 0))
			defer func() { _ = unix.Close(fd) }()
			Expect(Ino(fd, unix.CLONE_NEWNET)).To(
				BeNumerically(">=", minNamespaceIno))
		})

		It("rejects an invalid file descriptor", func() {
			Expect(InterceptGomegaFailure(func() {
				_ = Ino(-1, unix.CLONE_NEWPID)
			})).To(MatchError(ContainSubstring("cannot stat pid namespace reference -1")))
		})
	})

	It("determines the current namespaces", func() {
		for _, typ := range []int{
			unix.CLONE_NEWCGROUP,
			unix.CLONE_NEWIPC,
			unix.CLONE_NEWNS,
			unix.CLONE_NEWNET,
			unix.CLONE_NEWPID,
			unix.CLONE_NEWTIME,
			unix.CLONE_NEWUSER,
			unix.CLONE_NEWUTS,
		} {
			runtime.LockOSThread()
			fd := Current(typ) // n.b. schedules automatic close of returned file descriptor
			runtime.UnlockOSThread()
			Expect(fd).NotTo(BeZero())
			Expect(Type(fd)).To(Equal(typ))
		}
	})

	It("returns the correct current inode number", func() {
		netnsIno := CurrentIno(unix.CLONE_NEWNET)
		Expect(netnsIno).NotTo(BeZero())
		Expect(netnsIno).To(Equal(CurrentIno(unix.CLONE_NEWNET)))
	})

})
