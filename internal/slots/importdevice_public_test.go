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
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/retr0h/tonestack/internal/slots"
	"github.com/retr0h/tonestack/pkg/sdk/device"
	"github.com/retr0h/tonestack/pkg/sdk/device/mocks"
	"github.com/retr0h/tonestack/pkg/sdk/device/wire"
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

func (w *writable) Close() {}

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
		filepath.Join("..", "..", "pkg", "sdk", "device", "wire", "testdata", "preset.bin"))
	s.Require().NoError(err)

	return raw
}

func (s *ImportDevicePublicTestSuite) preset() string {
	return filepath.Join("..", "..", "pkg", "sdk", "compile", "testdata", "preset0.hlx")
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

		// the document that left for the device must hold the file's chain.
		sent     bool
		contains []string
		errText  string
	}{
		{
			name:     "the chain a file describes",
			writes:   true,
			sent:     true,
			contains: []string{"03B", "written"},
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
			if tt.writes || tt.refuses || tt.destination != "" {
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

			change, err := slots.ImportWith(s.T().Context(), dev, slots.ImportOptions{
				File: file, Slot: 7, CatalogPath: tt.catalog,
				BackupDir: s.T().TempDir(),
			})

			if tt.errText != "" {
				s.Require().Error(err)
				s.Require().Contains(err.Error(), tt.errText)

				return
			}

			s.Require().NoError(err)

			for _, want := range tt.contains {
				s.Require().Contains(did(change), want)
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

// TestImportDevice covers the entry point somebody actually runs.
//
// One line: find a session, hand it on, release it. The only line in this
// file that needs hardware, so the session is stood in for.
func (s *ImportDevicePublicTestSuite) TestImportDevice() {
	tests := []struct {
		name     string
		attached bool
		contains string
		errText  string
	}{
		{name: "a device on the bus", attached: true, contains: "written"},
		{name: "nothing on the bus", errText: "nothing on the bus"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			restore := *slots.OpenDevice
			defer func() { *slots.OpenDevice = restore }()

			if tt.attached {
				// The destination is read first, so what it held is kept.
				s.dev.MockEditor.EXPECT().
					ReadPreset(gomock.Any(), 0, 7).
					Return(nil, nil)

				s.dev.MockWriter.EXPECT().
					WriteNamedPreset(
						gomock.Any(), gomock.Any(), gomock.Any(),
						gomock.Any(), gomock.Any()).
					Return(nil)

				*slots.OpenDevice = func(context.Context) (device.Editor, error) {
					return s.dev, nil
				}
			} else {
				*slots.OpenDevice = func(context.Context) (device.Editor, error) {
					return nil, errors.New("nothing on the bus")
				}
			}

			change, err := slots.ImportDevice(s.T().Context(),
				slots.ImportOptions{
					File: s.preset(), Slot: 7, BackupDir: s.T().TempDir(),
				})

			if tt.errText != "" {
				s.Require().ErrorContains(err, tt.errText)

				return
			}

			s.Require().NoError(err)
			s.Require().Contains(did(change), tt.contains)
		})
	}
}

func TestImportDevicePublicTestSuite(t *testing.T) {
	suite.Run(t, new(ImportDevicePublicTestSuite))
}
