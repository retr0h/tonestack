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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/vmihailenco/msgpack/v5"

	"github.com/retr0h/tonestack/pkg/sdk/wire"
)

// SplicePublicTestSuite covers editing a section without re-encoding it.
//
// Re-encoding is what corrupts a preset. Line 6 pick an encoding per field
// rather than per value, so a decode and encode of section 0 of preset.bin
// returns 835 bytes where the device wrote 588. Splicing exists so the other
// 587 bytes are never read, let alone rewritten.
type SplicePublicTestSuite struct {
	suite.Suite
}

// every section a device writes, in the order the offset table lists them.
var sectionKeys = []int8{0, 1, 2, 3, 4, 5, 6, 7, 10}

// captures are the three slots an HX Stomp was asked for.
func (s *SplicePublicTestSuite) captures() []string {
	return []string{"preset.bin", "switches.bin", "empty.bin"}
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
		err  string
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
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			from := tt.body
			if from == nil {
				from = body
			}

			start, end, err := wire.Locate(from, tt.path)

			if tt.err != "" {
				s.Require().ErrorIs(err, wire.ErrNoSuchPath)
				s.Require().Contains(err.Error(), tt.err)

				return
			}

			s.Require().NoError(err)
			s.Require().Equal(tt.want, from[start:end])
		})
	}
}

func (s *SplicePublicTestSuite) TestSpliceRaw() {
	body := s.blocks()

	tests := []struct {
		name string
		path wire.Path
		raw  []byte
		want []byte
		err  string
	}{
		{
			name: "putting a value back leaves the section alone",
			path: wire.Path{22, 2, 20, 24, 25},
			raw:  []byte{0xcd, 0x01, 0x05},
			want: body,
		},
		{
			name: "a shorter replacement shortens the section",
			path: wire.Path{22, 2, 20, 24, 25},
			raw:  []byte{0x07},
		},
		{
			name: "a whole subtree can be replaced",
			path: wire.Path{22, 2, 20},
			raw:  []byte{0xc0},
		},
		{
			name: "a path naming nothing changes nothing",
			path: wire.Path{22, 99},
			raw:  []byte{0x00},
			err:  "index 99 is outside an array of 20",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := wire.SpliceRaw(body, tt.path, tt.raw)

			if tt.err != "" {
				s.Require().ErrorIs(err, wire.ErrNoSuchPath)
				s.Require().Contains(err.Error(), tt.err)

				return
			}

			s.Require().NoError(err)

			if tt.want != nil {
				s.Require().Equal(tt.want, got)

				return
			}

			start, end, err := wire.Locate(body, tt.path)
			s.Require().NoError(err)
			s.Require().Equal(len(body)-(end-start)+len(tt.raw), len(got))
			s.Require().Equal(body[:start], got[:start])
			s.Require().Equal(tt.raw, got[start:start+len(tt.raw)])
			s.Require().Equal(body[end:], got[start+len(tt.raw):])
		})
	}
}

// TestSpliceRawKeepsEveryOtherByte is the claim the whole design rests on.
//
// Every value in every section of every capture is located and written back
// as itself. Nothing outside the located range is ever read, so a section
// that survives this survives any edit of one value.
func (s *SplicePublicTestSuite) TestSpliceRawKeepsEveryOtherByte() {
	spliced := 0

	for _, name := range s.captures() {
		for _, key := range sectionKeys {
			body := s.section(name, key)

			for _, path := range s.paths(body) {
				start, end, err := wire.Locate(body, path)
				s.Require().NoError(err, "%s section %d path %v", name, key, path)

				got, err := wire.SpliceRaw(body, path, body[start:end])
				s.Require().NoError(err)
				s.Require().True(bytes.Equal(body, got),
					"%s section %d path %v changed the section", name, key, path)

				spliced++
			}
		}
	}

	s.T().Logf("SPLICED %d", spliced)
	s.Require().Equal(5413, spliced, "every value in all three captures")
}

