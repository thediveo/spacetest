// Copyright 2026 Harald Albrecht.
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
	"os"

	"golang.org/x/sys/unix"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/thediveo/fdooze"
	. "github.com/thediveo/success"
)

var _ = Describe("PIDs from PID fds", func() {

	BeforeEach(func() {
		goodfds := Filedescriptors()
		DeferCleanup(func() {
			Expect(Filedescriptors()).NotTo(HaveLeakedFds(goodfds))
		})
	})

	It("rejects non-existing fd", func() {
		Expect(PIDfromPIDFd(666_666)).Error().To(HaveOccurred())
	})

	It("rejects non-PID fd", func() {
		thisaintapidfd := Successful(unix.Open(".", unix.O_RDONLY, 0))
		defer func() { _ = unix.Close(thisaintapidfd) }()
		Expect(PIDfromPIDFd(thisaintapidfd)).Error().To(MatchError(ContainSubstring("is not a PID fd")))
	})

	It("returns the correct PID", func() {
		mypidfd := Successful(unix.PidfdOpen(os.Getpid(), 0))
		defer func() { _ = unix.Close(mypidfd) }()
		Expect(PIDfromPIDFd(mypidfd)).To(Equal(os.Getpid()))
	})

})
