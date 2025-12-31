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

package spacer

import (
	"context"
	"io"
	"sync"
	"time"

	gi "github.com/onsi/ginkgo/v2"
	g "github.com/onsi/gomega"
	"github.com/onsi/gomega/gcustom"
	"github.com/onsi/gomega/gexec"
	"github.com/onsi/gomega/types"
	"github.com/thediveo/spacetest/spacer/api"
	"github.com/thediveo/spacetest/spacer/gobmsg"
	"github.com/thediveo/spacetest/spacer/service"
	"github.com/thediveo/spacetest/uds"
	"golang.org/x/sys/unix"
)

// Client connects to exactly one spacer service instance, which might be
// in-process or a separate process.
//
// # Important
//
// Client cannot(!) be used concurrently.
type Client struct {
	conn   *uds.Conn
	enc    *gobmsg.Encoder
	dec    *gobmsg.Decoder
	stdout io.Writer
	stderr io.Writer
}

var (
	spacerbinarymu      sync.Mutex
	spacerServiceBinary string
)

func spacerServicePath() string {
	spacerbinarymu.Lock()
	defer spacerbinarymu.Unlock()

	if spacerServiceBinary != "" {
		return spacerServiceBinary
	}

	gi.By("building the spacer service binary")
	var err error
	spacerServiceBinary, err = gexec.BuildWithEnvironment(
		"github.com/thediveo/spacetest/spacer/service/cmd",
		[]string{"CGO_ENABLED=0"},
		"-tags=usergo,netgo")
	g.Expect(err).NotTo(g.HaveOccurred(), "cannot build spacer service binary")
	return spacerServiceBinary
}

// New returns a new client connected to a new spacer service instance. This
// service instance will terminate either when the passed context gets cancelled
// or when the Close method of the returned client object is called.
//
// Make sure to call [gexec.CleanupBuildArtifacts] in your AfterSuite.
func New(ctx context.Context, opts ...Option) *Client {
	gi.GinkgoHelper()

	c := &Client{}
	for _, opt := range opts {
		g.Expect(opt(c)).To(g.Succeed(), "cannot apply option")
	}

	servicebinpath := spacerServicePath()

	dupond, dupont, err := uds.NewPair()
	g.Expect(err).NotTo(g.HaveOccurred(), "cannot create connected unix domain socket pair")

	go func() {
		service.Serve(ctx, dupont, &service.Spacemaker{
			Exe:    servicebinpath,
			Stdout: c.stdout,
			Stderr: c.stderr,
		})
		_ = dupont.Close()
	}()

	c.conn = dupond
	c.enc = gobmsg.NewEncoder()
	c.dec = gobmsg.NewDecoder()
	return c
}

// Close the connection to the spacer service instance. This will cause the
// previously connected spacer service instance to automatically terminate.
//
// Please note that all Client instances are independent, so closing one will
// not afflict any other Client instance.
func (c *Client) Close() {
	_ = c.conn.Close()
}

// Subspace returns a new client as well as new user and/or PID child
// namespaces. The user and/or PID namespaces are children of the connected
// service's user and PID namespaces. For the “initial” client returned by [New]
// the parent user and PID namespaces are those of the test process's user and
// PID namespaces. For clients returned from Subspace calls the parent PID and
// user namespaces are those of the particular service process's namespaces.
//
// Subspace also schedules a DeferCleanup to automatically close the open file
// descriptors of the namespaces returned when the current node ends, where
// Subspace was called. Callers thus must not close the returned file
// descriptors themselves. Callers are free to [unix.Dup] any
// namespace-referencing file descriptor to break out of this fd lifecycle.
func (c *Client) Subspace(user, pid bool) (*Client, api.Subspaces) {
	gi.GinkgoHelper()

	resp := do[*api.SubspaceResponse](c, api.SubspaceRequest{
		Spaces: uint64(namespaces(0).ifrequested(user, unix.CLONE_NEWUSER).
			ifrequested(pid, unix.CLONE_NEWPID)),
	}, "subspace")
	subconn, err := uds.NewUnixConn(resp.Conn, "subspace")
	g.Expect(err).NotTo(g.HaveOccurred(), "subspace connection failure")
	newclient := &Client{
		conn:   subconn,
		enc:    gobmsg.NewEncoder(),
		dec:    gobmsg.NewDecoder(),
		stdout: c.stdout,
		stderr: c.stderr,
	}

	gi.DeferCleanup(func(userfd, pidfd int) {
		if pidfd > 0 {
			_ = unix.Close(pidfd)
		}
		if userfd > 0 {
			_ = unix.Close(userfd)
		}
	}, resp.User, resp.PID)

	return newclient, resp.Subspaces
}

