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
	"fmt"
	"runtime"

	"golang.org/x/sys/unix"

	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck // ST1001 rule does not apply
	. "github.com/onsi/gomega"    //nolint:staticcheck // ST1001 rule does not apply
)

// EnterTransient creates and enters a new mount namespace, returning a function
// that needs to be defer'ed. It additionally remounts “/” in this new mount
// namespace to set propagation of mount points to “private” to prevent mount
// point changes to propagate back into the host.
//
// Note: the current OS-level thread won't be unlocked when the calling unit
// test returns, as we cannot undo unsharing filesystem attributes (using
// CLONE_FS) such as the root directory, current directory, and umask
// attributes.
//
// # Background
//
// [unshare(1)] defaults the mount point propagation to "MS_REC | MS_PRIVATE",
// see [util-linux/unshare.c UNSHARE_PROPAGATION_DEFAULT].
//
// [unshare(1)] is remounting "/" in order to apply its propagation defaults --
// that are to not FUBAR the mount points in the mount namespace we got our
// mount points from during unsharing the mount namespace, see
// [util-linux/unshare.c set_propagation].
//
//	/* C */ mount("none", "/", NULL, flags, NULL
//
// [util-linux/unshare.c set_propagation]: https://github.com/util-linux/util-linux/blob/86b6684e7a215a0608bd130371bd7b3faae67aca/sys-utils/unshare.c#L160
// [unshare(1)]: https://man7.org/linux/man-pages/man1/unshare.1.html
// [util-linux/unshare.c UNSHARE_PROPAGATION_DEFAULT]: https://github.com/util-linux/util-linux/blob/86b6684e7a215a0608bd130371bd7b3faae67aca/sys-utils/unshare.c#L57
func EnterTransient() func() {
	GinkgoHelper()

	runtime.LockOSThread() // ...kind of point of no return

	callersMountNamespace, err := unix.Open("/proc/thread-self/ns/mnt", unix.O_RDONLY, 0)
	Expect(err).NotTo(HaveOccurred(), "cannot determine current mount namespace from procfs")

	// Decouple some filesystem-related attributes of this thread from the ones
	// of our process...
	Expect(unix.Unshare(unix.CLONE_FS|unix.CLONE_NEWNS)).To(Succeed(),
		"cannot create new mount namespace")
	// Remount root to ensure that later mount point manipulations do not
	// propagate back into our host, trashing it.
	Expect(unix.Mount("none", "/", "/", unix.MS_REC|unix.MS_PRIVATE, "")).To(Succeed(),
		"cannot change / mount propagation to private")

	// Our cleanup cannot be DeferCleanup'ed, because we need to restore the current
	// locked go routine, so that the defer rollback sequence is kept correct.
	return func() {
		if err := unix.Setns(callersMountNamespace, 0); err != nil {
			panic(fmt.Sprintf("cannot restore original mount namespace, reason: %s", err.Error()))
		}
		_ = unix.Close(callersMountNamespace)
		// do NOT unlock the OS-level thread, as we cannot undo unsharing CLONE_FS
	}
}

// NewTransient creates a new transient mount namespace that is kept alive by a
// an idle OS-level thread; this idle thread is automatically terminated upon
// returning from the current test.
func NewTransient() (mntfd int, procfsroot string) {
	GinkgoHelper()

	// closing the done channel tells the Go routine we will kick off next to
	// call it a day and terminate (well, unless the called fn is stuck).
	done := make(chan struct{})
	DeferCleanup(func() { close(done) })

	// Kick off a separate Go routine which we then can lock to its OS-level
	// thread and later dispose off because it is tainted due to unsharing the
	// sharing of file attributes.
	readyCh := make(chan idlerDetails)
	go func() {
		defer GinkgoRecover()
		runtime.LockOSThread()

		// Whatever is going to happen to us, make sure to unblock the receiving
		// Go routine, and even if this is the zero value...
		defer func() {
			close(readyCh)
		}()

		// Decouple some filesystem-related attributes of this thread from the ones
		// of our process...
		Expect(unix.Unshare(unix.CLONE_FS|unix.CLONE_NEWNS)).To(Succeed(),
			"cannot create new mount namespace")
		// Remount root to ensure that later mount point manipulations do not
		// propagate back into our host, trashing it.
		Expect(unix.Mount("none", "/", "/", unix.MS_REC|unix.MS_PRIVATE, "")).To(
			Succeed(), "cannot change / mount propagation to private")

		readyCh <- idlerDetails{
			mntnsfd: Current(),
			TID:     unix.Gettid(),
		}

		<-done // ...idle around, then fall off the discworld...
	}()
	idlerInfo := <-readyCh
	Expect(idlerInfo.mntnsfd).NotTo(BeZero())
	procfsroot = fmt.Sprintf("/proc/%d/root", idlerInfo.TID)
	return idlerInfo.mntnsfd, procfsroot
}

// idlerDetails passes information about an idler's TID and mount namespace
// reference from the idler go routine to its creator.
type idlerDetails struct {
	mntnsfd int
	TID     int
}
