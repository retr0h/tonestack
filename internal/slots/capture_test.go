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

package slots

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
)

// brokenWriter fails every write, so a reporting failure is reported rather
// than dropped.
type brokenWriter struct{}

func (*brokenWriter) Write([]byte) (int, error) { return 0, errors.New("boom") }

// CaptureTestSuite covers what happens to a device's answer before anybody
// knows how to decode it.
type CaptureTestSuite struct {
	suite.Suite
}

// TestDump keeps a device's answer where somebody can read it.
func (s *CaptureTestSuite) TestDump() {
	tests := []struct {
		name  string
		got   any
		file  string
		want  string
		bytes []byte
		err   bool
	}{
		{
			name: "nothing at all when nobody asked",
			got:  map[any]any{"a": 1},
		},
		{
			name: "a decoded answer, as JSON",
			got:  map[string]any{"name": "Longview"},
			file: "reply.json",
			want: "Longview",
		},
		{
			// A preset comes back as an opaque run of bytes that
			// MessagePack's string type carries, and it is not valid UTF-8.
			// Encoding it as JSON would replace every byte above 0x7f, which
			// destroys the offsets and model numbers the capture exists to
			// study.
			name: "a preset, verbatim",
			got:  "\xa9l6-helix\x00\xda\x000\xff\xfe",
			file: "preset.bin",
			want: "\xa9l6-helix\x00\xda\x000\xff\xfe",
		},
		{
			name:  "bytes, verbatim",
			got:   []byte{0xa9, 0x00, 0xff},
			file:  "preset.bin",
			bytes: []byte{0xa9, 0x00, 0xff},
		},
		{
			name: "somewhere it cannot write",
			got:  map[string]any{},
			file: filepath.Join("no", "such", "dir.json"),
			err:  true,
		},
		{
			// Nothing guarantees what a device answers with. Something JSON
			// cannot represent is reported rather than written as a file
			// that lost it.
			name: "something JSON cannot hold",
			got:  make(chan int),
			file: "reply.json",
			err:  true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var path string

			if tt.file != "" {
				path = filepath.Join(s.T().TempDir(), tt.file)
			}

			s.T().Setenv(dumpEnv, path)

			err := dump(tt.got)

			if tt.err {
				s.Require().Error(err)

				return
			}

			s.Require().NoError(err)

			if path == "" {
				return
			}

			body, readErr := os.ReadFile(path) //nolint:gosec // a path this test wrote
			s.Require().NoError(readErr)

			if tt.bytes != nil {
				s.Require().Equal(tt.bytes, body)

				return
			}

			s.Require().Contains(string(body), tt.want)
		})
	}
}

// TestDescribe says what arrived when nothing here can decode it.
func (s *CaptureTestSuite) TestDescribe() {
	tests := []struct {
		name  string
		shape string
		to    io.Writer
		want  string
		err   bool
	}{
		{
			name:  "what the device answered with",
			shape: "map with 2 keys",
			want:  "map with 2 keys",
		},
		{
			name:  "nowhere to say it",
			shape: "map with 2 keys",
			to:    &brokenWriter{},
			err:   true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var buf bytes.Buffer

			to := tt.to
			if to == nil {
				to = &buf
			}

			err := describe(to, "HX Stomp", 3, tt.shape)

			if tt.err {
				s.Require().Error(err)

				return
			}

			s.Require().NoError(err)
			s.Require().Contains(buf.String(), tt.want)
			s.Require().Contains(buf.String(), "slot 02A")
		})
	}
}

func TestCaptureTestSuite(t *testing.T) {
	suite.Run(t, new(CaptureTestSuite))
}
