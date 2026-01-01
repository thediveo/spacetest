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

package api

import (
	"github.com/thediveo/spacetest"
	"golang.org/x/sys/unix"
)

// RoomsRequest requests new namespaces of the types cgroup, IPC, mnt, net, time,
// and UTS. It cannot be used to request PID and user namespaces, use
// [SubspaceRequest] instead.
type RoomsRequest struct {
	// at most unix.CLONE_NEWCGROUP | unix.CLONE_NEWIPC | unix.CLONE_NEWNS |
	// unix.CLONE_NEWNET | unix.CLONE_NEWTIME | unix.CLONE_NEWUTS; but not
	// unix.CLONE_NEWUSER | unix.CLONE_NEWPID
	Spaces uint64
}

// RoomsResponse contains open file descriptors (>0) referencing the requested
// new namespaces. A zero file descriptor value indicates that no namespace of
// that particular type was requested and created.
//
// Please note that the receiver takes ownership of the returned file
// descriptors and thus is responsible to close them when not needing them
// anymore.
type RoomsResponse struct {
	Cgroup, IPC, Mnt, Net, Time, UTS int
}

var _ Request = (*RoomsRequest)(nil)

func (s RoomsRequest) request() {}

var (
	_ Response   = (*RoomsResponse)(nil)
	_ FdsEncoder = (*RoomsResponse)(nil)
	_ FdsDecoder = (*RoomsResponse)(nil)
)

func (s RoomsResponse) response() {}

// EncodeFds returns the file descriptors contained in the response message,
// replacing the original message fields with zero values so the fields don't
// get transferred by gob. gob, not golb.
func (s *RoomsResponse) EncodeFds() []int {
	return auxiliaryFds(nil).borrow(&s.Cgroup).
		borrow(&s.IPC).
		borrow(&s.Mnt).
		borrow(&s.Net).
		borrow(&s.Time).
		borrow(&s.UTS)
}

// DecodeFds distributes the passed file descriptors that were received as
// auxiliary data with a response message back into their corresponding message
// fields. DecodeFds closes any passed file descriptors it cannot make any sense
// of.
func (s *RoomsResponse) DecodeFds(fds []int) {
	for _, fd := range fds {
		switch typ, _ := unix.IoctlRetInt(fd, spacetest.NS_GET_NSTYPE); typ {
		case unix.CLONE_NEWCGROUP:
			s.Cgroup = fd
		case unix.CLONE_NEWIPC:
			s.IPC = fd
		case unix.CLONE_NEWNS:
			s.Mnt = fd
		case unix.CLONE_NEWNET:
			s.Net = fd
		case unix.CLONE_NEWTIME:
			s.Time = fd
		case unix.CLONE_NEWUTS:
			s.UTS = fd
		default:
			_ = unix.Close(fd)
		}
	}
}