func (s *SplicePublicTestSuite) TestSplice() {
	body := s.blocks()

	tests := []struct {
		name  string
		body  []byte
		path  wire.Path
		value any
		want  []byte
		err   error
	}{
		{
			name:  "a wide number stays wide when the new one fits",
			path:  wire.Path{22, 2, 20, 24, 25},
			value: 7,
			want:  []byte{0xcd, 0x00, 0x07},
		},
		{
			name:  "a narrow number widens only when it must",
			path:  wire.Path{22, 3, 20, 24, 25},
			value: 300,
			want:  []byte{0xcd, 0x01, 0x2c},
		},
		{
			name:  "a narrow number that still fits stays narrow",
			path:  wire.Path{22, 3, 20, 24, 25},
			value: 12,
			want:  []byte{0x0c},
		},
		{
			name:  "a parameter is written as a float32",
			path:  wire.Path{22, 3, 20, 11, 4, 0},
			value: 0.25,
			want:  []byte{0xca, 0x3e, 0x80, 0x00, 0x00},
		},
		{
			name:  "turning a block off",
			path:  wire.Path{22, 3, 20, 10},
			value: false,
			want:  []byte{0xc2},
		},
		{
			name:  "clearing a body",
			path:  wire.Path{22, 2, 20},
			value: nil,
			want:  []byte{0xc0},
		},
		{
			name:  "a value with no encoding",
			path:  wire.Path{22, 3, 20, 10},
			value: struct{}{},
			err:   wire.ErrBadValue,
		},
		{
			name:  "a path naming nothing",
			path:  wire.Path{404},
			value: 1,
			err:   wire.ErrNoSuchPath,
		},
		{
			name:  "a float32 argument is a float32 on the wire",
			path:  wire.Path{22, 3, 20, 11, 4, 0},
			value: float32(0.25),
			want:  []byte{0xca, 0x3e, 0x80, 0x00, 0x00},
		},
		{
			name:  "a float stays float64 when that is what it replaces",
			body:  s.wrap([]byte{0xcb, 0, 0, 0, 0, 0, 0, 0, 0}),
			path:  wire.Path{1},
			value: 1.0,
			want:  []byte{0xcb, 0x3f, 0xf0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		},
		{
			name:  "a string keeps a str8 prefix",
			body:  s.wrap([]byte{0xd9, 0x01, 0x61}),
			path:  wire.Path{1},
			value: "bc",
			want:  []byte{0xd9, 0x02, 0x62, 0x63},
		},
		{
			name:  "a string keeps a str16 prefix",
			body:  s.wrap([]byte{0xda, 0x00, 0x01, 0x61}),
			path:  wire.Path{1},
			value: "bc",
			want:  []byte{0xda, 0x00, 0x02, 0x62, 0x63},
		},
		{
			name:  "a short string is a fixstr",
			body:  s.wrap([]byte{0x00}),
			path:  wire.Path{1},
			value: "bc",
			want:  []byte{0xa2, 0x62, 0x63},
		},
		{
			name:  "a string past 31 bytes takes a str8",
			body:  s.wrap([]byte{0x00}),
			path:  wire.Path{1},
			value: strings.Repeat("a", 40),
			want:  append([]byte{0xd9, 0x28}, strings.Repeat("a", 40)...),
		},
		{
			name:  "a string past 255 bytes takes a str16",
			body:  s.wrap([]byte{0x00}),
			path:  wire.Path{1},
			value: strings.Repeat("a", 300),
			want:  append([]byte{0xda, 0x01, 0x2c}, strings.Repeat("a", 300)...),
		},
		{
			name:  "a string past 65535 bytes takes a str32",
			body:  s.wrap([]byte{0x00}),
			path:  wire.Path{1},
			value: strings.Repeat("a", 70000),
			want: append(
				[]byte{0xdb, 0x00, 0x01, 0x11, 0x70},
				strings.Repeat("a", 70000)...),
		},
		{
			name:  "a negative number reaching for a fixint",
			body:  s.wrap([]byte{0x00}),
			path:  wire.Path{1},
			value: -8,
			want:  []byte{0xf8},
		},
		{
			name:  "a number past 127 reaching for a uint8",
			body:  s.wrap([]byte{0x00}),
			path:  wire.Path{1},
			value: 200,
			want:  []byte{0xcc, 0xc8},
		},
		{
			name:  "a number below -32 reaching for an int8",
			body:  s.wrap([]byte{0x00}),
			path:  wire.Path{1},
			value: -100,
			want:  []byte{0xd0, 0x9c},
		},
		{
			name:  "a number past 255 reaching for a uint16",
			body:  s.wrap([]byte{0x00}),
			path:  wire.Path{1},
			value: 1000,
			want:  []byte{0xcd, 0x03, 0xe8},
		},
		{
			name:  "a number below -128 reaching for an int16",
			body:  s.wrap([]byte{0x00}),
			path:  wire.Path{1},
			value: -1000,
			want:  []byte{0xd1, 0xfc, 0x18},
		},
		{
			name:  "a number past 65535 reaching for a uint32",
			body:  s.wrap([]byte{0x00}),
			path:  wire.Path{1},
			value: 100000,
			want:  []byte{0xce, 0x00, 0x01, 0x86, 0xa0},
		},
		{
			name:  "a number below -32768 reaching for an int32",
			body:  s.wrap([]byte{0x00}),
			path:  wire.Path{1},
			value: -100000,
			want:  []byte{0xd2, 0xff, 0xfe, 0x79, 0x60},
		},
		{
			name:  "a number past four bytes reaching for a uint64",
			body:  s.wrap([]byte{0x00}),
			path:  wire.Path{1},
			value: int64(math.MaxUint32) + 1,
			want:  []byte{0xcf, 0, 0, 0, 0x01, 0, 0, 0, 0},
		},
		{
			name:  "a number below four bytes reaching for an int64",
			body:  s.wrap([]byte{0x00}),
			path:  wire.Path{1},
			value: int64(math.MinInt32) - 1,
			want:  []byte{0xd3, 0xff, 0xff, 0xff, 0xff, 0x7f, 0xff, 0xff, 0xff},
		},
		{
			name:  "an int8 that still fits its own width",
			body:  s.wrap([]byte{0xd0, 0x80}),
			path:  wire.Path{1},
			value: -100,
			want:  []byte{0xd0, 0x9c},
		},
		{
			name:  "an int16 that still fits its own width",
			body:  s.wrap([]byte{0xd1, 0x80, 0x00}),
			path:  wire.Path{1},
			value: -1000,
			want:  []byte{0xd1, 0xfc, 0x18},
		},
		{
			name:  "a uint32 that still fits its own width",
			body:  s.wrap([]byte{0xce, 0, 0, 0, 0}),
			path:  wire.Path{1},
			value: 7,
			want:  []byte{0xce, 0, 0, 0, 0x07},
		},
		{
			name:  "an int32 that still fits its own width",
			body:  s.wrap([]byte{0xd2, 0x80, 0, 0, 0}),
			path:  wire.Path{1},
			value: -7,
			want:  []byte{0xd2, 0xff, 0xff, 0xff, 0xf9},
		},
		{
			name:  "a uint64 that still fits its own width",
			body:  s.wrap([]byte{0xcf, 0, 0, 0, 0, 0, 0, 0, 0}),
			path:  wire.Path{1},
			value: 7,
			want:  []byte{0xcf, 0, 0, 0, 0, 0, 0, 0, 0x07},
		},
		{
			name:  "an int64 that still fits its own width",
			body:  s.wrap([]byte{0xd3, 0, 0, 0, 0, 0, 0, 0, 0}),
			path:  wire.Path{1},
			value: -7,
			want: []byte{
				0xd3, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xf9,
			},
		},
		{
			name:  "a negative fixint that still fits its own width",
			body:  s.wrap([]byte{0xff}),
			path:  wire.Path{1},
			value: -8,
			want:  []byte{0xf8},
		},
		{
			name:  "a negative number replacing a wide unsigned value",
			body:  s.wrap([]byte{0xcf, 0, 0, 0, 0, 0, 0, 0, 0}),
			path:  wire.Path{1},
			value: -1,
			want:  []byte{0xff},
		},
		{
			name:  "turning a block on",
			path:  wire.Path{22, 4, 20, 10},
			value: true,
			want:  []byte{0xc3},
		},
		{
			name:  "a small number replacing something that is not a number",
			body:  s.wrap([]byte{0xc0}),
			path:  wire.Path{1},
			value: 5,
			want:  []byte{0x05},
		},
		{
			name:  "a uint8 that still fits its own width",
			body:  s.wrap([]byte{0xcc, 0x05}),
			path:  wire.Path{1},
			value: 200,
			want:  []byte{0xcc, 0xc8},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			from := tt.body
			if from == nil {
				from = body
			}

			got, err := wire.Splice(from, tt.path, tt.value)

			if tt.err != nil {
				s.Require().ErrorIs(err, tt.err)

				return
			}

			s.Require().NoError(err)

			start, _, err := wire.Locate(from, tt.path)
			s.Require().NoError(err)
			s.Require().Equal(tt.want, got[start:start+len(tt.want)])
		})
	}
}

