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
	"github.com/thediveo/spacetest/spacer/api"
	"github.com/thediveo/spacetest/uds"
	"golang.org/x/sys/unix"
)

type mock struct{}

var _ Spacer = (*mock)(nil)

func (m *mock) Moin(*api.MoinRequest) api.Response {
	return &api.MoinResponse{}
}

func (m *mock) Subspace(req *api.SubspaceRequest) api.Response {
	if req.Spaces == 0 || req.Spaces & ^uint64(unix.CLONE_NEWUSER|unix.CLONE_NEWPID) != 0 {
		return &api.ErrorResponse{Reason: "invalid"}
	}
	dupond, dupont, err := uds.NewPair()
	defer func() {
		_ = dupond.Close()
		_ = dupont.Close()
	}()
	if err != nil {
		return &api.ErrorResponse{Reason: err.Error()}
	}
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
	return resp
}
