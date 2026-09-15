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

package slots_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/retr0h/tonestack/pkg/sdk/internal/device"
	"github.com/retr0h/tonestack/pkg/sdk/internal/device/mocks"
	"github.com/retr0h/tonestack/pkg/sdk/internal/wire"
	"github.com/retr0h/tonestack/pkg/sdk/preset"
	slotpkg "github.com/retr0h/tonestack/pkg/sdk/slot"
)

// ImportDevicePublicTestSuite covers putting a preset file on a device.
//
// Nothing here reaches hardware. What it establishes is that the document
// leaving for the device holds the chain the file describes, checked by
// reading the bytes back with the decoder the reading commands use.
type ImportDevicePublicTestSuite struct {
	suite.Suite

	ctrl *gomock.Controller
	dev  *writable
}

// writable is a session that can both read and write.
type writable struct {
	*mocks.MockEditor
	*mocks.MockWriter
}

func (w *writable) Close() error { return nil }

func (s *ImportDevicePublicTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.dev = &writable{
		MockEditor: mocks.NewMockEditor(s.ctrl),
		MockWriter: mocks.NewMockWriter(s.ctrl),
	}
}

func (s *ImportDevicePublicTestSuite) TearDownTest() { s.ctrl.Finish() }

// preset is a .hlx the corpus carries, with a real chain in it.
func (s *ImportDevicePublicTestSuite) answer() []byte {
	raw, err := os.ReadFile(
		filepath.Join("..", "wire", "testdata", "preset.bin"))
	s.Require().NoError(err)

	return raw
}

func (s *ImportDevicePublicTestSuite) preset() string {
	return filepath.Join("..", "compile", "testdata", "preset0.hlx")
}

// unknownGear writes a preset naming a model no catalog carries.
func (s *ImportDevicePublicTestSuite) unknownGear() string {
	path := filepath.Join(s.T().TempDir(), "unknown.hlx")
	s.Require().NoError(os.WriteFile(path, []byte(`{
	  "version": 6,
	  "data": {"device": 2162689, "tone": {"dsp0": {"block0": {
	    "@model": "HD2_NoSuchThing", "@position": 1, "@enabled": true
	  }}}, "meta": {"name": "Nowhere"}},
	  "schema": "L6Preset"
	}`), 0o600))

	return path
}

// crowded writes a preset whose blocks reach past the eight positions a
// device gives a path.
func (s *ImportDevicePublicTestSuite) crowded() string {
	blocks := make([]string, 0, 17)
	for i := range 17 {
		blocks = append(blocks, fmt.Sprintf(
			`"block%d": {"@model": "HD2_DistTeemah", "@position": %d, "@enabled": true}`,
			i, i))
	}

	path := filepath.Join(s.T().TempDir(), "crowded.hlx")
	s.Require().NoError(os.WriteFile(path, fmt.Appendf(nil, `{
	  "version": 6,
	  "data": {"device": 2162689, "tone": {"dsp0": {%s}}, "meta": {"name": "Crowded"}},
	  "schema": "L6Preset"
	}`, strings.Join(blocks, ",")), 0o600))

	return path
}

