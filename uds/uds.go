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

package uds

import (
	"errors"
	"net"
	"os"

	"golang.org/x/sys/unix"
)

// Conn represents a (stream) unix domain socket connection that can send and
// receive open file descriptors. It wraps [*net.UnixConn]. Use [NewPair] to
// create a pair of directly peer-to-peer connected Conn objects. Use
// [Conn.SendWithFds] and [Conn.ReceiveWithFds] to transfer requests and
// responses with open file descriptors piggybacked on.
type Conn struct {
	*net.UnixConn
}

// NewPair returns a pair of peer-to-peer connected (stream) unix domain sockets
// that can transfer open file descriptors across process boundaries.
func NewPair() (dupond, dupont *Conn, err error) {
	fdpair, err := unix.Socketpair(unix.AF_UNIX, unix.SOCK_STREAM, 0)
	if err != nil {
		return nil, nil, err
	}
	dupond, err = NewUnixConn(fdpair[0], "dupond")
	if err != nil {
		// fdpair[0] is always closed by now, but we don't want to leak
		// fdpair[1]...
		_ = unix.Close(fdpair[1])
		return nil, nil, err
	}
	dupont, err = NewUnixConn(fdpair[1], "dupont")
	if err != nil {
		// fdpair[0] was closed already, fdpair[1] is always closed by now too,
		// so we only need to dispose of the first successfully created
		// UnixConn...
		_ = dupond.Close()
		return nil, nil, err
	}
	return dupond, dupont, nil
}

// SendWithFds sends the passed data as well as the passed file descriptors over
// the (stream) UDS connection in a single control message (ancillary data).
func (c *Conn) SendWithFds(b []byte, fds ...int) (noob int, err error) {
	// Please note that unix.UnixRights returns a single control message
	// consisting of the header as well as the fd payload.
	oob := unix.UnixRights(fds...)
	_, noob, err = c.WriteMsgUnix(b, oob, nil)
	return noob, err
}

// ReceiveWithFds returns the file descriptors received in a single control
// message (ancillary data) from the (stream) UDS connection, otherwise it
// returns an error.
func (c *Conn) ReceiveWithFds(b []byte, maxfds int) (n int, fds []int, err error) {
	// We're trying to do the reverse of what unix.UnixRights does: it packages
	// file descriptors as int32's and then there's control message header
	// overhead, but this is where unix.CmsgSpace gives us the correct number
	// for the amount of control message payload.
	oob := make([]byte, unix.CmsgSpace(maxfds*4))
	n, noob, _, _, err := c.ReadMsgUnix(b, oob)
	if err != nil {
		return 0, nil, err
	}
	cms, err := unix.ParseSocketControlMessage(oob[:noob])
	if err != nil {
		return 0, nil, err
	}
	for _, cm := range cms {
		if cm.Header.Level != unix.SOL_SOCKET || cm.Header.Type != unix.SCM_RIGHTS {
			continue // nah, don't understand, skip it.
		}
		fds, err := unix.ParseUnixRights(&cm)
		if err != nil {
			return 0, nil, err
		}
		return n, fds, err
	}
	// no fds received is also okay, such as when receiving error responses.
	return n, nil, nil
}

// NewUnixConn returns a *net.UnixConn for the passed unix domain socket fd;
// otherwise, it then returns an error in case of failure.
//
// Why do we want a UnixConn? Because it has ReadMsgUnix and WriteMsgUnix
// methods for receiving and sending out-of-band data, also known as “control
// information” or “ancillary data” (for instance, see sendmsg(2),
// https://www.man7.org/linux/man-pages/man2/sendmsg.2.html).
//
// Important: NewUnixConn always takes ownership of the passed file descriptor
// and will close it, even in case of error. A caller must not use the passed
// file descriptor anymore and the caller must not close the passed file
// descriptor themselves.
func NewUnixConn(udsfd int, nickname string) (*Conn, error) {
	f := os.NewFile(uintptr(udsfd), nickname)
	if f == nil {
		return nil, errors.New("not a file descriptor")
	}
	defer func() { _ = f.Close() }()
	netconn, err := net.FilePacketConn(f)
	if err != nil {
		return nil, err
	}
	unixconn, ok := netconn.(*net.UnixConn)
	if !ok {
		_ = netconn.Close()
		return nil, errors.New("not a unix domain socket")
	}
	return &Conn{UnixConn: unixconn}, nil
}
