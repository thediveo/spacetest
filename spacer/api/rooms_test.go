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

package api

import (
	"github.com/thediveo/spacetest"
	"golang.org/x/sys/unix"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/thediveo/fdooze"
	. "github.com/thediveo/success"
)

var _ = Describe("making space", func() {

	BeforeEach(func() {
		goodfds := Filedescriptors()
		DeferCleanup(func() {
			Expect(Filedescriptors()).NotTo(HaveLeakedFds(goodfds))
		})
	})

	When("responding to a make request", func() {

		It("transfers make response fds out-of-band", func() {
			resp := &RoomsResponse{
				Cgroup: spacetest.Current(unix.CLONE_NEWCGROUP),
				IPC:    spacetest.Current(unix.CLONE_NEWIPC),
				Mnt:    spacetest.Current(unix.CLONE_NEWNS),
				Net:    spacetest.Current(unix.CLONE_NEWNET),
				Time:   spacetest.Current(unix.CLONE_NEWTIME),
				UTS:    spacetest.Current(unix.CLONE_NEWUTS),
			}
			fds := resp.EncodeFds()
			Expect(fds).To(HaveLen(6))
			Expect(*resp).To(BeZero())
			resp.DecodeFds(fds)
			Expect(spacetest.Type(resp.Cgroup)).To(Equal(unix.CLONE_NEWCGROUP))
			Expect(spacetest.Type(resp.IPC)).To(Equal(unix.CLONE_NEWIPC))
			Expect(spacetest.Type(resp.IPC)).To(Equal(unix.CLONE_NEWIPC))
			Expect(spacetest.Type(resp.Mnt)).To(Equal(unix.CLONE_NEWNS))
			Expect(spacetest.Type(resp.Net)).To(Equal(unix.CLONE_NEWNET))
			Expect(spacetest.Type(resp.Time)).To(Equal(unix.CLONE_NEWTIME))
			Expect(spacetest.Type(resp.UTS)).To(Equal(unix.CLONE_NEWUTS))
		})

		It("it drops invalid fds", func() {
			fd1 := Successful(unix.Open(".", unix.O_RDONLY, 0))
			defer func() { _ = unix.Close(fd1) }()

			var resp RoomsResponse
			resp.DecodeFds([]int{fd1, spacetest.Current(unix.CLONE_NEWNET)})
			Expect(resp.Cgroup).To(BeZero())
			Expect(resp.IPC).To(BeZero())
			Expect(resp.Mnt).To(BeZero())
			Expect(resp.Net).NotTo(BeZero())
			Expect(resp.Time).To(BeZero())
			Expect(resp.UTS).To(BeZero())
		})

	})

})