// TestImportWith puts a preset file into a slot on a session.
func (s *ImportDevicePublicTestSuite) TestImportWith() {
	tests := []struct {
		name string
		// which file to import: a corpus preset unless a case says otherwise.
		file    string
		catalog string
		// what the device does with the write.
		writes  bool
		refuses bool
		// a session that can read but not write.
		readOnly bool
		// what the destination slot answers when it is read to be kept.
		// Empty means it holds nothing.
		destination string
		// the listing that names the destination is refused.
		unlisted bool

		// the document that left for the device must hold the file's chain.
		sent     bool
		contains []string
		// the name the kept copy of the destination carries.
		keptAs string
		// how many backups the error names.
		keptInErr int
		errText   string
	}{
		{
			name:     "the chain a file describes",
			writes:   true,
			sent:     true,
			contains: []string{"03B", "written"},
		},
		{
			// Kept under the name the device gives it, which is what
			// somebody looking for it later will search for.
			name:        "a destination holding a preset",
			writes:      true,
			destination: "held",
			keptAs:      "Minor Threat",
		},
		{
			name:     "a destination it cannot name",
			unlisted: true,
			errText:  "listing presets",
		},
		{
			name:    "a preset file that is not there",
			file:    "nowhere.hlx",
			errText: "nowhere.hlx",
		},
		{
			name:    "a catalog that is not there",
			catalog: "nowhere.json",
			errText: "nowhere.json",
		},
		{
			name:    "a device that refuses the write",
			refuses: true,
			errText: "writing slot 03B",
		},
		{
			// What the slot held is already on disk by then, and only the
			// error is left to say where.
			name:        "a device that refuses the write after the slot was kept",
			refuses:     true,
			destination: "held",
			errText:     "writing slot 03B",
			keptInErr:   1,
		},
		{
			// A reading session must not be handed the ability to overwrite
			// somebody's work.
			name:     "a session that cannot write",
			readOnly: true,
			errText:  "cannot write",
		},
		{
			// The destination is read so that what it held is kept. A device
			// that will not say what is there is one whose slot cannot be
			// replaced safely.
			name:        "a destination it cannot read",
			destination: "refused",
			errText:     "before replacing it",
		},
		{
			name:    "gear the model table does not carry",
			file:    "unknown",
			errText: "does not carry",
		},
		{
			name:    "a chain that runs into the routing",
			file:    "crowded",
			errText: "the device keeps its routing there",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var sent []byte

			switch {
			case tt.writes:
				s.dev.MockWriter.EXPECT().
					WriteNamedPreset(gomock.Any(), 0, 7, gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _, _ int, _ string, doc []byte) error {
						sent = doc

						return nil
					})
			case tt.refuses:
				s.dev.MockWriter.EXPECT().
					WriteNamedPreset(
						gomock.Any(), gomock.Any(), gomock.Any(),
						gomock.Any(), gomock.Any()).
					Return(errors.New("the device said no"))
			}

			file := s.preset()

			switch tt.file {
			case "":
			case "unknown":
				file = s.unknownGear()
			case "crowded":
				file = s.crowded()
			default:
				file = tt.file
			}

			dev := device.Editor(s.dev)
			if tt.readOnly {
				dev = mocks.NewMockEditor(s.ctrl)
			}

			// The destination is read before it is replaced, so that what it
			// held is kept. An empty answer is a slot with nothing in it,
			// which is nothing to lose rather than a reason to stop.
			// Exactly once, and only where the write gets far enough to
			// reach it. The controller is shared across these rows, so an
			// expectation left standing would answer a later one's call.
			if tt.unlisted {
				s.dev.MockEditor.EXPECT().Presets(gomock.Any(), 0).
					Return(nil, errors.New("boom"))
			}

			if tt.writes || tt.refuses || tt.destination != "" {
				s.dev.MockEditor.EXPECT().Presets(gomock.Any(), 0).
					Return([]wire.Preset{{Slot: 7, Name: "Minor Threat"}}, nil)

				switch tt.destination {
				case "refused":
					s.dev.MockEditor.EXPECT().
						ReadPreset(gomock.Any(), 0, 7).
						Return(nil, errors.New("boom"))
				case "held":
					s.dev.MockEditor.EXPECT().
						ReadPreset(gomock.Any(), 0, 7).
						Return(s.answer(), nil)
				default:
					s.dev.MockEditor.EXPECT().
						ReadPreset(gomock.Any(), 0, 7).
						Return(nil, nil)
				}
			}

			f := flows(s.T(), tt.catalog)
			f.BackupDir = s.T().TempDir()

			change, err := f.ImportWith(s.T().Context(), dev, file, slotpkg.Address{Slot: 7})

			if tt.errText != "" {
				s.Require().Error(err)
				s.Require().Contains(err.Error(), tt.errText)
				requireKept(s.Require(), err, tt.keptInErr)

				return
			}

			s.Require().NoError(err)

			for _, want := range tt.contains {
				s.Require().Contains(did(change), want)
			}

			if tt.keptAs != "" {
				s.Require().Len(change.Kept, 1)

				raw, err := os.ReadFile(change.Kept[0]) //nolint:gosec // a path this test chose
				s.Require().NoError(err)

				doc, err := preset.Read(bytes.NewReader(raw))
				s.Require().NoError(err)
				s.Require().Equal(tt.keptAs, doc.Data.Meta.Name)
			}

			if !tt.sent {
				return
			}

			got, err := wire.DecodePreset(sent)
			s.Require().NoError(err)
			s.Require().NotEmpty(got.Blocks, "the file's chain reached the device")
			s.Require().Len(got.Snapshots, 3, "and the snapshots came with it")
			s.Require().NotEmpty(got.Routing, "and the routing the blank carried")
		})
	}
}

func TestImportDevicePublicTestSuite(t *testing.T) {
	suite.Run(t, new(ImportDevicePublicTestSuite))
}
