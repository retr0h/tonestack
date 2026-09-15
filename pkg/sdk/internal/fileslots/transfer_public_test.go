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

package fileslots_test

import (
	"context"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/internal/fileslots"
	"github.com/retr0h/tonestack/pkg/sdk/preset"
	"github.com/retr0h/tonestack/pkg/sdk/result"
)

// TransferPublicTestSuite covers putting a reading on disk, which the device
// flows do through the same function.
type TransferPublicTestSuite struct {
	suite.Suite
}

// TestWrite covers a reading written in the format asked for.
func (s *TransferPublicTestSuite) TestWrite() {
	blank, err := preset.Blank()
	s.Require().NoError(err)

	tests := []struct {
		name string
		read result.Reading
		as   result.Format
		out  string
		// what the written file must say.
		contains string
		errText  string
		// a file somebody already has at out, and what the write does about it.
		taken    bool
		existing result.Existing
		is       error
	}{
		{
			name:     "over a file, replaced",
			read:     result.Reading{Name: "Blank", Doc: blank},
			as:       result.FormatPreset,
			out:      "one.hlx",
			contains: `"schema"`,
			taken:    true,
		},
		{
			// The write refuses it: the file is what the caller finds, not
			// something a look beforehand decided.
			name:     "over a file, kept",
			read:     result.Reading{Name: "Blank", Doc: blank},
			as:       result.FormatPreset,
			out:      "one.hlx",
			taken:    true,
			existing: result.KeepExisting,
			is:       fs.ErrExist,
		},
		{
			name:     "the device's own file",
			read:     result.Reading{Name: "Blank", Doc: blank},
			as:       result.FormatPreset,
			out:      "one.hlx",
			contains: `"schema"`,
		},
		{
			// Reported, rather than a file holding nothing where a preset was
			// meant to be.
			name:    "a preset that will not encode",
			read:    result.Reading{Doc: &preset.Document{Meta: json.RawMessage("{")}},
			as:      result.FormatPreset,
			out:     "broken.hlx",
			errText: "encoding preset",
		},
		{
			// A reading with no rig in it is not a rig, and the contract says
			// so on the way out.
			name:    "a rig with no chain",
			as:      result.FormatRig,
			out:     "empty.yaml",
			errText: "writing the rig",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			out := filepath.Join(s.T().TempDir(), tt.out)

			if tt.taken {
				s.Require().NoError(os.WriteFile(out, []byte("somebody's preset"), 0o600))
			}

			written, err := fileslots.Write(tt.read, 4, out, tt.as, tt.existing)

			if tt.is != nil {
				s.Require().ErrorIs(err, tt.is)

				raw, readErr := os.ReadFile(out) //nolint:gosec // a path this test chose
				s.Require().NoError(readErr)
				s.Require().Equal("somebody's preset", string(raw))

				return
			}

			if tt.errText != "" {
				s.Require().ErrorContains(err, tt.errText)
				s.Require().NoFileExists(out)

				return
			}

			s.Require().NoError(err)
			s.Require().Equal(result.Written{Slot: 4, Name: tt.read.Name, Path: out}, written)

			raw, err := os.ReadFile(out) //nolint:gosec // a path this test chose
			s.Require().NoError(err)
			s.Require().Contains(string(raw), tt.contains)

			// And it reads back as a preset, which is what writing one is for.
			_, err = fileslots.ReadPreset(context.Background(), out)
			s.Require().NoError(err)
		})
	}
}

func TestTransferPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(TransferPublicTestSuite))
}
