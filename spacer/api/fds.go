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

// auxillaryFds is a list of open file descriptors, to be transferred as
// auxillary data with some message.
type auxillaryFds []int

// borrow checks if a namespace fd is open (>0) and then appends it to the list
// of file descriptors to transmit as auxillary data as well as zero'ing the fd
// value in its original place (as we don't want to transmit it twice in-band
// and out-of-band). If the referenced fd isn't open, then the original fd list
// will be returned unchanged.
func (f auxillaryFds) borrow(fd *int) auxillaryFds {
	if *fd <= 0 {
		return f
	}
	fds := append(f, *fd)
	*fd = 0
	return fds
}
