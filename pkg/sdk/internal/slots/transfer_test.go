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
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/preset"
	"github.com/retr0h/tonestack/pkg/sdk/result"
)

// TransferTestSuite covers putting a reading on disk.
type TransferTestSuite struct {
	suite.Suite
}

// TestWrite covers the failures an export reports rather than writing a file
// that is not a preset.
func (s *TransferTestSuite) TestWrite() {
	tests := []struct {
		name    string
		read    result.Reading
		is      error
		errText string
	}{
		{
			name: "a reading with no document",
			is:   ErrEmptySlot,
		},
		{
			name:    "a document that will not encode",
			read:    result.Reading{Doc: &preset.Document{Meta: json.RawMessage("{")}},
			errText: "encoding preset",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			path := filepath.Join(s.T().TempDir(), "out.hlx")

			_, err := write(tt.read, ExportOptions{As: FormatPreset, OutputPath: path})

			s.Require().Error(err)

			if tt.is != nil {
				s.Require().ErrorIs(err, tt.is)
			}

			if tt.errText != "" {
				s.Require().ErrorContains(err, tt.errText)
			}

			s.Require().NoFileExists(path)
		})
	}
}

func TestTransferTestSuite(t *testing.T) {
	suite.Run(t, new(TransferTestSuite))
}
