// Copyright (c) 2026 John Dewey

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to
// deal in the Software without restriction, including without limitation the
// rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
// sell copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER
// DEALINGS IN THE SOFTWARE.

package wire_test

import (
	"bytes"
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/vmihailenco/msgpack/v5"

	"github.com/retr0h/tonestack/pkg/sdk/internal/wire"
)

// EncodePublicTestSuite covers writing one value the way the byte it replaces
// was written.
type EncodePublicTestSuite struct {
	suite.Suite
}

// TestEncodeLike covers keeping the original's width where the value fits it.
func (s *EncodePublicTestSuite) TestEncodeLike() {
	tests := []struct {
		name  string
		value any
		// like is the first byte of the value being replaced.
		like byte
		want []byte
		// check the result with a real decoder rather than against the bytes
		// this package produced.
		decodes bool
		err     error
		errText string
	}{
		{
			name:  "a wide number stays wide when the new one fits",
			value: 7,
			like:  0xcd,
			want:  []byte{0xcd, 0x00, 0x07},
		},
		{
			name:    "a model number, read back by a real decoder",
			value:   300,
			like:    0x05,
			want:    []byte{0xcd, 0x01, 0x2c},
			decodes: true,
		},
		{
			name:    "a value with no encoding",
			value:   make(chan int),
			err:     wire.ErrBadValue,
			errText: "chan int",
		},
		{
			name:  "a narrow number that still fits stays narrow",
			value: 12,
			like:  0x05,
			want:  []byte{0x0c},
		},
		{
			name:  "a parameter is written as a float32",
			value: 0.25,
			like:  0xca,
			want:  []byte{0xca, 0x3e, 0x80, 0x00, 0x00},
		},
		{
			name:  "turning a block off",
			value: false,
			like:  0xc3,
			want:  []byte{0xc2},
		},
		{
			name:  "turning a block on",
			value: true,
			like:  0xc2,
			want:  []byte{0xc3},
		},
		{
			name:  "clearing a body",
			value: nil,
			like:  0x85,
			want:  []byte{0xc0},
		},
		{
			name:  "a float32 argument is a float32 on the wire",
			value: float32(0.25),
			like:  0xca,
			want:  []byte{0xca, 0x3e, 0x80, 0x00, 0x00},
		},
		{
			name:  "a float stays float64 when that is what it replaces",
			value: 1.0,
			like:  0xcb,
			want:  []byte{0xcb, 0x3f, 0xf0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		},
		{
			name:  "a string keeps a str8 prefix",
			value: "bc",
			like:  0xd9,
			want:  []byte{0xd9, 0x02, 0x62, 0x63},
		},
		{
			name:  "a string keeps a str16 prefix",
			value: "bc",
			like:  0xda,
			want:  []byte{0xda, 0x00, 0x02, 0x62, 0x63},
		},
		{
			// Past what a fixstr holds but inside a str8. The original's
			// width is checked first, so a str16 stays one.
			name:  "a string of 32 to 255 bytes keeps a str16 prefix",
			value: strings.Repeat("a", 40),
			like:  0xda,
			want:  append([]byte{0xda, 0x00, 0x28}, strings.Repeat("a", 40)...),
		},
		{
			name:  "a string too long for its str8 widens",
			value: strings.Repeat("a", 300),
			like:  0xd9,
			want:  append([]byte{0xda, 0x01, 0x2c}, strings.Repeat("a", 300)...),
		},
		{
			name:  "a short string is a fixstr",
			value: "bc",
			like:  0x00,
			want:  []byte{0xa2, 0x62, 0x63},
		},
		{
			name:  "a string past 31 bytes takes a str8",
			value: strings.Repeat("a", 40),
			like:  0x00,
			want:  append([]byte{0xd9, 0x28}, strings.Repeat("a", 40)...),
		},
		{
			name:  "a string past 255 bytes takes a str16",
			value: strings.Repeat("a", 300),
			like:  0x00,
			want:  append([]byte{0xda, 0x01, 0x2c}, strings.Repeat("a", 300)...),
		},
		{
			name:  "a string past 65535 bytes takes a str32",
			value: strings.Repeat("a", 70000),
			like:  0x00,
			want: append(
				[]byte{0xdb, 0x00, 0x01, 0x11, 0x70},
				strings.Repeat("a", 70000)...),
		},
		{
			name:  "a negative number reaching for a fixint",
			value: -8,
			like:  0x00,
			want:  []byte{0xf8},
		},
		{
			name:  "a number past 127 reaching for a uint8",
			value: 200,
			like:  0x00,
			want:  []byte{0xcc, 0xc8},
		},
		{
			name:  "a number below -32 reaching for an int8",
			value: -100,
			like:  0x00,
			want:  []byte{0xd0, 0x9c},
		},
		{
			name:  "a number past 255 reaching for a uint16",
			value: 1000,
			like:  0x00,
			want:  []byte{0xcd, 0x03, 0xe8},
		},
		{
			name:  "a number below -128 reaching for an int16",
			value: -1000,
			like:  0x00,
			want:  []byte{0xd1, 0xfc, 0x18},
		},
		{
			name:  "a number past 65535 reaching for a uint32",
			value: 100000,
			like:  0x00,
			want:  []byte{0xce, 0x00, 0x01, 0x86, 0xa0},
		},
		{
			name:  "a number below -32768 reaching for an int32",
			value: -100000,
			like:  0x00,
			want:  []byte{0xd2, 0xff, 0xfe, 0x79, 0x60},
		},
		{
			name:  "a number past four bytes reaching for a uint64",
			value: int64(math.MaxUint32) + 1,
			like:  0x00,
			want:  []byte{0xcf, 0, 0, 0, 0x01, 0, 0, 0, 0},
		},
		{
			name:  "a number below four bytes reaching for an int64",
			value: int64(math.MinInt32) - 1,
			like:  0x00,
			want:  []byte{0xd3, 0xff, 0xff, 0xff, 0xff, 0x7f, 0xff, 0xff, 0xff},
		},
		{
			name:  "an int8 that still fits its own width",
			value: -100,
			like:  0xd0,
			want:  []byte{0xd0, 0x9c},
		},
		{
			name:  "an int16 that still fits its own width",
			value: -1000,
			like:  0xd1,
			want:  []byte{0xd1, 0xfc, 0x18},
		},
		{
			name:  "a uint32 that still fits its own width",
			value: 7,
			like:  0xce,
			want:  []byte{0xce, 0, 0, 0, 0x07},
		},
		{
			name:  "an int32 that still fits its own width",
			value: -7,
			like:  0xd2,
			want:  []byte{0xd2, 0xff, 0xff, 0xff, 0xf9},
		},
		{
			name:  "a uint64 that still fits its own width",
			value: 7,
			like:  0xcf,
			want:  []byte{0xcf, 0, 0, 0, 0, 0, 0, 0, 0x07},
		},
		{
			name:  "an int64 that still fits its own width",
			value: -7,
			like:  0xd3,
			want: []byte{
				0xd3, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xf9,
			},
		},
		{
			name:  "a negative fixint that still fits its own width",
			value: -8,
			like:  0xff,
			want:  []byte{0xf8},
		},
		{
			name:  "a negative number replacing a wide unsigned value",
			value: -1,
			like:  0xcf,
			want:  []byte{0xff},
		},
		{
			name:  "a small number replacing something that is not a number",
			value: 5,
			like:  0xc0,
			want:  []byte{0x05},
		},
		{
			name:  "a uint8 that still fits its own width",
			value: 200,
			like:  0xcc,
			want:  []byte{0xcc, 0xc8},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := wire.EncodeLike(tt.value, tt.like)

			if tt.err != nil {
				s.Require().ErrorIs(err, tt.err)
				s.Require().Contains(err.Error(), tt.errText)

				return
			}

			s.Require().NoError(err)
			s.Require().Equal(tt.want, got)

			if !tt.decodes {
				return
			}

			var back any
			s.Require().NoError(msgpack.NewDecoder(bytes.NewReader(got)).Decode(&back))
			s.Require().EqualValues(tt.value, back)
		})
	}
}

func TestEncodePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(EncodePublicTestSuite))
}
