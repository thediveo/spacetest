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
	"encoding/gob"
)

type (
	FdsEncoder interface{ EncodeFds() (fds []int) }
	FdsDecoder interface{ DecodeFds(fds []int) }
)

type (
	Request  interface{ request() }
	Response interface{ response() }
)

// ErrorResponse can be transferred in place of any other service response.
type ErrorResponse struct {
	Reason string
}

var _ Response = (*ErrorResponse)(nil)

func (er ErrorResponse) response() {}

// Register the individual request and response struct types so that we can use
// interface polymorphy when receiving request (sic!) or responses.
func init() {
	gob.Register(&ErrorResponse{})

	gob.Register(&SubspaceRequest{})
	gob.Register(&SubspaceResponse{})
	gob.Register(&RoomsRequest{})
	gob.Register(&RoomsResponse{})
}

type UnhandlebarRequest struct{}

func (u *UnhandlebarRequest) request() {}
