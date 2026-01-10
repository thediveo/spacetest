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

package service

import (
	"cmp"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"runtime"
	"syscall"

	"github.com/thediveo/spacetest"
	"github.com/thediveo/spacetest/spacer/api"
	"github.com/thediveo/spacetest/uds"
	"golang.org/x/sys/unix"
)

const validSpaces = unix.CLONE_NEWCGROUP |
	unix.CLONE_NEWIPC |
	unix.CLONE_NEWNS |
	unix.CLONE_NEWNET |
	unix.CLONE_NEWTIME |
	unix.CLONE_NEWUTS

type Spacemaker struct {
	Exe    string
	Stdout io.Writer
	Stderr io.Writer
	log    *slog.Logger
}

func (s *Spacemaker) Slog() *slog.Logger {
	if s.log != nil {
		return s.log
	}
	s.log = slog.New(slog.NewTextHandler(
		cmp.Or(s.Stderr, io.Writer(os.Stderr)),
		&slog.HandlerOptions{Level: slog.LevelInfo}))
	return s.log
}

var _ Spacer = (*Spacemaker)(nil)

// Subspace creates either a new user or PID namespace, or both, and returns
// open file descriptors referencing them; additionally, it returns a file
// descriptor for a unix domain socket that is connected to a child Spacemaker
// service process.
func (s *Spacemaker) Subspace(req *api.SubspaceRequest) api.Response {
	if req.Spaces & ^uint64(unix.CLONE_NEWUSER|unix.CLONE_NEWPID) != 0 {
		return &api.ErrorResponse{Reason: "out of space"}
	}
	if req.Spaces&uint64(unix.CLONE_NEWUSER|unix.CLONE_NEWPID) == 0 {
		return &api.ErrorResponse{Reason: "no space requested"}
	}

	// We start by creating a pair of connected unix domain sockets: one we'll
	// pass to the service we'll soon start, the other we'll pass back in our
	// response. This then allows the requester to directly talk to the newly
	// started sub service.
	dupond, dupont, err := uds.NewPair()
	defer func() {
		_ = dupond.Close()
		_ = dupont.Close()
	}()
	if err != nil {
		s.Slog().Error("cannot create unix domain socket pair",
			slog.Int("PID", os.Getpid()),
			slog.String("err", err.Error()))
		return &api.ErrorResponse{Reason: "failed to create unix domain socket pair, reason: " + err.Error()}
	}

	// In order to pass one of the connected unix domain sockets to the
	// about-to-be-started sub service, we first need to get an *os.File (which
	// while being a duplicate of the socket has a lifecycle of its own).
	dupontf, err := dupont.File()
	if err != nil {
		s.Slog().Error("cannot fetch service *os.File",
			slog.Int("PID", os.Getpid()),
			slog.String("err", err.Error()))
		return &api.ErrorResponse{Reason: "failed to fetch service *os.File, reason: " + err.Error()}
	}
	defer func() { _ = dupontf.Close() }()

	// We can finally start ourselves again as a new child process, creating the
	// requested user and PID namespaces.
	subspace := exec.Command(cmp.Or(s.Exe, "/proc/self/exe"))
	subspace.Stdout = cmp.Or(s.Stdout, io.Writer(os.Stdout))
	subspace.Stderr = cmp.Or(s.Stderr, io.Writer(os.Stderr))
	subspace.ExtraFiles = []*os.File{dupontf}
	subspace.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: uintptr(req.Spaces & uint64(unix.CLONE_NEWUSER|unix.CLONE_NEWPID)),
		// We additionally need to map at least our root UID and root GUID
		// between parent and child user namespace as otherwise we won't be able
		// to create other namespaces inside the child user namespace.
		UidMappings: []syscall.SysProcIDMap{
			{
				HostID:      0,
				ContainerID: 0,
				Size:        1,
			},
		},
		GidMappings: []syscall.SysProcIDMap{
			{
				HostID:      0,
				ContainerID: 0,
				Size:        1,
			},
		},
	}
	s.Slog().Info("starting new subspace service instance")
	if err := subspace.Start(); err != nil {
		s.Slog().Error("cannot start sub service",
			slog.Int("PID", os.Getpid()),
			slog.String("err", err.Error()))
		return &api.ErrorResponse{Reason: "failed to start sub service, reason: " + err.Error()}
	}
	go func() {
		childpid := subspace.Process.Pid
		s.Slog().Info("waiting in background for subspace to close",
			slog.Int("pid", childpid))
		_, _ = subspace.Process.Wait()
		s.Slog().Info("subspace closed", slog.Int("pid", childpid))
	}()

	// Good! We finally can prepare our response; but for this we need to get
	// our hands on the file descriptor for other connected unix domain socket...
	dupondf, err := dupond.File()
	if err != nil {
		s.Slog().Error("cannot fetch client *os.File",
			slog.Int("PID", os.Getpid()),
			slog.String("err", err.Error()))
		return &api.ErrorResponse{Reason: "failed to fetch client *os.File, reason: " + err.Error()}
	}
	defer func() { _ = dupondf.Close() }()

	connfd, err := unix.Dup(int(dupondf.Fd()))
	if err != nil {
		s.Slog().Error("cannot fetch client fd",
			slog.Int("PID", os.Getpid()),
			slog.String("err", err.Error()))
		return &api.ErrorResponse{Reason: "failed to fetch client fd, reason: " + err.Error()}
	}

	var userfd, pidfd int
	if req.Spaces&unix.CLONE_NEWUSER != 0 {
		userfd, err = unix.Open(fmt.Sprintf("/proc/%d/ns/user", subspace.Process.Pid), os.O_RDONLY, 0)
		if err != nil {
			_ = unix.Close(connfd)
			s.Slog().Error("cannot fetch new user namespace",
				slog.Int("PID", os.Getpid()),
				slog.String("err", err.Error()))
			return &api.ErrorResponse{Reason: "failed to determine new user namespace, reason: " + err.Error()}
		}
	}
	if req.Spaces&unix.CLONE_NEWPID != 0 {
		pidfd, err = unix.Open(fmt.Sprintf("/proc/%d/ns/pid", subspace.Process.Pid), os.O_RDONLY, 0)
		if err != nil {
			_ = unix.Close(userfd)
			_ = unix.Close(connfd)
			s.Slog().Error("cannot fetch new PID namespace",
				slog.Int("PID", os.Getpid()),
				slog.String("err", err.Error()))
			return &api.ErrorResponse{Reason: "failed to determine new PID namespace, reason: " + err.Error()}
		}
	}

	return &api.SubspaceResponse{
		Conn: connfd,
		Subspaces: api.Subspaces{
			User: userfd,
			PID:  pidfd,
		},
	}
}

