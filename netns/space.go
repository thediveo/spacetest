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

package netns

import (
	"github.com/thediveo/spacetest"
	"golang.org/x/sys/unix"

	gi "github.com/onsi/ginkgo/v2"
)

// Current returns a file descriptor referencing the calling OS-level thread's
// current network namespace. Please note that the caller's go routine should be
// thread-locked ([runtime.LockOSThread]).
//
// Additionally, Current schedules a [gi.DeferCleanup] of the returned file
// descriptor to be closed at the end of the current test in order to avoid
// leaking it.
func Current() int {
	gi.GinkgoHelper()

	return spacetest.Current(unix.CLONE_NEWNET)
}

// CurrentIno returns the identification of the network namespace in form of a
// inode number for the current OS-level thread.
func CurrentIno() uint64 {
	gi.GinkgoHelper()

	return Ino("/proc/thread-self/ns/net")
}

// Ino returns the identification (in form of an inode number) of the passed
// network namespace, either referenced by a file descriptor or a VFS path name.
//
// If the specified reference is invalid, Ino fails the current test.
func Ino[R ~int | ~string](netns R) uint64 {
	gi.GinkgoHelper()

	return spacetest.Ino(netns, unix.CLONE_NEWNET)
}
