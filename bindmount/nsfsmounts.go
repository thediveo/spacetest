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

package bindmount

import (
	"iter"
	"os"
	"strings"

	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck // ST1001 rule does not apply
	. "github.com/onsi/gomega"    //nolint:staticcheck // ST1001 rule does not apply
)

// mountinfo field indices for fields before the “-” delimiter. See:
// https://www.man7.org/linux/man-pages/man5/proc_pid_mountinfo.5.html
const (
	mountIDField = iota
	parentIDField
	stdevField
	rootField
	mountpointField
)

// mountinfo field indices for fields after the “-” delimiter. See:
// https://www.man7.org/linux/man-pages/man5/proc_pid_mountinfo.5.html
const (
	fsTypeField = iota
)

// AllNsfsMounts returns an iterator over all nsfs type bind-mounts in the
// current mount namespace. It fails the current spec/test if
// “/proc/thread-self/mountinfo” cannot be read.
func AllNsfsMounts() iter.Seq[string] {
	GinkgoHelper()
	return AllNsfsMountsPath("/proc/thread-self/mountinfo")
}

// AllNsfsMountsPath returns an iterator over all nsfs type bind-mounts listed
// in the path in mountinfo format. It fails the current spec/test if the
// specified path cannot be read.
func AllNsfsMountsPath(path string) iter.Seq[string] {
	return func(yield func(string) bool) {
		GinkgoHelper()

		mntinfo, err := os.ReadFile(path)
		Expect(err).NotTo(HaveOccurred())

		for line := range strings.Lines(string(mntinfo)) {
			block1, block2, ok := strings.Cut(line, " - ")
			if !ok {
				continue
			}

			beforeSep := strings.Fields(block1)
			if len(beforeSep) < 5 {
				continue
			}

			afterSep := strings.Fields(block2)
			if len(afterSep) < 1 || afterSep[fsTypeField] != "nsfs" {
				continue
			}

			if !yield(beforeSep[mountpointField]) {
				return
			}
		}
	}
}
