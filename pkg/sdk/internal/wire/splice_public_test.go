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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/internal/wire"
)

// SplicePublicTestSuite covers finding the bytes one value occupies, so a
// section can be edited without re-encoding it.
//
// Re-encoding is what corrupts a preset. Line 6 pick an encoding per field
// rather than per value, so a decode and encode of section 0 of preset.bin
// returns 835 bytes where the device wrote 588. Locating a value exists so the
// other 587 bytes are never read, let alone rewritten.
type SplicePublicTestSuite struct {
	suite.Suite
}

// section returns one section of one capture, exactly as the device sent it.
func (s *SplicePublicTestSuite) section(
	name string,
	key int8,
) []byte {
	raw, err := os.ReadFile(filepath.Join("testdata", name))
	s.Require().NoError(err)

	doc, err := wire.DecodeDocument(raw)
	s.Require().NoError(err)

	body, ok := doc.Section(key)
	s.Require().True(ok, "capture %s has no section %d", name, key)

	return body
}

// blocks is section 0 of the bass preset, the one every path below indexes.
func (s *SplicePublicTestSuite) blocks() []byte {
	return s.section("preset.bin", 0)
}

// wrap builds the one-pair map {1: value}, so a row can supply any encoding.
func (s *SplicePublicTestSuite) wrap(value []byte) []byte {
	return append([]byte{0x81, 0x01}, value...)
}

