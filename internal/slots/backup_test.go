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
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/vmihailenco/msgpack/v5"
	"go.uber.org/mock/gomock"

	"github.com/retr0h/tonestack/pkg/preset"
	"github.com/retr0h/tonestack/pkg/sdk/mocks"
)

// BackupTestSuite covers keeping what a slot held before replacing it.
//
// A device has no undo, so the value of every one of these is that somebody
// gets their preset back. The round trip is the test that matters: what is
// written has to be a preset this tool can read and put back.
type BackupTestSuite struct {
	suite.Suite

	ctrl *gomock.Controller
}

func (s *BackupTestSuite) SetupTest() { s.ctrl = gomock.NewController(s.T()) }

func (s *BackupTestSuite) TearDownTest() { s.ctrl.Finish() }

// answer returns one slot as an HX Stomp sent it.
func (s *BackupTestSuite) answer() []byte {
	raw, err := os.ReadFile(
		filepath.Join("..", "..", "pkg", "sdk", "wire", "testdata", "preset.bin"))
	s.Require().NoError(err)

	return raw
}

// emptied returns a preset with nothing on the grid, as the device sends one.
func (s *BackupTestSuite) emptied() []byte {
	var buf bytes.Buffer

	enc := msgpack.NewEncoder(&buf)
	s.Require().NoError(enc.EncodeString("l6-helix\x00"))
	s.Require().NoError(enc.EncodeString("offsets"))
	s.Require().NoError(enc.Encode(map[int8]any{0: map[int8]any{22: []any{}}}))

	return buf.Bytes()
}

// TestBackupDir covers where a backup goes when nobody says.
func (s *BackupTestSuite) TestBackupDir() {
	tests := []struct {
		name    string
		named   string
		state   string
		noHome  bool
		want    []string
		errText string
	}{
		{
			name:  "somewhere the caller named",
			named: filepath.Join("some", "where"),
			want:  []string{filepath.Join("some", "where")},
		},
		{
			name:  "the state directory, when there is one",
			state: filepath.Join("xdg", "state"),
			want:  []string{"xdg", "state", "tonestack", "presets"},
		},
		{
			name: "under the home directory, when there is not",
			want: []string{".local", "state", "tonestack", "presets"},
		},
		{
			name:    "nowhere to call home",
			noHome:  true,
			errText: "finding somewhere to keep a backup",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.T().Setenv("XDG_STATE_HOME", tt.state)

			if tt.noHome {
				s.T().Setenv("HOME", "")
			}

			got, err := backupDir(tt.named)

			if tt.errText != "" {
				s.Require().ErrorContains(err, tt.errText)

				return
			}

			s.Require().NoError(err)

			for _, want := range tt.want {
				s.Require().Contains(got, want)
			}
		})
	}
}

// TestBackup covers writing what a slot held to a file.
func (s *BackupTestSuite) TestBackup() {
	tests := []struct {
		name string
		body func() []byte
		// a directory that cannot be written into.
		readOnly bool
		// a path that is a file, so no directory can be made under it.
		blocked bool
		kept    bool
		errText string
	}{
		{
			name: "a slot holding a preset",
			body: s.answer,
			kept: true,
		},
		{
			// Nothing to lose, and a file saying so would only leave
			// somebody wondering what it was for.
			name: "a slot holding nothing",
			body: func() []byte { return nil },
		},
		{
			// A slot the device answers for and has nothing in. There is
			// nothing to lose, so nothing is written and nothing complains.
			name: "a slot the device says is empty",
			body: s.emptied,
		},
		{
			name:    "an answer that is not a preset",
			body:    func() []byte { return []byte("not a preset at all") },
			errText: "reading slot",
		},
		{
			name:    "nowhere to put it",
			body:    s.answer,
			blocked: true,
			errText: "making room for a backup",
		},
		{
			name:     "a directory it cannot write into",
			body:     s.answer,
			readOnly: true,
			errText:  "writing",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			dir := s.T().TempDir()

			switch {
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

			path, err := backup(tt.body(), DeviceOptions{Slot: 7}, dir)

			if tt.errText != "" {
				s.Require().ErrorContains(err, tt.errText)

				return
			}

			s.Require().NoError(err)

			if !tt.kept {
				s.Require().Empty(path)

				return
			}

			// Named for the slot the pedal shows and the moment, so a second write does not
			// overwrite the copy taken before the first.
			s.Require().Contains(filepath.Base(path), "03B-")
			s.Require().True(strings.HasSuffix(path, ".hlx"))

			// The claim worth holding: what came out is a preset this tool
			// can read, which is what putting it back needs.
			f, err := os.Open(path) //nolint:gosec // a path this test chose
			s.Require().NoError(err)

			defer func() { s.Require().NoError(f.Close()) }()

			doc, err := preset.Read(f)
			s.Require().NoError(err)
			s.Require().NotEmpty(doc.Data.Meta.Name)
		})
	}
}

