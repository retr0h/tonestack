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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/internal/setlist"
)

// OpenTestSuite covers putting a setlist back on disk.
type OpenTestSuite struct {
	suite.Suite
}

// TestSave covers writing a setlist, and leaving what was there alone when
// that fails.
func (s *OpenTestSuite) TestSave() {
	tests := []struct {
		name    string
		doc     *setlist.Document
		errText string
	}{
		{
			// A setlist file holds exactly one setlist. One that will not
			// encode is reported, and the file already there is not replaced
			// by an empty one.
			name:    "a document that will not encode",
			doc:     &setlist.Document{},
			errText: "holds one setlist",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			path := filepath.Join(s.T().TempDir(), "out.hls")
			s.Require().NoError(os.WriteFile(path, []byte("what was there"), 0o600))

			err := save(path, tt.doc)

			s.Require().ErrorContains(err, tt.errText)

			got, readErr := os.ReadFile(path) //nolint:gosec // a path this test chose
			s.Require().NoError(readErr)
			s.Require().Equal("what was there", string(got))
		})
	}
}

func TestOpenTestSuite(t *testing.T) {
	suite.Run(t, new(OpenTestSuite))
}