// NewTransient creates a new Linux kernel namespace of the specified type using
// the connected spacer service, returning a file descriptor referencing the
// newly created namespace.  NewTransient can be used for the following types of
// namespaces:
//   - unix.CLONE_NEWCGROUP,
//   - unix.CLONE_NEWIPC,
//   - unix.CLONE_NEWNS,
//   - unix.CLONE_NEWNET,
//   - unix.CLONE_NEWTIME,
//   - unix.CLONE_NEWUTS.
//
// NewTransient differs from [spacetest.NewTransient] in that it is able to
// create new (non-hierarchical) namespaces inside child user and PID
// namespaces. The latter can't be used in those cases because of the limitation
// that multi-threaded (Go) processes are not allowed to switch into different
// user and PID namespaces.
func (c *Client) NewTransient(typ int) int {
	gi.GinkgoHelper()

	switch typ {
	case unix.CLONE_NEWCGROUP:
		return c.Rooms(true, false, false, false, false, false).Cgroup
	case unix.CLONE_NEWIPC:
		return c.Rooms(false, true, false, false, false, false).IPC
	case unix.CLONE_NEWNS:
		return c.Rooms(false, false, true, false, false, false).Mnt
	case unix.CLONE_NEWNET:
		return c.Rooms(false, false, false, true, false, false).Net
	case unix.CLONE_NEWTIME:
		return c.Rooms(false, false, false, false, true, false).Time
	case unix.CLONE_NEWUTS:
		return c.Rooms(false, false, false, false, false, true).UTS
	}
	g.Expect(typ).To(beInvalid())
	return -1 // never reached
}

func beInvalid() types.GomegaMatcher {
	return gcustom.MakeMatcher(func(int) (bool, error) {
		return true, nil
	}).WithMessage("invalid namespace type")
}

// Rooms returns new namespaces of the requested type(s). The namespaces are
// returned as open file descriptors referencing them.
//
// Rooms also schedules a DeferCleanup to automatically close the open file
// descriptors of the namespaces returned when the current node ends, where
// Rooms was called. Callers thus must not close the returned file descriptors
// themselves. Callers are free to [unix.Dup] any namespace-referencing file
// descriptor to break out of this fd lifecycle.
//
// The namespaces are created “in” the user namespace the client belongs to. In
// case of the client instance returned by [New] this is the caller's program
// user namespace. Client insteances returned by [Client.Subspace] work on their
// respective sub user namespaces (where requested when calling Subspace).
func (c *Client) Rooms(cgroup, ipc, mnt, net, time, uts bool) api.RoomsResponse {
	gi.GinkgoHelper()

	resp := do[*api.RoomsResponse](c, api.RoomsRequest{
		Spaces: uint64(namespaces(0).ifrequested(cgroup, unix.CLONE_NEWCGROUP).
			ifrequested(ipc, unix.CLONE_NEWIPC).
			ifrequested(mnt, unix.CLONE_NEWNS).
			ifrequested(net, unix.CLONE_NEWNET).
			ifrequested(time, unix.CLONE_NEWTIME).
			ifrequested(uts, unix.CLONE_NEWUTS)),
	}, "rooms")

	gi.DeferCleanup(func(cgroupfd, ipcfd, mntfd, netfd, timefd, utsfd int) {
		if cgroupfd > 0 {
			_ = unix.Close(cgroupfd)
		}
		if ipcfd > 0 {
			_ = unix.Close(ipcfd)
		}
		if mntfd > 0 {
			_ = unix.Close(mntfd)
		}
		if netfd > 0 {
			_ = unix.Close(netfd)
		}
		if timefd > 0 {
			_ = unix.Close(timefd)
		}
		if utsfd > 0 {
			_ = unix.Close(utsfd)
		}
	}, resp.Cgroup, resp.IPC, resp.Mnt, resp.Net, resp.Time, resp.UTS)

	return *resp
}

type namespaces uint64

func (n namespaces) ifrequested(b bool, flag uint64) namespaces {
	if !b {
		return n
	}
	return n | namespaces(flag)
}

// do the passed API request, returning a non-failure API response; or otherwise
// failing the current test.
func (c *Client) do(req api.Request, name string) api.Response {
	gi.GinkgoHelper()

	msg, err := c.enc.Encode(&req)
	g.Expect(err).NotTo(g.HaveOccurred(), "cannot encode %s request", name)
	g.Expect(c.conn.SendWithFds(msg)).Error().NotTo(g.HaveOccurred(),
		"cannot send %s request", name)

	g.Expect(c.conn.SetReadDeadline(time.Now().Add(5*time.Second))).To(g.Succeed(),
		"cannot receive %s response", name)
	n, fds, err := c.conn.ReceiveWithFds(c.dec.Buffer(), 3)
	g.Expect(err).NotTo(g.HaveOccurred(), "cannot receive %s response", name)

	var resp api.Response
	g.Expect(c.dec.Decode(n, &resp)).To(g.Succeed(),
		"cannot decode %s response", name)
	g.Expect(resp).NotTo(api.HaveFailed(), "%s service failed", name)
	if r, ok := resp.(api.FdsDecoder); ok {
		r.DecodeFds(fds)
	} else {
		g.Expect(fds).To(g.BeEmpty(),
			"%s service received fds when it shouldn't; response: %T", name, resp)
	}
	return resp
}

// do the passed API request on the specified client, returning a response of
// type R, or otherwise failing the current test.
func do[R any](c *Client, req api.Request, name string) R {
	gi.GinkgoHelper()

	resp := c.do(req, name)
	r, ok := resp.(R)
	g.Expect(ok).To(g.BeTrue(), "not a %s response", name)
	return r
}