// TestSaid covers naming the file a slot's old contents went to.
func (s *BackupTestSuite) TestSaid() {
	tests := []struct {
		name  string
		kept  []string
		deaf  bool
		wants []string
		quiet bool
	}{
		{
			name:  "one file",
			kept:  []string{"/tmp/one.hlx"},
			wants: []string{"kept", "/tmp/one.hlx"},
		},
		{
			name:  "two, because a swap replaces two slots",
			kept:  []string{"/tmp/one.hlx", "/tmp/two.hlx"},
			wants: []string{"/tmp/one.hlx", "/tmp/two.hlx"},
		},
		{
			name:  "a slot that held nothing",
			kept:  []string{""},
			quiet: true,
		},
		{name: "nothing kept at all", quiet: true},
		{name: "a reader nobody can write to", kept: []string{"/tmp/x.hlx"}, deaf: true},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var buf bytes.Buffer

			w := io.Writer(&buf)
			if tt.deaf {
				w = &failing{}
			}

			err := said(w, tt.kept...)

			if tt.deaf {
				s.Require().Error(err)

				return
			}

			s.Require().NoError(err)

			if tt.quiet {
				s.Require().Empty(buf.String())

				return
			}

			for _, want := range tt.wants {
				s.Require().Contains(buf.String(), want)
			}
		})
	}
}

// TestHolds covers reading a slot that may hold nothing.
func (s *BackupTestSuite) TestHolds() {
	tests := []struct {
		name    string
		body    []byte
		err     error
		errText string
	}{
		{name: "a slot with a preset in it", body: []byte{1, 2, 3}},
		{name: "a slot with nothing in it"},
		{
			name:    "a device that will not say",
			err:     errors.New("boom"),
			errText: "reading slot 01A before replacing it",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			dev := mocks.NewMockEditor(s.ctrl)
			dev.EXPECT().ReadPreset(gomock.Any(), 0, 0).Return(tt.body, tt.err)

			got, err := holds(context.Background(), dev, 0, 0)

			if tt.errText != "" {
				s.Require().ErrorContains(err, tt.errText)

				return
			}

			s.Require().NoError(err)
			s.Require().Equal(tt.body, got)
		})
	}
}

// TestKeep covers backing up one slot an edit is about to replace.
func (s *BackupTestSuite) TestKeep() {
	dir := s.T().TempDir()

	got, err := keep(nil, EditOptions{BackupDir: dir}, 3)
	s.Require().NoError(err)
	s.Require().Empty(got, "a slot holding nothing contributes nothing")

	got, err = keep(s.answer(), EditOptions{BackupDir: dir}, 3)
	s.Require().NoError(err)
	s.Require().Len(got, 1)

	_, err = keep(s.answer(), EditOptions{
		BackupDir: dir, CatalogPath: filepath.Join("testdata", "nope.json"),
	}, 3)
	s.Require().Error(err)
}

// failing is a writer nothing can be written to. Standing in for a standard
// library interface, so it is written by hand.
type failing struct{}

func (*failing) Write([]byte) (int, error) { return 0, errors.New("no") }

func TestBackupTestSuite(t *testing.T) {
	suite.Run(t, new(BackupTestSuite))
}
