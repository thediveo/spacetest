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

package mntns

import (
	"golang.org/x/sys/unix"

	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck // ST1001 rule does not apply
	. "github.com/onsi/gomega"    //nolint:staticcheck // ST1001 rule does not apply
)

// MountSysfsRO mounts a new “sysfs” instance read-only onto “/sys” when the caller
// is in a new and transient mount namespace. Otherwise, MountSysfsRO will fail
// the current test.
func MountSysfsRO() {
	GinkgoHelper()

	// Ensure that we're not still in the process's original mount namespace, as
	// otherwise we would overmount the host's /sysfs.
	Expect(CurrentIno()).NotTo(Equal(Ino("/proc/self/ns/mnt")),
		"current mount namespace must not be the process's original mount namespace")

	Expect(unix.Mount(
		"none", "/sys", "sysfs",
		unix.MS_RDONLY|unix.MS_NODEV|unix.MS_NOEXEC|unix.MS_NOSUID|unix.MS_RELATIME,
		"")).To(Succeed(),
		"cannot mount new sysfs instance on /sys")
}
