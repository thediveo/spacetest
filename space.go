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
	"fmt"
	"runtime"

	"github.com/thediveo/ioctl"
	"golang.org/x/sys/unix"

	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck // ST1001 rule does not apply
	. "github.com/onsi/gomega"    //nolint:staticcheck // ST1001 rule does not apply
)

// Reference is a Linux kernel namespace reference in VFS path textual form or
// as an open file descriptor
type Reference interface{ ~int | ~string }

// Linux kernel [ioctl(2)] command for [namespace relationship queries].
//
// [ioctl(2)]: https://man7.org/linux/man-pages/man2/ioctl.2.html
// [namespace relationship queries]: https://elixir.bootlin.com/linux/v6.2.11/source/include/uapi/linux/nsfs.h
const _NSIO = 0xb7

// Returns the type of namespace CLONE_NEW* value referred to by a file
// descriptor.
var NS_GET_NSTYPE = ioctl.IO(_NSIO, 0x3)

// Avoid problems that would happen when we accidentally unshare the initial
// thread, so we lock it here, thus ensuring that other Go routines (and
// especially tests) won't ever get scheduled onto the initial thread anymore.
func init() {
	runtime.LockOSThread()
}

// Type returns the type constant for the Linux kernel namespace referenced
// either by a file descriptor or a VFS path name.
//
// If the specified reference is invalid, Type fails the current test.
func Type[R Reference](ref R) int {
	GinkgoHelper()

	switch ref := any(ref).(type) {
	case int:
		typ, err := unix.IoctlRetInt(ref, NS_GET_NSTYPE)
		Expect(err).NotTo(HaveOccurred(),
			"cannot determine type of namespace")
		return typ
	case string:
		fd, err := unix.Open(ref, unix.O_RDONLY, 0)
		Expect(err).NotTo(HaveOccurred(),
			"cannot determine type of namespace referenced as %q", ref)
		defer func() { _ = unix.Close(fd) }()
		typ, err := unix.IoctlRetInt(fd, NS_GET_NSTYPE)
		Expect(err).NotTo(HaveOccurred(),
			"cannot determine type of namespace referenced as %q", ref)
		return typ
	}
	return 0 // ST0666 cannot be reached
}

// Ino returns the identification (inode number) of the passed Linux kernel
// namespace that is either referenced by a file descriptor or a VFS path name.
//
// If the specified reference is invalid or doesn't match the passed type of
// namespace, Ino fails the current test.
func Ino[R Reference](ref R, typ int) uint64 {
	GinkgoHelper()

	var namespaceStat unix.Stat_t
	switch ref := any(ref).(type) {
	case int:
		Expect(unix.Fstat(ref, &namespaceStat)).To(Succeed(),
			func() string {
				return fmt.Sprintf("cannot stat %s namespace reference %v", Name(typ), ref)
			})
	case string:
		Expect(unix.Stat(ref, &namespaceStat)).To(Succeed(),
			func() string {
				return fmt.Sprintf("cannot stat %s namespace reference %v", Name(typ), ref)
			})
	}
	Expect(Type(ref)).To(Equal(typ),
		"not a %s namespace", Name(typ))
	return namespaceStat.Ino
}

// Current returns a file descriptor referencing the calling OS-level thread's
// current namespace of type “typ”. Please note that the caller's go routine
// should be thread-locked. “typ” should be any of [unix.CLONE_NEWNS] (for mount
// namespaces), [unix.CLONE_NETNS] (network), et cetera.
//
// Additionally, Current schedules a DeferCleanup of the returned file
// descriptor to be closed at the end of the current test in order to avoid
// leaking it.
//
// If the specified typ of namespace is unknown, Current fails the current test.
func Current(typ int) int {
	GinkgoHelper()

	typename := Name(typ)
	Expect(typename).NotTo(BeEmpty(),
		"unknown type of namespace %d", typ)
	nsfd, err := unix.Open("/proc/thread-self/ns/"+typename, unix.O_RDONLY, 0)
	Expect(err).NotTo(HaveOccurred(),
		"cannot determine current %s namespace from procfs", typename)
	DeferCleanup(func() {
		_ = unix.Close(nsfd)
	})
	return nsfd
}

// CurrentIno returns the identification (inode number) for the namespace (of
// the specified type) the OS-level thread is currently attached to.
func CurrentIno(typ int) uint64 {
	GinkgoHelper()

	return Ino("/proc/thread-self/ns/"+Name(typ), typ)
}
