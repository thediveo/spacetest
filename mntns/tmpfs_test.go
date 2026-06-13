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

package mntns

import (
	"os"

	"golang.org/x/sys/unix"

	"github.com/thediveo/spacetest/units"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("mounting new tmpfs", func() {

	BeforeEach(func() {
		if os.Getuid() != 0 {
			Skip("needs root")
		}
	})

	It("it mounts a fresh tmpfs", func() {
		var origtmpfsstats unix.Statfs_t
		Expect(unix.Statfs("/tmp", &origtmpfsstats)).To(Succeed())

		// Create a new transient mount namespace, attach to it, and then mount
		// a new tmpfs instance there. As this mount is inside the transient
		// mount namespace this tmpfs is transient too and automatically removed
		// when the mount namespace gets garbage collected.
		mntnsfd, _ := NewTransient()
		Execute(mntnsfd, func() {
			Expect(func() {
				MountTmpfs(64 * units.KiB)
			}).NotTo(Panic())
			var fsstats unix.Statfs_t
			Expect(unix.Statfs("/tmp", &fsstats)).To(Succeed())
			Expect(uint64(fsstats.Blocks) * uint64(fsstats.Bsize)).To(Equal(uint64(64 * units.KiB)))
			Expect(uint64(fsstats.Blocks) * uint64(fsstats.Bsize)).
				NotTo(Equal(uint64(origtmpfsstats.Blocks) * uint64(origtmpfsstats.Bsize)))
		})

		// This test just ensures that the new tmpf was actually mounted inside
		// a transient mount namespace and not in the test programs's mount
		// namespace.
		var fsstats unix.Statfs_t
		Expect(unix.Statfs("/tmp", &fsstats)).To(Succeed())
		Expect(uint64(fsstats.Blocks) * uint64(fsstats.Bsize)).
			To(Equal(uint64(origtmpfsstats.Blocks) * uint64(origtmpfsstats.Bsize)))
	})

})
