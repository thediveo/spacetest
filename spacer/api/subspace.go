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

// SubspaceRequest indicates which hierarchical namespaces to create, as the
// OR'ed combination of at most unix.CLONE_NEWUSER and unix.CLONE_NEWPID. The
// request must specify at least one of the user and pid namespaces, but is not
// allowed to specify any other type of namespace.
type SubspaceRequest struct {
	Spaces uint64 // at most unix.CLONE_NEWUSER | unix.CLONE_NEWPID
}

// SubspaceResponse returns the connected unix domain socket to talk to a
// subspace spacer service instance, as well as the new user and/or PID
// namespaces created along with the subspacer service.
//
// Please note that the receiver takes ownership of the returned file
// descriptors and thus is responsible to close them when not needing them
// anymore. Closing the connection fd will also terminate the connected subspace
// service; sub-subspace services will not be affected.
type SubspaceResponse struct {
	Conn int // fd of client unix domain socket
	Subspaces
}

// Subspaces contains namespace references in form of open file descriptors. The
// receiver of a Subspaces value takes ownership and is thus responsible to
// properly close them when not needing them anymore.
type Subspaces struct {
	User int // if >0, the user namespace referencing fd.
	PID  int // if >0, the PID namespace referencing fd.
}

var _ Request = (*SubspaceRequest)(nil)

func (s SubspaceRequest) request() {}

var (
	_ Response   = (*SubspaceResponse)(nil)
	_ FdsEncoder = (*SubspaceResponse)(nil)
	_ FdsDecoder = (*SubspaceResponse)(nil)
)

func (s SubspaceResponse) response() {}

// EncodeFds returns the file descriptors contained in the response message,
// replacing the original message fields with zero values so the fields don't
// get transferred by gob.
func (s *SubspaceResponse) EncodeFds() []int {
	return auxiliaryFds(nil).
		borrow(&s.Conn).
		borrow(&s.User).
		borrow(&s.PID)
}

// DecodeFds distributes the passed file descriptors that were received as
// auxiliary data with a response message back into their corresponding message
// fields. DecodeFds closes any passed file descriptors it cannot make any sense
// of.
func (s *SubspaceResponse) DecodeFds(fds []int) {
	s.Conn = fds[0]
	for _, fd := range fds[1:] {
		switch typ, _ := unix.IoctlRetInt(fd, spacetest.NS_GET_NSTYPE); typ {
		case unix.CLONE_NEWUSER:
			s.User = fd
		case unix.CLONE_NEWPID:
			s.PID = fd
		default:
			_ = unix.Close(fd)
		}
	}
}
