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
	"fmt"

	"golang.org/x/sys/unix"

	"github.com/thediveo/spacetest/units"

	gi "github.com/onsi/ginkgo/v2"
	g "github.com/onsi/gomega"
)

// MountTmpfs mounts a new tmpfs instance at /tmp with the specified capacity.
// See [MountTmpfsOn] for further details.
func MountTmpfs(size units.Capacity) { MountTmpfsOn("/tmp", size) }

// MountTmpfsOn mounts a new tmpfs instance at the specified path with the
// specified capacity. The size is in bytes. The tmpfs is mounted with
// MS_RELATIME flags and “inode64” option.
func MountTmpfsOn(target string, size units.Capacity) {
	gi.GinkgoHelper()

	g.Expect(unix.Mount(
		"tmpfs",
		target,
		"tmpfs",
		unix.MS_RELATIME,
		fmt.Sprintf("size=%d,inode64", size),
	)).To(g.Succeed(),
		"cannot mount new tmpfs on %s", target)
}