// TestASplicedSectionStillDecodes checks the result against a real decoder
// rather than against the bytes this package produced.
func (s *SplicePublicTestSuite) TestASplicedSectionStillDecodes() {
	body := s.blocks()

	got, err := wire.Splice(body, wire.Path{22, 3, 20, 24, 25}, 300)
	s.Require().NoError(err)

	dec := msgpack.NewDecoder(bytes.NewReader(got))
	dec.SetMapDecoder(func(d *msgpack.Decoder) (any, error) {
		return d.DecodeUntypedMap()
	})

	raw, err := dec.DecodeInterface()
	s.Require().NoError(err)

	doc, ok := raw.(map[any]any)
	s.Require().True(ok)

	entry := s.at(s.index(s.at(doc, 22), 3), 20)
	s.Require().EqualValues(300, s.at(s.at(entry, 24), 25))
}

// TestMalformedSectionsAreReported covers bytes no device would send.
func (s *SplicePublicTestSuite) TestMalformedSectionsAreReported() {
	tests := []struct {
		name string
		body []byte
		path wire.Path
		want error
		msg  string
	}{
		{
			name: "an empty section",
			body: []byte{},
			path: wire.Path{},
			want: wire.ErrNoSuchPath,
			msg:  "ran off the end",
		},
		{
			name: "a map claiming a pair that is not there",
			body: []byte{0x81},
			path: wire.Path{1},
			want: wire.ErrMalformed,
			msg:  "no value here",
		},
		{
			name: "an array claiming an element that is not there",
			body: s.wrap([]byte{0x91}),
			path: wire.Path{1},
			want: wire.ErrMalformed,
			msg:  "no value here",
		},
		{
			name: "a string running past the end",
			body: []byte{0x81, 0x01, 0xd9, 0x40, 0x61},
			path: wire.Path{},
			want: wire.ErrMalformed,
			msg:  "value runs past the end",
		},
		{
			name: "a length prefix running past the end",
			body: []byte{0x81, 0x01, 0xda, 0x00},
			path: wire.Path{1},
			want: wire.ErrMalformed,
			msg:  "length runs past the end",
		},
		{
			name: "a map16 with no count",
			body: []byte{0x81, 0x01, 0xde, 0x00},
			path: wire.Path{1, 0},
			want: wire.ErrNoSuchPath,
			msg:  "is not a container, it is code 0xde",
		},
		{
			name: "a leaf where the path expects a container",
			body: s.wrap([]byte{0x2a}),
			path: wire.Path{1, 0},
			want: wire.ErrNoSuchPath,
			msg:  "is not a container",
		},
		{
			name: "a code the format does not define",
			body: []byte{0xc1},
			path: wire.Path{},
			want: wire.ErrMalformed,
			msg:  "unknown code 0xc1",
		},
		{
			name: "a key that is not an integer",
			body: []byte{0x81, 0xa1, 0x61, 0x01},
			path: wire.Path{1},
			want: wire.ErrMalformed,
			msg:  "is not an integer",
		},
		{
			name: "a value that cannot be stepped over to reach a later key",
			body: []byte{0x82, 0x01, 0xd9, 0x40, 0x61, 0x02, 0x2a},
			path: wire.Path{2},
			want: wire.ErrMalformed,
			msg:  "value runs past the end",
		},
		{
			name: "an element that cannot be stepped over to reach a later one",
			body: s.wrap([]byte{0x92, 0xd9, 0x40, 0x61}),
			path: wire.Path{1, 1},
			want: wire.ErrMalformed,
			msg:  "value runs past the end",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			_, _, err := wire.Locate(tt.body, tt.path)

			s.Require().ErrorIs(err, tt.want)
			s.Require().Contains(err.Error(), tt.msg)
		})
	}
}