func (s *SplicePublicTestSuite) TestLocate() {
	body := s.blocks()

	tests := []struct {
		name string
		body []byte
		path wire.Path
		want []byte
		// what the failure must be, ErrNoSuchPath unless a case says
		// otherwise, and what it must say.
		errIs error
		err   string
	}{
		{
			name: "the routing kind of the first entry",
			path: wire.Path{22, 0, 19},
			want: []byte{0x00},
		},
		{
			name: "a model number the device wrote wide",
			path: wire.Path{22, 2, 20, 24, 25},
			want: []byte{0xcd, 0x01, 0x05},
		},
		{
			name: "a model number that fitted a fixint",
			path: wire.Path{22, 3, 20, 24, 25},
			want: []byte{0x78},
		},
		{
			name: "whether a block is on",
			path: wire.Path{22, 3, 20, 10},
			want: []byte{0xc3},
		},
		{
			name: "a parameter, always a float32",
			path: wire.Path{22, 3, 20, 11, 4, 0},
			want: []byte{0xca, 0x3f, 0x47, 0xae, 0x14},
		},
		{
			name: "a body the device left empty",
			path: wire.Path{22, 1, 20},
			want: []byte{0xc0},
		},
		{
			name: "an empty path is the whole section",
			path: wire.Path{},
			want: body,
		},
		{
			name: "a key that is not in the map",
			path: wire.Path{99},
			err:  "key 99 is not in the map",
		},
		{
			name: "an index past the end of the array",
			path: wire.Path{22, 20},
			err:  "index 20 is outside an array of 20",
		},
		{
			name: "a path that keeps going after a leaf",
			path: wire.Path{21, 0},
			err:  "is not a container",
		},
		{
			name: "a negative fixint",
			body: s.wrap([]byte{0xff}),
			path: wire.Path{1},
			want: []byte{0xff},
		},
		{
			name: "an int8",
			body: s.wrap([]byte{0xd0, 0x80}),
			path: wire.Path{1},
			want: []byte{0xd0, 0x80},
		},
		{
			name: "a uint32",
			body: s.wrap([]byte{0xce, 0x00, 0x01, 0x00, 0x00}),
			path: wire.Path{1},
			want: []byte{0xce, 0x00, 0x01, 0x00, 0x00},
		},
		{
			name: "an int32",
			body: s.wrap([]byte{0xd2, 0xff, 0xff, 0xff, 0xff}),
			path: wire.Path{1},
			want: []byte{0xd2, 0xff, 0xff, 0xff, 0xff},
		},
		{
			name: "a uint64",
			body: s.wrap([]byte{0xcf, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01}),
			path: wire.Path{1},
			want: []byte{0xcf, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
		},
		{
			name: "an int64",
			body: s.wrap([]byte{0xd3, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}),
			path: wire.Path{1},
			want: []byte{0xd3, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		},
		{
			name: "a float64",
			body: s.wrap([]byte{0xcb, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}),
			path: wire.Path{1},
			want: []byte{0xcb, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		},
		{
			name: "a fixstr",
			body: s.wrap([]byte{0xa1, 0x61}),
			path: wire.Path{1},
			want: []byte{0xa1, 0x61},
		},
		{
			name: "a str8",
			body: s.wrap([]byte{0xd9, 0x01, 0x61}),
			path: wire.Path{1},
			want: []byte{0xd9, 0x01, 0x61},
		},
		{
			name: "a str16",
			body: s.wrap([]byte{0xda, 0x00, 0x01, 0x61}),
			path: wire.Path{1},
			want: []byte{0xda, 0x00, 0x01, 0x61},
		},
		{
			name: "a str32",
			body: s.wrap([]byte{0xdb, 0x00, 0x00, 0x00, 0x01, 0x61}),
			path: wire.Path{1},
			want: []byte{0xdb, 0x00, 0x00, 0x00, 0x01, 0x61},
		},
		{
			name: "a bin8",
			body: s.wrap([]byte{0xc4, 0x01, 0x00}),
			path: wire.Path{1},
			want: []byte{0xc4, 0x01, 0x00},
		},
		{
			name: "a bin16",
			body: s.wrap([]byte{0xc5, 0x00, 0x01, 0x00}),
			path: wire.Path{1},
			want: []byte{0xc5, 0x00, 0x01, 0x00},
		},
		{
			name: "a bin32",
			body: s.wrap([]byte{0xc6, 0x00, 0x00, 0x00, 0x01, 0x00}),
			path: wire.Path{1},
			want: []byte{0xc6, 0x00, 0x00, 0x00, 0x01, 0x00},
		},
		{
			name: "an ext8",
			body: s.wrap([]byte{0xc7, 0x01, 0x0a, 0x00}),
			path: wire.Path{1},
			want: []byte{0xc7, 0x01, 0x0a, 0x00},
		},
		{
			name: "an ext16",
			body: s.wrap([]byte{0xc8, 0x00, 0x01, 0x0a, 0x00}),
			path: wire.Path{1},
			want: []byte{0xc8, 0x00, 0x01, 0x0a, 0x00},
		},
		{
			name: "an ext32",
			body: s.wrap([]byte{0xc9, 0x00, 0x00, 0x00, 0x01, 0x0a, 0x00}),
			path: wire.Path{1},
			want: []byte{0xc9, 0x00, 0x00, 0x00, 0x01, 0x0a, 0x00},
		},
		{
			name: "a fixext1",
			body: s.wrap([]byte{0xd4, 0x0a, 0x00}),
			path: wire.Path{1},
			want: []byte{0xd4, 0x0a, 0x00},
		},
		{
			name: "a fixext2",
			body: s.wrap([]byte{0xd5, 0x0a, 0x00, 0x00}),
			path: wire.Path{1},
			want: []byte{0xd5, 0x0a, 0x00, 0x00},
		},
		{
			name: "a fixext4",
			body: s.wrap([]byte{0xd6, 0x0a, 0x00, 0x00, 0x00, 0x00}),
			path: wire.Path{1},
			want: []byte{0xd6, 0x0a, 0x00, 0x00, 0x00, 0x00},
		},
		{
			name: "a fixext8",
			body: s.wrap([]byte{0xd7, 0x0a, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}),
			path: wire.Path{1},
			want: []byte{0xd7, 0x0a, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		},
		{
			name: "a fixext16",
			body: s.wrap(
				[]byte{
					0xd8,
					0x0a,
					0x00,
					0x00,
					0x00,
					0x00,
					0x00,
					0x00,
					0x00,
					0x00,
					0x00,
					0x00,
					0x00,
					0x00,
					0x00,
					0x00,
					0x00,
					0x00,
				},
			),
			path: wire.Path{1},
			want: []byte{
				0xd8,
				0x0a,
				0x00,
				0x00,
				0x00,
				0x00,
				0x00,
				0x00,
				0x00,
				0x00,
				0x00,
				0x00,
				0x00,
				0x00,
				0x00,
				0x00,
				0x00,
				0x00,
			},
		},
		{
			name: "an array16",
			body: s.wrap([]byte{0xdc, 0x00, 0x01, 0x2a}),
			path: wire.Path{1, 0},
			want: []byte{0x2a},
		},
		{
			name: "an array32",
			body: s.wrap([]byte{0xdd, 0x00, 0x00, 0x00, 0x01, 0x2a}),
			path: wire.Path{1, 0},
			want: []byte{0x2a},
		},
		{
			name: "a map16",
			body: s.wrap([]byte{0xde, 0x00, 0x01, 0x02, 0x2a}),
			path: wire.Path{1, 2},
			want: []byte{0x2a},
		},
		{
			name: "a map32",
			body: s.wrap([]byte{0xdf, 0x00, 0x00, 0x00, 0x01, 0x02, 0x2a}),
			path: wire.Path{1, 2},
			want: []byte{0x2a},
		},
		{
			name: "a key written as a negative fixint",
			body: []byte{0x81, 0xff, 0x2a},
			path: wire.Path{-1},
			want: []byte{0x2a},
		},
		{
			name: "a key written as a uint8",
			body: []byte{0x81, 0xcc, 0xc8, 0x2a},
			path: wire.Path{200},
			want: []byte{0x2a},
		},
		{
			name: "a key written as an int8",
			body: []byte{0x81, 0xd0, 0xc0, 0x2a},
			path: wire.Path{-64},
			want: []byte{0x2a},
		},
		{
			name: "a key written as a uint16",
			body: []byte{0x81, 0xcd, 0x01, 0x00, 0x2a},
			path: wire.Path{256},
			want: []byte{0x2a},
		},
		{
			name: "a key written as an int16",
			body: []byte{0x81, 0xd1, 0xff, 0x00, 0x2a},
			path: wire.Path{-256},
			want: []byte{0x2a},
		},
		{
			name: "a key written as a uint32",
			body: []byte{0x81, 0xce, 0x00, 0x01, 0x00, 0x00, 0x2a},
			path: wire.Path{65536},
			want: []byte{0x2a},
		},
		{
			name: "a key written as an int32",
			body: []byte{0x81, 0xd2, 0xff, 0xff, 0x00, 0x00, 0x2a},
			path: wire.Path{-65536},
			want: []byte{0x2a},
		},
		{
			name: "an index below the start of an array",
			body: s.wrap([]byte{0x91, 0x2a}),
			path: wire.Path{1, -1},
			err:  "index -1 is outside an array of 1",
		},
		// Bytes no device would send.
		{
			name:  "an empty section",
			body:  []byte{},
			path:  wire.Path{},
			errIs: wire.ErrNoSuchPath,
			err:   "ran off the end",
		},
		{
			name:  "a map claiming a pair that is not there",
			body:  []byte{0x81},
			path:  wire.Path{1},
			errIs: wire.ErrMalformed,
			err:   "no value here",
		},
		{
			name:  "an array claiming an element that is not there",
			body:  s.wrap([]byte{0x91}),
			path:  wire.Path{1},
			errIs: wire.ErrMalformed,
			err:   "no value here",
		},
		{
			name:  "a string running past the end",
			body:  []byte{0x81, 0x01, 0xd9, 0x40, 0x61},
			path:  wire.Path{},
			errIs: wire.ErrMalformed,
			err:   "value runs past the end",
		},
		{
			name:  "a length prefix running past the end",
			body:  []byte{0x81, 0x01, 0xda, 0x00},
			path:  wire.Path{1},
			errIs: wire.ErrMalformed,
			err:   "length runs past the end",
		},
		{
			name:  "a map16 with no count",
			body:  []byte{0x81, 0x01, 0xde, 0x00},
			path:  wire.Path{1, 0},
			errIs: wire.ErrNoSuchPath,
			err:   "is not a container, it is code 0xde",
		},
		{
			name:  "a leaf where the path expects a container",
			body:  s.wrap([]byte{0x2a}),
			path:  wire.Path{1, 0},
			errIs: wire.ErrNoSuchPath,
			err:   "is not a container",
		},
		{
			name:  "a code the format does not define",
			body:  []byte{0xc1},
			path:  wire.Path{},
			errIs: wire.ErrMalformed,
			err:   "unknown code 0xc1",
		},
		{
			name:  "a key that is not an integer",
			body:  []byte{0x81, 0xa1, 0x61, 0x01},
			path:  wire.Path{1},
			errIs: wire.ErrMalformed,
			err:   "is not an integer",
		},
		{
			name:  "a value that cannot be stepped over to reach a later key",
			body:  []byte{0x82, 0x01, 0xd9, 0x40, 0x61, 0x02, 0x2a},
			path:  wire.Path{2},
			errIs: wire.ErrMalformed,
			err:   "value runs past the end",
		},
		{
			name:  "an element that cannot be stepped over to reach a later one",
			body:  s.wrap([]byte{0x92, 0xd9, 0x40, 0x61}),
			path:  wire.Path{1, 1},
			errIs: wire.ErrMalformed,
			err:   "value runs past the end",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			from := tt.body
			if from == nil {
				from = body
			}

			start, end, err := wire.Locate(from, tt.path)

			if tt.err != "" {
				want := tt.errIs
				if want == nil {
					want = wire.ErrNoSuchPath
				}

				s.Require().ErrorIs(err, want)
				s.Require().Contains(err.Error(), tt.err)

				return
			}

			s.Require().NoError(err)
			s.Require().Equal(tt.want, from[start:end])
		})
	}
}

func TestSplicePublicTestSuite(t *testing.T) {
	suite.Run(t, new(SplicePublicTestSuite))
}
