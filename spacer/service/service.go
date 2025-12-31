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
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"time"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/thediveo/spacetest/spacer/api"
	"github.com/thediveo/spacetest/spacer/gobmsg"
	"github.com/thediveo/spacetest/uds"
	"golang.org/x/sys/unix"
)

// Spacer services the spacer API.
type Spacer interface {
	Subspace(*api.SubspaceRequest) api.Response
	Room(*api.RoomsRequest) api.Response
	Slog() *slog.Logger
}

// Serve services requests on the passed *uds.Conn until the client disconnects,
// using the passed spacer to carry out the requests.
//
// Since this function is used in testing, it generates slog records over the
// course of its operation. You might thus want to send slog output to the
// GinkgoWriter: this way, you won't be bothered with slog output unless your
// test fails ($HEAVENS forbid!) or you explicitly request to see it all using
// “-ginkgo.v” when running tests.
func Serve(ctx context.Context, conn *uds.Conn, spacer Spacer) {
	id := petname.Generate(2, "-")
	spacer.Slog().Info("spacer serving loop started", slog.String("spacer-id", id))
	defer func() {
		spacer.Slog().Info("spacer serving loop terminated", slog.String("spacer-id", id))
	}()

	enc := gobmsg.NewEncoder()
	dec := gobmsg.NewDecoder()

	for {
		// Check and exit if the context is done by now.
		select {
		case <-ctx.Done():
			spacer.Slog().Info("context cancelled", slog.String("spacer-id", id))
			return
		default:
		}
		// Now try to read in the next service request; we don't expect any fds
		// with it. We set a read deadline so that we can check our context from
		// time to time. If we hit the deadline that's fine, we simply restart.
		if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
			spacer.Slog().Error("cannot set deadline",
				slog.String("spacer-id", id),
				slog.String("err", err.Error()))
			return
		}
		n, _, err := conn.ReceiveWithFds(dec.Buffer(), 0)
		if err != nil {
			if errors.Is(err, os.ErrDeadlineExceeded) {
				continue
			}
			// https://go.dev/wiki/ErrorValueFAQ
			if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
				spacer.Slog().Info("client disconnected", slog.String("spacer-id", id))
				return
			}
			spacer.Slog().Error("cannot receive",
				slog.String("spacer-id", id),
				slog.String("err", err.Error()))
			return
		}
		// Try to decode the read service request contained in the received
		// message. Please note that req will then hold the request value
		// itself, but not a pointer to a request value. Gotcha.
		var req api.Request
		if err := dec.Decode(n, &req); err != nil {
			spacer.Slog().Error("cannot decode incoming request",
				slog.String("spacer-id", id),
				slog.String("err", err.Error()))
			return
		}
		// handle the service request and get a response.
		spacer.Slog().Info("serving request",
			slog.String("spacer-id", id),
			slog.String("service", fmt.Sprintf("%T", req)))
		var resp api.Response
		switch req := req.(type) {
		case *api.SubspaceRequest:
			resp = spacer.Subspace(req)
		case *api.RoomsRequest:
			resp = spacer.Room(req)
		default:
			spacer.Slog().Error("unhandled request",
				slog.String("spacer-id", id),
				slog.String("type", fmt.Sprintf("%T", req)))
			return
		}
		// Finally encode the response; pay attention to passing a pointer to
		// the interface, see also the gob "interface" example,
		// https://pkg.go.dev/encoding/gob#example-package-Interface
		msg, err := enc.Encode(&resp)
		if err != nil {
			spacer.Slog().Error("cannot encode response",
				slog.String("spacer-id", id),
				slog.String("err", err.Error()))
			return
		}
		// are there any file descriptors to transfer...?
		var fds []int
		if fdsencoder, ok := resp.(api.FdsEncoder); ok {
			fds = fdsencoder.EncodeFds()
		}
		_, err = conn.SendWithFds(msg, fds...)
		// Make sure to close the file descriptors because they're now in
		// transit with the kernel in charge, or the kernel didn't take
		// ownership and then we need to close them also as to not leak them.
		for _, fd := range fds {
			_ = unix.Close(fd)
		}
		if err != nil {
			spacer.Slog().Error("cannot send",
				slog.String("spacer-id", id),
				slog.String("err", err.Error()))
			return
		}
	}
}
