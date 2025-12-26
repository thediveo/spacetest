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

	"github.com/thediveo/spacetest/spacer/api"
	"github.com/thediveo/spacetest/uds"
	"golang.org/x/sys/unix"
)

type mock struct{}

var _ Spacer = (*mock)(nil)

func (m *mock) Moin(*api.MoinRequest) api.Response {
	return &api.MoinResponse{}
}

func (m *mock) Room(*api.MakeRequest) api.Response {
	return &api.ErrorResponse{Reason: "not mocked"}
}

// Subspace ...
func (m *mock) Subspace(req *api.SubspaceRequest) api.Response {
	if req.Spaces&(unix.CLONE_NEWUSER|unix.CLONE_NEWPID) == 0 ||
		req.Spaces & ^uint64(unix.CLONE_NEWUSER|unix.CLONE_NEWPID) != 0 {
		return &api.ErrorResponse{Reason: "invalid"}
	}

	dupond, dupont, err := uds.NewPair()
	if err != nil {
		return &api.ErrorResponse{Reason: err.Error()}
	}
	defer func() { _ = dupond.Close() }()

	go func() {
		defer func() { _ = dupont.Close() }()
		Serve(context.Background(), dupont, &mock{})
	}()

	f, err := dupond.File()
	if err != nil {
		return &api.ErrorResponse{Reason: "cannot retrieve File"}
	}
	defer func() { _ = f.Close() }()
	fd, err := unix.Dup(int(f.Fd()))
	if err != nil {
		return &api.ErrorResponse{Reason: "cannot dup fd"}
	}
	resp := &api.SubspaceResponse{
		Conn: fd,
	}

	if req.Spaces&unix.CLONE_NEWUSER != 0 {
		resp.User, err = unix.Open("/proc/self/ns/user", unix.O_RDONLY, 0)
		if err != nil {
			_ = unix.Close(resp.Conn)
			return &api.ErrorResponse{Reason: "user namespace failure"}
		}
	}
	if req.Spaces&unix.CLONE_NEWPID != 0 {
		resp.User, err = unix.Open("/proc/self/ns/pid", unix.O_RDONLY, 0)
		if err != nil {
			_ = unix.Close(resp.Conn)
			_ = unix.Close(resp.User)
			return &api.ErrorResponse{Reason: "user namespace failure"}
		}
	}

	return resp
}
