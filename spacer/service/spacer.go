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
)

type Spacemaker struct{}

var _ Spacer = (*Spacemaker)(nil)

func NewSpacemaker(conn *uds.Conn) *Spacemaker {

}

// Moin just responds with Moin as it is ancient custom.
func (s *Spacemaker) Moin(*api.MoinRequest) api.Response {
	return &api.MoinResponse{}
}

// Subspace
func (s *Spacemaker) Subspace(*api.SubspaceRequest) api.Response {
	return nil // FIXME:
}

func (s *Spacemaker) Room(*api.MakeRequest) api.Response {
	return nil // FIXME:
}
