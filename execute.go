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
	"slices"

	"golang.org/x/sys/unix"

	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck // ST1001 rule does not apply
	. "github.com/onsi/gomega"    //nolint:staticcheck // ST1001 rule does not apply
)

// Execute the passed fn synchronously while attached to the specified
// namespace(s) and otherwise defaulting to the caller's currently attached
// namespaces.
//
// Execute will fail the current test when trying to switch to a different user
// namespace: switching the user namespace is not possible for multi-threaded
// processes, this is a design decision of the Linux kernel user namespace
// developers.
//
// When a mount namespace is passed in, then fn will be executed on a separate
// throw-away go routine (and locked to a throw-away OS-level thread). Where the
// caller does not specify different namespace(s) the underlying thread will be
// attached to the caller's namespaces. This ensures the expected behavior when
// using Execute after especially [EnterTransient].
//
// When the list of namespaces to switch to does not contain a mount namespace
// then the passed fn will be called synchronously on the caller's go routine,
// while locked to the underlying OS-level thread.
func Execute(fn func(), nsfd int, nsfds ...int) {
	GinkgoHelper()

	var mntnsfd = int(-1)
	var othernsfds []int

	for _, nsfd := range append([]int{nsfd}, nsfds...) {
		switch Type(nsfd) {
		case unix.CLONE_NEWUSER:
			Expect("user").NotTo(Equal("user"), "cannot Execute() in different user namespace")
		case unix.CLONE_NEWNS:
			mntnsfd = nsfd
		default:
			othernsfds = append(othernsfds, nsfd)
		}
	}

	if mntnsfd >= 0 {
		goSeparate(fn, mntnsfd, othernsfds...)
		return
	}
	goInAndOut(fn, othernsfds...)
}

// goInAndOut runs the passed fn on the current go routine and locked to its
// OS-level thread, temporarily switching into the specified namespaces while fn
// runs.
//
// If anything fails, this will automatically fail the current test.
func goInAndOut(fn func(), othernsfds ...int) {
	runtime.LockOSThread()

	var callersNamespaces []int
	defer func() {
		// In case we came here because switching into the specified namespaces
		// failed, then we silently try to restore things as good as possible
		// (which is questionable) and then re-panic. We never unlock the
		// OS-level thread from its go routine.
		if r := recover(); r != nil {
			for _, nsfd := range slices.Backward(callersNamespaces) {
				_ = unix.Setns(nsfd, 0)
			}
			panic(r)
		}
		// Try restoring the namespaces that the caller was attached to before
		// we temporarily switched into different ones.
		for _, nsfd := range slices.Backward(callersNamespaces) {
			Expect(unix.Setns(nsfd, 0)).To(Succeed(),
				func() string {
					return fmt.Sprintf("cannot restore %s namespace", Name(Type(nsfd)))
				})
		}
		// Only unlock OS-level thread from go routine if we were successful in
		// restoring all changed namespaces.
		runtime.UnlockOSThread()
	}()

	// Attach to the specified namespaces and fail if this doesn't work.
	for _, nsfd := range othernsfds {
		typ := Type(nsfd)
		callersNamespaces = append(callersNamespaces, Current(typ))
		Expect(unix.Setns(nsfd, typ)).To(Succeed(),
			func() string {
				return fmt.Sprintf("cannot switch into %s namespace", Name(typ))
			})
	}

	fn()
}

// goSeparate runs the passed fn on a separate go routine, locked to its
// OS-level thread, and with its own filesystem attributes (when mntnsfd >= 0).
// This allows the locked thread to attach to a mount namespace different from
// the mount namespace of our process and other threads.
//
// Moreover, the passed fn will run attached on its separate OS-level thread to
// the passed namespaces references in othernsfds. Additionally, this separate
// thread will be attached to the namespaces of the caller thread that haven't
// been explicitly overrriden/passed in othernsfds. This ensures that fn is
// executed in the same namespace configuration as the caller.
//
// If anything fails, goSeparate will automatically fail the current test.
func goSeparate(fn func(), mntnsfd int, othernsfds ...int) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// Find out which of the non-user and non-mount namespaces aren't explicitly
	// set and which we therefore need to take over from the caller; the
	// caller's OS-level thread might have partically differing namespaces
	// configured compared to a fresh or reused "untainted" OS-level thread.
	pickupTypes := []int{
		unix.CLONE_NEWCGROUP,
		unix.CLONE_NEWIPC,
		unix.CLONE_NEWNET,
		unix.CLONE_NEWPID,
		unix.CLONE_NEWTIME,
		unix.CLONE_NEWUTS,
	}
	for _, nsfd := range othernsfds {
		typ := Type(nsfd)
		pickupTypes = slices.DeleteFunc(pickupTypes, func(e int) bool { return e == typ })
	}
	var pickupfds []int
	for _, typ := range pickupTypes {
		pickupfds = append(pickupfds, Current(typ))
	}

	panicCh := make(chan any)
	go func() {
		// We cannot really do a sensible "defer GinkgoRecover()" here, so
		// instead we're catching any panics and rethrow them on the caller's go
		// routine.
		defer func() {
			// Always properly close the namespace reference fds that we had
			// opened in order to ensure that the new transient thread is
			// attached to the same namespaces the caller is attached to, except
			// for those explicitly overridden.
			for _, nsfd := range pickupfds {
				_ = unix.Close(nsfd)
			}
			if r := recover(); r != nil {
				panicCh <- r
			}
			close(panicCh)
		}()

		runtime.LockOSThread()

		if mntnsfd >= 0 {
			Expect(unix.Unshare(unix.CLONE_FS)).To(Succeed(),
				"cannot unshare file attributes of transient func call OS-level thread")
			Expect(unix.Setns(mntnsfd, unix.CLONE_NEWNS)).To(Succeed(),
				"cannot switch into mnt namespace")
		}

		// now attach our separate thread to the same namespaces as the caller's
		// thread was attached to, except where the caller told us different
		// namespaces.
		for _, nsfd := range append(othernsfds, pickupfds...) {
			typ := Type(nsfd)
			name := Name(typ)
			if Ino(nsfd, typ) == Ino("/proc/thread-self/ns/"+name, typ) {
				// skip unnecessary namespace switching from the one namespace
				// into the same, as these may fail and thus cause us otherwise
				// unwanted false positives.
				continue
			}
			Expect(unix.Setns(nsfd, 0)).To(Succeed(),
				"cannot switch into %s namespace", name)
		}

		// setup is finally complete, call the passed fn and then let's be done
		// with it...
		fn()
	}()

	// receive panic, if any, and rethrow it on the caller's go routine.
	if r := <-panicCh; r != nil {
		panic(r)
	}
}
