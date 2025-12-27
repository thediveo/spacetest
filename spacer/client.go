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
	"time"

	gi "github.com/onsi/ginkgo/v2"
	g "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/thediveo/spacetest/spacer/api"
	"github.com/thediveo/spacetest/spacer/gobmsg"
	"github.com/thediveo/spacetest/spacer/service"
	"github.com/thediveo/spacetest/uds"
	"golang.org/x/sys/unix"
)

// Client ...
//
// Note: Client CANNOT be used concurrently.
type Client struct {
	conn *uds.Conn
	enc  *gobmsg.Encoder
	dec  *gobmsg.Decoder
}

// FIXME: cleanup
var spacerServiceBinary string

// New returns a new spacer client connected to a new spacer service instance.
// The service instance will terminate either when the passed context gets
// cancelled or when the Close method of the returned client object is called.
func New(ctx context.Context) *Client {
	gi.GinkgoHelper()

	if spacerServiceBinary == "" {
		var err error
		spacerServiceBinary, err = gexec.BuildWithEnvironment(
			"github.com/thediveo/spacetest/spacer/service/cmd",
			[]string{"CGO_ENABLED=0"},
			"-tags=usergo,netgo")
		g.Expect(err).NotTo(g.HaveOccurred(), "cannot build spacer service binary")
	}

	dupond, dupont, err := uds.NewPair()
	g.Expect(err).NotTo(g.HaveOccurred(), "cannot create connected unix domain socket pair")

	go func() {
		service.Serve(ctx, dupont, &service.Spacemaker{Exe: spacerServiceBinary})
		_ = dupont.Close()
	}()

	return &Client{
		conn: dupond,
		enc:  gobmsg.NewEncoder(),
		dec:  gobmsg.NewDecoder(),
	}
}

func (c *Client) Close() {
	_ = c.conn.Close()
}

func (c *Client) Subspace(user, pid bool) (*Client, api.Subspaces) {
	gi.GinkgoHelper()

	var spaces uint64
	if user {
		spaces |= unix.CLONE_NEWUSER
	}
	if pid {
		spaces |= unix.CLONE_NEWPID
	}
	var req api.Request = api.SubspaceRequest{Spaces: spaces}
	msg, err := c.enc.Encode(&req)
	g.Expect(err).NotTo(g.HaveOccurred(), "cannot encode subspace request")
	g.Expect(c.conn.SendWithFds(msg)).Error().NotTo(g.HaveOccurred(), "cannot send subspace request")

	g.Expect(c.conn.SetReadDeadline(time.Now().Add(5*time.Second))).To(g.Succeed(), "cannot receive subspace response")
	n, fds, err := c.conn.ReceiveFds(c.dec.Buffer(), 3)
	g.Expect(err).NotTo(g.HaveOccurred(), "cannot receive subspace response")
	var r api.Response
	g.Expect(c.dec.Decode(n, &r)).To(g.Succeed(), "cannot decode subspace response")
	if e, ok := r.(api.ErrorResponse); ok {
		g.Expect(e).To(g.BeZero(), "subspace service failed")
	}
	var resp api.SubspaceResponse
	g.Expect(r).To(g.BeAssignableToTypeOf(resp), "not a subspace response")
	resp = r.(api.SubspaceResponse)
	resp.DecodeFds(fds)

	subconn, err := uds.NewUnixConn(resp.Conn, "subspace")
	g.Expect(err).NotTo(g.HaveOccurred(), "subspace connection failure")
	newclient := &Client{
		conn: subconn,
		enc:  gobmsg.NewEncoder(),
		dec:  gobmsg.NewDecoder(),
	}

	return newclient, resp.Subspaces
}
