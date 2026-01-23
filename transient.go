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

	"golang.org/x/sys/unix"

	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck // ST1001 rule does not apply
	. "github.com/onsi/gomega"    //nolint:staticcheck // ST1001 rule does not apply
)

// EnterTransient creates and enters a new (and isolated) Linux kernel namespace
// of the specified type, returning a function that needs to be defer'ed in
// order to correctly switch the calling go routine and its locked OS-level
// thread back when the caller itself wants to leave (returns). For instance:
//
//	defer spacetest.EnterTransient(unix.CLONE_NEWNET)()
//
// EnterTransient locks the caller's go routine to its OS-level thread and
// unlocks it when the deferred clean up function finally gets called.
//
// In case the caller cannot be switched back correctly, the defer'ed cleanup
// function will panic with an error description detailing the reason.
//
// EnterTransient can be used for the following types of namespaces:
//   - unix.CLONE_NEWCGROUP,
//   - unix.CLONE_NEWIPC,
//   - unix.CLONE_NEWNET,
//   - unix.CLONE_NEWUTS.
//
// For mount namespaces (unix.CLONE_NEWNS) you will need to use the mount
// namespace-specific [github.com/thediveo/spacetest/mntns.EnterTransient]
// instead.
//
// In order to work with transient PID (unix.CLONE_NEWPID) namespaces use
// [NewTransient] and then [Execute], as it is not possible to re-associate the
// current OS-level thread with the original (parent) PID namespace after
// creating and switching into a new child PID namespace; the returned cleanup
// function would fail and purposely trigger a panic.
//
// Also, user namespaces cannot be entered with EnterTransient as the Linux
// kernel does not allow a thread to re-enter one of the original (that is,
// parent) user namespace(s). Use [Execute] instead to call a specific function
// synchronously from a transient go routine with its own transient OS-level
// thread.
//
// To work with transient time (unix.CLONE_NEWTIME) namespaces use
// [NewTransient] and then [Execute], as it is not possible to re-associate the
// current OS-level thread with the original (parent) PID or time namespace
// after creating and switching into a new child PID or time namespace; the
// returned cleanup function would fail and purposely trigger a panic.
func EnterTransient(typ int) func() {
	GinkgoHelper()

	name := Name(typ)
	Expect(typ).To(BeElementOf([]int{
		unix.CLONE_NEWCGROUP,
		unix.CLONE_NEWIPC,
		unix.CLONE_NEWNET,
		unix.CLONE_NEWPID,
		unix.CLONE_NEWUTS,
	}), "unsupported type %s", name)

	runtime.LockOSThread()

	callersNamespace, err := unix.Open("/proc/thread-self/ns/"+name, unix.O_RDONLY, 0)
	Expect(err).NotTo(HaveOccurred(),
		"cannot determine current %s namespace from procfs", name)
	Expect(unix.Unshare(typ)).To(Succeed(),
		"cannot create new %s namespace", Name(typ))

	// Our cleanup cannot be DeferCleanup'ed, because we need to restore the current
	// locked go routine, so that the defer rollback sequence is kept correct.
	return func() {
		if err := unix.Setns(callersNamespace, typ); err != nil {
			panic(fmt.Sprintf("leaving from EnterTransient: cannot restore original %s namespace, reason: %s", name, err.Error()))
		}
		_ = unix.Close(callersNamespace)
		runtime.UnlockOSThread()
	}
}

// NewTransient creates a new Linux kernel namespace of the specified type, but
// doesn't enter it. Instead, it returns a file descriptor referencing the newly
// created namespace. NewTransient can be used for the following types of
// namespaces:
//   - unix.CLONE_NEWCGROUP,
//   - unix.CLONE_NEWIPC,
//   - unix.CLONE_NEWNET,
//   - unix.CLONE_NEWUTS.
//
// For mount namespaces (unix.CLONE_NEWNS) you will need to use the mount
// namespace-specific [github.com/thediveo/spacetest/mntns.NewTransient]
// instead.
//
// Additionally to creating a new namespace, NewTransient also schedules a
// Ginkgo deferred cleanup in order to close the fd referencing this new
// namespace. The caller thus must not close the file descriptor returned.
//
// When NewTransient returns, the caller's Go routine is in the same OS-level
// thread lock/unlock state as before the call.
func NewTransient(typ int) int {
	GinkgoHelper()

	name := Name(typ)
	Expect(typ).To(BeElementOf([]int{
		unix.CLONE_NEWCGROUP,
		unix.CLONE_NEWIPC,
		unix.CLONE_NEWNET,
		unix.CLONE_NEWUTS,
	}), "unsupported type %s", name)

	// if anything below breaks we won't unlock the OS-level thread on purpose
	// so that it gets thrown away as the unit test fails and unwinds.
	runtime.LockOSThread()

	// As Linux only allows us to create a new namespace in combination
	// immediately entering it, we first need to (literally!) get hold on our
	// current namespace of the specified type, so we can later re-attach our
	// OS-level thread to it again.
	callersNamespace, err := unix.Open("/proc/thread-self/ns/"+name, unix.O_RDONLY, 0)
	Expect(err).NotTo(HaveOccurred(),
		"cannot determine current %s namespace from procfs", name)
	defer func() {
		// make sure to always close the fd to the original namespace as to not
		// leak it.
		_ = unix.Close(callersNamespace)
	}()

	Expect(unix.Unshare(typ)).To(Succeed(),
		"cannot create new %s namespace", name)
	newNamespace, err := unix.Open("/proc/thread-self/ns/"+name, unix.O_RDONLY, 0)
	Expect(err).NotTo(HaveOccurred(),
		"cannot determine new %s namespace from procfs", name)
	Expect(unix.Setns(callersNamespace, typ)).To(Succeed(),
		"cannot switch back into original %s namespace", name)
	DeferCleanup(func() { _ = unix.Close(newNamespace) })

	runtime.UnlockOSThread()
	return newNamespace
}