// Room creates one or multiple new namespaces of the requested types, returning
// file descriptors referencing these new namespaces. Room does not allow
// namespaces of type PID and user to be created due to restrictions in the
// Linux kernel when running multi-threaded; use the Subspace service instead.
func (s *Spacemaker) Room(req *api.RoomsRequest) api.Response {
	if req.Spaces & ^uint64(validSpaces) != 0 {
		return &api.ErrorResponse{Reason: "out of space"}
	}
	if req.Spaces&validSpaces == 0 {
		return &api.ErrorResponse{Reason: "no space requested"}
	}

	resp := &api.RoomsResponse{}

	// if the passed type of namespace is requested, then create a new namespace
	// of said type and store its referencing file descriptor in *fd. We're
	// thankful to observe all safety precautions and thus create new namespaces
	// on separate throw-away go routines. We're thus feeling so safe and zupa
	// zekure, welcome to the cult.
	errmsg := ""
	unshare := func(typ uint64, fd *int) {
		if req.Spaces&typ == 0 {
			return // ...nope, not requested.
		}
		ch := make(chan struct {
			fd  int
			err error
		})
		go func() {
			defer close(ch)
			fd, err := s.newNamespace(int(typ))
			ch <- struct {
				fd  int
				err error
			}{fd: fd, err: err}
			// OS-level thread still unlocked, so will get thrown away
		}()
		res := <-ch
		if res.err != nil {
			if errmsg != "" {
				errmsg += ","
			}
			errmsg += spacetest.Name(int(typ)) + ":" + res.err.Error()
			return
		}
		*fd = res.fd
	}

	unshare(unix.CLONE_NEWCGROUP, &resp.Cgroup)
	unshare(unix.CLONE_NEWIPC, &resp.IPC)
	unshare(unix.CLONE_NEWNS, &resp.Mnt)
	unshare(unix.CLONE_NEWNET, &resp.Net)
	unshare(unix.CLONE_NEWTIME, &resp.Time)
	unshare(unix.CLONE_NEWUTS, &resp.UTS)

	if errmsg != "" {
		if resp.Cgroup > 0 {
			_ = unix.Close(resp.Cgroup)
		}
		if resp.IPC > 0 {
			_ = unix.Close(resp.IPC)
		}
		if resp.Mnt > 0 {
			_ = unix.Close(resp.Mnt)
		}
		if resp.Net > 0 {
			_ = unix.Close(resp.Net)
		}
		if resp.Time > 0 {
			_ = unix.Close(resp.Time)
		}
		if resp.UTS > 0 {
			_ = unix.Close(resp.UTS)
		}
		return &api.ErrorResponse{Reason: errmsg}
	}

	return resp
}

// newNamespace returns a file descriptor referencing a new namespace of the
// passed type, or 0 in case of failure. When returning, the caller's go routine
// will intentionally still be locked to its OS-level thread so that it will be
// thrown away after the caller's go routine finally terminates. Thus, call
// newNamespace on a separate throw-away go routine.
func (s *Spacemaker) newNamespace(typ int) (int, error) {
	runtime.LockOSThread()
	// never unlock

	name := spacetest.Name(typ)
	callerns, err := unix.Open("/proc/thread-self/ns/"+name, unix.O_RDONLY, 0)
	if err != nil {
		s.Slog().Error("cannot determine current namespace",
			slog.String("type", name),
			slog.String("err", err.Error()))
		return 0, err
	}
	defer func() {
		_ = unix.Setns(callerns, 0)
		_ = unix.Close(callerns)
	}()
	if typ == unix.CLONE_NEWNS {
		if err := unix.Unshare(unix.CLONE_FS); err != nil {
			s.Slog().Error("cannot unshare fs attributes",
				slog.String("type", name),
				slog.String("err", err.Error()))
			return 0, err
		}
	}
	if err := unix.Unshare(typ); err != nil {
		s.Slog().Error("cannot create new namespace",
			slog.String("type", name),
			slog.String("err", err.Error()))
		return 0, err
	}
	if typ == unix.CLONE_NEWNS {
		if err := unix.Mount("none", "/", "/", unix.MS_REC|unix.MS_PRIVATE, ""); err != nil {
			s.Slog().Error("cannot recursively remount / as private",
				slog.String("type", name),
				slog.String("err", err.Error()))
			return 0, err
		}
	}
	newns, err := unix.Open("/proc/thread-self/ns/"+name, unix.O_RDONLY, 0)
	if err != nil {
		s.Slog().Error("cannot determine new namespace",
			slog.String("type", name),
			slog.String("err", err.Error()))
		return 0, err
	}
	return newns, nil
}
