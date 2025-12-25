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
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/thediveo/spacetest/spacer/api"
	"github.com/thediveo/spacetest/spacer/gobmsg"
	"github.com/thediveo/spacetest/uds"
)

func Serve(ctx context.Context, conn *uds.Conn) {
	enc := gobmsg.NewEncoder()
	dec := gobmsg.NewDecoder()

	for {
		// Check and exit if the context is done by now.
		select {
		case <-ctx.Done():
			return
		default:
		}
		// Now try to read in the next service request; we don't expect any fds
		// with it. We set a read deadline so that we can check our context from
		// time to time. If we hit the deadline that's fine, we simply restart.
		if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
			slog.Error("cannot set deadline", slog.String("err", err.Error()))
			return
		}
		n, _, err := conn.ReceiveFds(dec.Buffer(), 0)
		if err != nil {
			if errors.Is(err, os.ErrDeadlineExceeded) {
				continue
			}
			// https://go.dev/wiki/ErrorValueFAQ
			if errors.Is(err, io.EOF) {
				return
			}
			slog.Error("cannot receive",
				slog.String("err", err.Error()))
			return
		}
		// try to decode the read service request.
		var req api.Request
		if err := dec.Decode(n, &req); err != nil {
			slog.Error("cannot decode incoming request",
				slog.String("err", err.Error()))
			return
		}
		// handle the service request and get a response.
		var resp api.Response
		switch req := req.(type) {
		case *api.MoinRequest:
			_ = req
			resp = &api.MoinResponse{}
		default:
			resp = &api.ErrorResponse{
				Reason: "unknown request",
			}
		}
		// Finally encode the response; pay attention to passing a pointer to
		// the interface, see also the gob "interface" example,
		// https://pkg.go.dev/encoding/gob#example-package-Interface
		msg, err := enc.Encode(&resp)
		if err != nil {
			slog.Error("cannot encode response", slog.String("err", err.Error()))
			return
		}
		// are there any file descriptors to transfer...?
		var fds []int
		if fdsencoder, ok := resp.(api.FdsEncoder); ok {
			fds = fdsencoder.EncodeFds()
		}
		if _, err := conn.SendWithFds(msg, fds...); err != nil {
			slog.Error("cannot send", slog.String("err", err.Error()))
			return
		}
	}

}