// TestBadValueErrorNamesTheType is the other error this package raises.
func (s *SplicePublicTestSuite) TestBadValueErrorNamesTheType() {
	_, err := wire.Splice(s.blocks(), wire.Path{21}, make(chan int))

	s.Require().ErrorIs(err, wire.ErrBadValue)
	s.Require().Contains(err.Error(), "chan int")
}

// paths walks a section and returns every path in it.
func (s *SplicePublicTestSuite) paths(body []byte) []wire.Path {
	dec := msgpack.NewDecoder(bytes.NewReader(body))
	dec.SetMapDecoder(func(d *msgpack.Decoder) (any, error) {
		return d.DecodeUntypedMap()
	})

	raw, err := dec.DecodeInterface()
	s.Require().NoError(err)

	return s.walk(raw, wire.Path{})
}

// walk collects the path to every value under one.
func (s *SplicePublicTestSuite) walk(
	v any,
	at wire.Path,
) []wire.Path {
	out := []wire.Path{append(wire.Path{}, at...)}

	switch t := v.(type) {
	case map[any]any:
		for key, child := range t {
			n, ok := s.asInt(key)
			s.Require().True(ok, "every key in a preset is an integer")
			out = append(out, s.walk(child, append(at, n))...)
		}
	case []any:
		for i, child := range t {
			out = append(out, s.walk(child, append(at, i))...)
		}
	}

	return out
}

