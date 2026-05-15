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

package bindmount

import (
	"os"
	"runtime"
	"strconv"

	"golang.org/x/sys/unix"

	"github.com/thediveo/spacetest"

	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck // ST1001 rule does not apply
	. "github.com/onsi/gomega"    //nolint:staticcheck // ST1001 rule does not apply
)

const (
	// BindmountPrefix is the prefix for namespace bind mounts inside the
	// default directory for temporary files.
	BindmountPrefix = "bindmount-ns-"
)

// NewTransient creates a new transient bind mount of the namespace referenced
// in “ref” inside the default directory for temporary files, returning the path
// name where the namespace has been bind-mounted at. It automatically cleans up
// the bind mount at the end of the current spec/node. The namespace can be
// referenced either by a file descriptor or a VFS path name.
func NewTransient[R ~int | ~string](ref R) string {
	GinkgoHelper()
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	f, err := os.CreateTemp(os.TempDir(), BindmountPrefix)
	Expect(err).NotTo(HaveOccurred(), "cannot create transient mountpoint")
	mountpoint := f.Name()
	Expect(f.Close()).To(Succeed())
	DeferCleanup(func() {
		Expect(os.Remove(mountpoint)).To(Succeed(), "cannot remove transient mountpoint")
	})

	_ = spacetest.Type(ref) // ensure it's a namespace reference, not something else.
	switch ref := any(ref).(type) {
	case int:
		Expect(unix.Mount(
			"/proc/thread-self/fd/"+strconv.FormatInt(int64(ref), 10),
			mountpoint,
			"", unix.MS_BIND|unix.MS_RDONLY, "")).To(Succeed())
	case string:
		Expect(unix.Mount(
			ref,
			mountpoint,
			"", unix.MS_BIND|unix.MS_RDONLY, "")).To(Succeed())
	}
	DeferCleanup(func() {
		Expect(unix.Unmount(mountpoint, 0)).To(Succeed(), "cannot unmount namespace")
	})
	return mountpoint
}
