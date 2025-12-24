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

var _ = Describe("subspace", func() {

	BeforeEach(func() {
		goodfds := Filedescriptors()
		DeferCleanup(func() {
			Expect(Filedescriptors()).NotTo(HaveLeakedFds(goodfds))
		})
	})

	When("responding to a subspace request", func() {

		It("transfers subspace response fds out-of-band", func() {
			fd1 := Successful(unix.Open(".", unix.O_RDONLY, 0))
			defer func() { _ = unix.Close(fd1) }()
			resp := &SubspaceResponse{
				Conn: fd1,
				Subspaces: Subspaces{
					User: spacetest.Current(unix.CLONE_NEWUSER),
					PID:  spacetest.Current(unix.CLONE_NEWPID),
				},
			}
			fds := resp.EncodeFds()
			Expect(fds).To(HaveLen(3))
			Expect(*resp).To(BeZero())
			resp.DecodeFds(fds)
			Expect(resp.Conn).To(Equal(fd1))
			Expect(spacetest.Type(resp.User)).To(Equal(unix.CLONE_NEWUSER))
			Expect(spacetest.Type(resp.PID)).To(Equal(unix.CLONE_NEWPID))
		})

		It("it drops invalid fds", func() {
			fd1 := Successful(unix.Open(".", unix.O_RDONLY, 0))
			defer func() { _ = unix.Close(fd1) }()
			fd2 := Successful(unix.Open(".", unix.O_RDONLY, 0))
			defer func() { _ = unix.Close(fd2) }()

			var resp SubspaceResponse
			resp.DecodeFds([]int{fd1, fd2})
			Expect(resp.Conn).To(Equal(fd1))
			Expect(resp.Subspaces).To(BeZero())
		})

	})

})
