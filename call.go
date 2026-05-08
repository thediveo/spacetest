// Copyright 2026 Harald Albrecht.
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

package spacetest

import (
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck // ST1001 rule does not apply
)

// Call the passed fn synchronously, returning fn's single result value, while
// attached to the specified namespace(s) and otherwise defaulting to the
// caller's currently attached namespaces. See [Execute] for details and
// semantics about the other parameters as well as the behaviour especially
// regarding mount namespaces.
func Call[R any](fn func() R, nsfd int, nsfds ...int) R {
	GinkgoHelper()

	// We surely don't want to upset Go's race detector, so we need to cuddle it
	// a bit and employ a channel in the best of Go proverbs.
	ch := make(chan R, 1)
	Execute(func() { ch <- fn() }, nsfd, nsfds...)
	return <-ch
}
