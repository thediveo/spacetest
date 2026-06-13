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

package units

type Capacity uint64

// Common definitions for IEC memory capacities.
//
// See also: [Binary prefix] (Wikipedia)
//
// [Binary prefix]: https://en.wikipedia.org/wiki/Binary_prefix
const (
	B   Capacity = 1
	KiB          = B * 1024
	GiB          = KiB * 1024
	TiB          = GiB * 1024
	PiB          = TiB * 1024
	EiB          = PiB * 1024
)
