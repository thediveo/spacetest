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

	gi "github.com/onsi/ginkgo/v2"
)

// Execute fn synchronously in the network namespace referenced by the open file
// descriptor netnsfd.
//
// This is a convenience wrapper for [spacetest.Execute]; the latter allows to
// specify multiple namespaces to switch into in a single Execute.
func Execute(netnsfd int, fn func()) {
	gi.GinkgoHelper()

	spacetest.Execute(fn, netnsfd)
}
