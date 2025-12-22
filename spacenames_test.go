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

package spacetest

import (
	"golang.org/x/sys/unix"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("type names of namespaces", func() {

	DescribeTable("type names",
		func(typ int, expected string) {
			Expect(Name(typ)).To(Equal(expected))
		},
		Entry(nil, 0, ""),
		Entry(nil, unix.CLONE_NEWCGROUP, "cgroup"),
		Entry(nil, unix.CLONE_NEWIPC, "ipc"),
		Entry(nil, unix.CLONE_NEWNS, "mnt"),
		Entry(nil, unix.CLONE_NEWNET, "net"),
		Entry(nil, unix.CLONE_NEWPID, "pid"),
		Entry(nil, unix.CLONE_NEWTIME, "time"),
		Entry(nil, unix.CLONE_NEWUSER, "user"),
		Entry(nil, unix.CLONE_NEWUTS, "uts"),
	)

})
