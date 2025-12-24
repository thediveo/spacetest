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

// Encoder provides a gob encoder encoding into a byte slice.
type Encoder struct {
	buff bytes.Buffer
	enc  gob.Encoder
}

// NewEncoder returns a new encoder that maintains an internal buffer to encode
// into.
func NewEncoder() *Encoder {
	enc := &Encoder{}
	enc.buff.Grow(blocksize)
	enc.enc = *gob.NewEncoder(&enc.buff)
	return enc
}

// Encode the passed value in gob form and return its binary representation as a
// byte slice. The returned slice becomes invalid at the next call to Encode.
func (e *Encoder) Encode(v any) ([]byte, error) {
	e.buff.Reset()
	if err := e.enc.Encode(v); err != nil {
		return nil, err
	}
	return e.buff.Bytes(), nil
}

// Decoder provides a gob decoder decoding from a byte slice.
type Decoder struct {
	buff []byte
	r    *bytes.Reader
	dec  *gob.Decoder
}

// NewDecoder returns a new decoder that maintains an internal buffer to receive
// encoded data into, and to decode from.
func NewDecoder() *Decoder {
	buff := make([]byte, blocksize)
	r := bytes.NewReader(buff)
	return &Decoder{
		buff: buff,
		r:    r,
		dec:  gob.NewDecoder(r),
	}
}

// Buffer returns a buffer slice to be used for receiving data.
func (d *Decoder) Buffer() []byte {
	return d.buff
}

// Decode returns the decoded value currently stored in the first n bytes of the
// decoder's buffer. First, read a gob message into the slice provided by
// [Decoder.Buffer], also determining the amount of data read. Then call
// [Decoder.Decode] with this amount of data read to decode the value.
func (d *Decoder) Decode(n int, v any) error {
	d.r.Reset(d.buff[:n])
	return d.dec.Decode(v)
}
