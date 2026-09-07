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

func (s *CaptureTestSuite) TestDumpWritesNothingUnasked() {
	s.T().Setenv(dumpEnv, "")

	s.Require().NoError(dump(map[any]any{"a": 1}))
}

func (s *CaptureTestSuite) TestDumpKeepsWhatTheDeviceSaid() {
	path := filepath.Join(s.T().TempDir(), "reply.json")
	s.T().Setenv(dumpEnv, path)

	s.Require().NoError(dump(map[string]any{"name": "Longview"}))

	body, err := os.ReadFile(path) //nolint:gosec // a path this test wrote
	s.Require().NoError(err)
	s.Require().Contains(string(body), "Longview")
}

func (s *CaptureTestSuite) TestDumpKeepsAPresetVerbatim() {
	// A preset comes back as an opaque run of bytes that MessagePack's string
	// type carries, and it is not valid UTF-8. Encoding it as JSON would
	// replace every byte above 0x7f, which destroys the offsets and model
	// numbers the capture exists to study.
	path := filepath.Join(s.T().TempDir(), "preset.bin")
	s.T().Setenv(dumpEnv, path)

	raw := "\xa9l6-helix\x00\xda\x000\xff\xfe"
	s.Require().NoError(dump(raw))

	body, err := os.ReadFile(path) //nolint:gosec // a path this test wrote
	s.Require().NoError(err)
	s.Require().Equal(raw, string(body))
}

func (s *CaptureTestSuite) TestDumpKeepsBytesVerbatim() {
	path := filepath.Join(s.T().TempDir(), "preset.bin")
	s.T().Setenv(dumpEnv, path)

	s.Require().NoError(dump([]byte{0xa9, 0x00, 0xff}))

	body, err := os.ReadFile(path) //nolint:gosec // a path this test wrote
	s.Require().NoError(err)
	s.Require().Equal([]byte{0xa9, 0x00, 0xff}, body)
}

func (s *CaptureTestSuite) TestDumpReportsAnUnwritablePath() {
	s.T().Setenv(dumpEnv, filepath.Join(s.T().TempDir(), "no", "such", "dir.json"))

	s.Require().Error(dump(map[string]any{}))
}

func (s *CaptureTestSuite) TestDumpReportsSomethingItCannotEncode() {
	s.T().Setenv(dumpEnv, filepath.Join(s.T().TempDir(), "reply.json"))

	// A device answers with maps keyed by integers, which JSON has no way to
	// represent. Saying so beats writing a file that silently lost them.
	s.Require().Error(dump(map[any]any{1: "one"}))
}

func (s *CaptureTestSuite) TestDescribeReportsWhatArrived() {
	for _, tc := range []struct {
		name string
		got  any
		want string
	}{
		{"a document", map[any]any{1: "a", 2: "b"}, "map with 2 keys"},
		{"a blob", []byte{1, 2, 3}, "3 bytes"},
		{"a preset, which arrives as opaque bytes", "l6-helix\x00\xff", "10 bytes"},
		{"something else entirely", 42, "int"},
	} {
		s.Run(tc.name, func() {
			var out bytes.Buffer

			s.Require().NoError(describe(&out, "HX Stomp", 3, tc.got))
			s.Require().Contains(out.String(), tc.want)
			s.Require().Contains(out.String(), "slot 02A")
		})
	}
}

func (s *CaptureTestSuite) TestDescribeReportsAFailingWriter() {
	s.Require().Error(describe(&brokenWriter{}, "HX Stomp", 0, nil))
}

func TestCaptureTestSuite(t *testing.T) {
	suite.Run(t, new(CaptureTestSuite))
}