// at reads one integer-keyed entry, whatever width the key arrived in.
func (s *SplicePublicTestSuite) at(
	v any,
	key int,
) any {
	m, ok := v.(map[any]any)
	s.Require().True(ok, "expected a map, got %T", v)

	for k, child := range m {
		if n, ok := s.asInt(k); ok && n == key {
			return child
		}
	}

	s.Require().Fail("no key", "%d is not in the map", key)

	return nil
}

// index reads one array element.
func (s *SplicePublicTestSuite) index(
	v any,
	i int,
) any {
	arr, ok := v.([]any)
	s.Require().True(ok, "expected an array, got %T", v)

	return arr[i]
}

// asInt accepts whichever width a key arrived in.
func (s *SplicePublicTestSuite) asInt(v any) (int, bool) {
	switch n := v.(type) {
	case int8:
		return int(n), true
	case int16:
		return int(n), true
	case int32:
		return int(n), true
	case int64:
		return int(n), true
	case int:
		return n, true
	case uint8:
		return int(n), true
	case uint16:
		return int(n), true
	case uint32:
		return int(n), true
	case uint64:
		return int(n), true
	default:
		return 0, false
	}
}

func TestSplicePublicTestSuite(t *testing.T) {
	suite.Run(t, new(SplicePublicTestSuite))
}
