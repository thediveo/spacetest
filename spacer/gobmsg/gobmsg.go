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

package gobmsg

import (
	"bytes"
	"encoding/gob"
)

const blocksize = 8192

type Encoder struct {
	buff bytes.Buffer
	enc  gob.Encoder
}

func NewEncoder() *Encoder {
	enc := &Encoder{}
	enc.buff.Grow(blocksize)
	enc.enc = *gob.NewEncoder(&enc.buff)
	return enc
}

func (e *Encoder) Encode(v any) ([]byte, error) {
	e.buff.Reset()
	if err := e.enc.Encode(v); err != nil {
		return nil, err
	}
	return e.buff.Bytes(), nil
}

type Decoder struct {
	buff []byte
	r    *bytes.Reader
	dec  *gob.Decoder
}

func NewDecoder() *Decoder {
	buff := make([]byte, blocksize)
	r := bytes.NewReader(buff)
	return &Decoder{
		buff: buff,
		r:    r,
		dec:  gob.NewDecoder(r),
	}
}

func (d *Decoder) Buffer() []byte {
	return d.buff
}

func (d *Decoder) Decode(n int, v any) error {
	d.r.Reset(d.buff[:n])
	return d.dec.Decode(v)
}
