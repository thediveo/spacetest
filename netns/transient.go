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

package netns

import (
	"github.com/thediveo/spacetest"
	"golang.org/x/sys/unix"

	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck // ST1001 rule does not apply
)

// EnterTransient creates and enters a new (and isolated) network namespace,
// returning a function that needs to be defer'ed in order to correctly switch
// the calling go routine and its locked OS-level thread back when the caller
// itself returns.
//
//	defer netns.EnterTransient()() // sic!
//
// In case the caller cannot be switched back correctly, the defer'ed clean up
// will panic with an error description.
func EnterTransient() func() {
	GinkgoHelper()

	return spacetest.EnterTransient(unix.CLONE_NEWNET)
}

// NewTransient creates a new network namespace, but doesn't enter it. Instead,
// it returns a file descriptor referencing the new network namespace.
// NewTransient also schedules a Ginkgo deferred cleanup in order to close the
// fd referencing the newly created network namespace. The caller thus must not
// close the file descriptor returned.
func NewTransient() int {
	GinkgoHelper()

	return spacetest.NewTransient(unix.CLONE_NEWNET)
}
