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

package backup_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/retr0h/tonestack/pkg/sdk/internal/backup"
	"github.com/retr0h/tonestack/pkg/sdk/internal/backup/mocks"
	"github.com/retr0h/tonestack/pkg/sdk/preset"
	"github.com/retr0h/tonestack/pkg/sdk/slot"
)

// BackupPublicTestSuite is the backup policy, one row per rule.
//
// The decoder is a double, so each row says exactly what the device's answer
// read as and the policy is all that is under test. Reading a real answer is
// the device flows' business, and their suites do it.
type BackupPublicTestSuite struct {
	suite.Suite
}

// decoded is what the double says one answer reads as.
type decoded struct {
	// blank is a preset that decodes, with a chain in it.
	blank bool
	// unencodable is a preset that will not encode again.
	unencodable bool
	err         error
}

// document is the preset a row's answer reads as. Nil is a slot with no
// blocks.
func (s *BackupPublicTestSuite) document(
	d decoded,
) *preset.Document {
	switch {
	case d.unencodable:
		return &preset.Document{Meta: json.RawMessage("{")}
	case d.blank:
		doc, err := preset.Blank()
		s.Require().NoError(err)

		doc.Data.Meta.Name = "Black Rusty"

		return doc
	default:
		return nil
	}
}

// TestKeep covers what is kept, in which format, under which name, and where.
func (s *BackupPublicTestSuite) TestKeep() {
	body := []byte("what the device answered")

	tests := []struct {
		name string
		held []backup.Held
		// what the decoder says each answer reads as, in order. A row whose
		// slot the decoder is never asked about has fewer.
		decodes []decoded
		// where to keep them: a directory unless a row says otherwise.
		blocked  bool
		readOnly bool
		noHome   bool
		// the clock stands still, so every backup wants the same name.
		pinned bool

		// the extension of each file kept, in order. Empty means nothing is.
		kept []string
		// part of the first kept file's name.
		base    string
		is      error
		errText string
		// how many files a failed Keep leaves behind, each still the preset
		// it was written as.
		intact int
	}{
		{
			// Rule 1. Nothing to lose, and a file saying so would only leave
			// somebody wondering what it was for. The decoder is not asked.
			name: "a slot the device answered nothing for",
			held: []backup.Held{{At: slot.Address{Slot: 3}, Name: "Black Rusty"}},
		},
		{
			// Rule 2. A blank slot nobody has used. Keeping it would fill the
			// backup directory with copies of nothing.
			name:    "no blocks in a slot still called New Preset",
			held:    []backup.Held{{Body: body, Name: "New Preset"}},
			decodes: []decoded{{}},
		},
		{
			// Rule 3, named for the slot the pedal shows, the setlist it is
			// in and the moment.
			name: "a slot that decodes",
			held: []backup.Held{{
				At: slot.Address{Setlist: 2, Slot: 7}, Name: "Black Rusty", Body: body,
			}},
			decodes: []decoded{{blank: true}},
			kept:    []string{".hlx"},
			base:    "03B-s2-",
		},
		{
			// Rule 3 still. A chain is somebody's, whatever the slot is called.
			name:    "blocks in a slot still called New Preset",
			held:    []backup.Held{{Body: body, Name: "New Preset"}},
			decodes: []decoded{{blank: true}},
			kept:    []string{".hlx"},
		},
		{
			// Rule 4. Somebody renamed it, so whatever is in it may be theirs.
			name:    "no blocks in a slot somebody renamed",
			held:    []backup.Held{{Body: body, Name: "Riff Ideas"}},
			decodes: []decoded{{}},
			kept:    []string{".bin"},
		},
		{
			// Rule 4. Nothing said what it is called, so nothing says it is
			// blank.
			name:    "no blocks in a slot no listing named",
			held:    []backup.Held{{Body: body}},
			decodes: []decoded{{}},
			kept:    []string{".bin"},
		},
		{
			// Rule 5. Named to the second, the second would have been written
			// over the first.
			name:    "the same slot twice within a second",
			held:    []backup.Held{{Body: body}, {Body: body}},
			decodes: []decoded{{blank: true}, {blank: true}},
			kept:    []string{".hlx", ".hlx"},
		},
		{
			// Rule 5, with the clock pinned so both backups want one name. The
			// second is refused rather than written over the first, and the
			// first is left as it was.
			name: "the same slot twice in the same instant",
			held: []backup.Held{
				{Body: body, At: slot.Address{Setlist: 1, Slot: 4}},
				{Body: body, At: slot.Address{Setlist: 1, Slot: 4}},
			},
			decodes: []decoded{{blank: true}, {blank: true}},
			pinned:  true,
			is:      fs.ErrExist,
			intact:  1,
		},
		{
			// A swap replaces two, and one holding nothing is left out of the
			// answer rather than reported as an empty path.
			name: "two slots, one of them holding nothing",
			held: []backup.Held{
				{Body: body, At: slot.Address{Slot: 3}},
				{At: slot.Address{Slot: 0}},
			},
			decodes: []decoded{{blank: true}},
			kept:    []string{".hlx"},
		},
		{
			// Reported rather than kept as bytes. By rule 6 the write stops.
			name:    "an answer that is not a preset",
			held:    []backup.Held{{Body: body, At: slot.Address{Slot: 7}}},
			decodes: []decoded{{err: errors.New("not a preset")}},
			errText: "reading slot 03B before replacing it: not a preset",
		},
		{
			// A backup that silently held a blank file would be found to be
			// useless only when it was needed.
			name:    "a preset that will not encode again",
			held:    []backup.Held{{Body: body}},
			decodes: []decoded{{unencodable: true}},
			errText: "encoding preset",
		},
		{
			// The first fails, so the second is not kept either and nothing
			// is reported as kept.
			name:    "two slots, the first of which will not read",
			held:    []backup.Held{{Body: body}, {Body: body}},
			decodes: []decoded{{err: errors.New("boom")}},
			errText: "boom",
		},
		{
			name:    "nowhere to put it",
			held:    []backup.Held{{Body: body}},
			decodes: []decoded{{blank: true}},
			blocked: true,
			errText: "making room for a backup",
		},
		{
			name:     "a directory it cannot write into",
			held:     []backup.Held{{Body: body}},
			decodes:  []decoded{{blank: true}},
			readOnly: true,
			errText:  "writing",
		},
		{
			// Nobody said where, and there is nowhere to work it out from.
			name:    "nowhere to work out where it goes",
			held:    []backup.Held{{Body: body}},
			decodes: []decoded{{blank: true}},
			noHome:  true,
			errText: "finding somewhere to keep a backup",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			dir := s.T().TempDir()

			switch {
			case tt.noHome:
				s.T().Setenv("XDG_STATE_HOME", "")
				s.T().Setenv("HOME", "")

				dir = ""
			case tt.blocked:
				dir = filepath.Join(dir, "a-file")
				s.Require().NoError(os.WriteFile(dir, []byte("x"), 0o600))

				dir = filepath.Join(dir, "under-it")
			case tt.readOnly:
				s.Require().NoError(os.Chmod(dir, 0o500))
				s.T().Cleanup(func() {
					s.Require().NoError(os.Chmod(dir, 0o700))
				})
			}

			decoder := mocks.NewMockDecoder(gomock.NewController(s.T()))

			var last *gomock.Call

			for i, d := range tt.decodes {
				h := tt.held[i]
				call := decoder.EXPECT().
					Document(gomock.Any(), h.Body, h.At, h.Name).
					Return(s.document(d), d.err)

				if last != nil {
					call.After(last)
				}

				last = call
			}

			keeper := backup.New(dir, decoder)

			if tt.pinned {
				at := time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)
				keeper.WithClock(func() time.Time { return at })
			}

			got, err := keeper.Keep(context.Background(), tt.held...)

			if tt.errText != "" || tt.is != nil {
				if tt.errText != "" {
					s.Require().ErrorContains(err, tt.errText)
				}

				if tt.is != nil {
					s.Require().ErrorIs(err, tt.is)
				}

				s.Require().Nil(got)

				if tt.intact == 0 {
					return
				}

				left, err := os.ReadDir(dir)
				s.Require().NoError(err)
				s.Require().Len(left, tt.intact)

				raw, err := os.ReadFile(filepath.Join(dir, left[0].Name()))
				s.Require().NoError(err)

				doc, err := preset.Read(bytes.NewReader(raw))
				s.Require().NoError(err)
				s.Require().Equal("Black Rusty", doc.Data.Meta.Name,
					"the first backup is still the preset it was written as")

				return
			}

			s.Require().NoError(err)
			s.Require().Len(got, len(tt.kept))

			for i, ext := range tt.kept {
				s.Require().True(strings.HasSuffix(got[i], ext), got[i])
				s.Require().Equal(dir, filepath.Dir(got[i]))

				raw, err := os.ReadFile(got[i])
				s.Require().NoError(err)

				if ext == ".bin" {
					s.Require().Equal(body, raw, "kept as the device's bytes")

					continue
				}

				// What came out is a preset this tool can read, which is what
				// putting it back needs.
				doc, err := preset.Read(bytes.NewReader(raw))
				s.Require().NoError(err)
				s.Require().Equal("Black Rusty", doc.Data.Meta.Name)
			}

			if len(got) == 2 {
				s.Require().NotEqual(got[0], got[1], "a backup never replaces another")
			}

			if tt.base != "" {
				s.Require().Contains(filepath.Base(got[0]), tt.base)
			}
		})
	}
}

func TestBackupPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(BackupPublicTestSuite))
}
