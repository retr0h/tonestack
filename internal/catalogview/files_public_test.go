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

package catalogview_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/internal/catalogview"
)

// FilesPublicTestSuite covers opening a catalog as something a caller can
// stand in for.
//
// Open is the package-level function of the same name, so what is asserted
// here is that reaching it through the value agrees with calling it directly.
type FilesPublicTestSuite struct {
	suite.Suite
}

// TestOpen covers reading a catalog through the value.
func (s *FilesPublicTestSuite) TestOpen() {
	tests := []struct {
		name string
		path string
		ok   bool
	}{
		{name: "the catalog in the binary", path: "", ok: true},
		{name: "one on disk", path: filepath.Join("testdata", "catalog.json"), ok: true},
		{name: "one that is not there", path: filepath.Join("testdata", "nope.json")},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := catalogview.Files{}.Open(tt.path)

			if !tt.ok {
				s.Require().Error(err)
				s.Require().Nil(got)

				return
			}

			want, err := catalogview.Open(tt.path)
			s.Require().NoError(err)
			s.Require().Equal(want, got)
		})
	}
}

func TestFilesPublicTestSuite(t *testing.T) {
	suite.Run(t, new(FilesPublicTestSuite))
}
