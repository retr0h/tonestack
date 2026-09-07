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

	"github.com/retr0h/tonestack/internal/slots"
	"github.com/retr0h/tonestack/pkg/sdk"
	"github.com/retr0h/tonestack/pkg/sdk/mocks"
	"github.com/retr0h/tonestack/pkg/sdk/wire"
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
func (s *ImportDevicePublicTestSuite) preset() string {
	return filepath.Join("..", "lift", "testdata", "preset0.hlx")
}

func (s *ImportDevicePublicTestSuite) TestImportWritesTheChainToTheSlot() {
	var sent []byte

	s.dev.MockWriter.EXPECT().
		WriteNamedPreset(gomock.Any(), 0, 7, gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _, _ int, _ string, doc []byte) error {
			sent = doc

			return nil
		})

	var out bytes.Buffer

	s.Require().NoError(slots.ImportWith(
		s.T().Context(), &out, s.dev,
		slots.ImportOptions{File: s.preset(), Slot: 7}))

	got, err := wire.DecodePreset(sent)
	s.Require().NoError(err)
	s.Require().NotEmpty(got.Blocks, "the file's chain reached the device")
	s.Require().Len(got.Snapshots, 3, "and the snapshots came with it")
	s.Require().NotEmpty(got.Routing, "and the routing the blank carried")

	s.Require().Contains(out.String(), "03B")
	s.Require().Contains(out.String(), "written")
}

func (s *ImportDevicePublicTestSuite) TestImportReportsWhatStopsIt() {
	tests := []struct {
		name   string
		opts   slots.ImportOptions
		expect func()
		want   string
	}{
		{
			name: "a preset file that is not there",
			opts: slots.ImportOptions{File: "nowhere.hlx", Slot: 7},
			want: "nowhere.hlx",
		},
		{
			name: "a catalog that is not there",
			opts: slots.ImportOptions{
				File:        s.preset(),
				Slot:        7,
				CatalogPath: "nowhere.json",
			},
			want: "nowhere.json",
		},
		{
			name: "a device that refuses the write",
			opts: slots.ImportOptions{File: s.preset(), Slot: 7},
			expect: func() {
				s.dev.MockWriter.EXPECT().
					WriteNamedPreset(
						gomock.Any(), gomock.Any(), gomock.Any(),
						gomock.Any(), gomock.Any()).
					Return(errors.New("the device said no"))
			},
			want: "writing slot 03B",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			if tt.expect != nil {
				tt.expect()
			}

			err := slots.ImportWith(
				s.T().Context(), &bytes.Buffer{}, s.dev, tt.opts)

			s.Require().Error(err)
			s.Require().Contains(err.Error(), tt.want)
		})
	}
}

// TestImportNeedsASessionThatCanWrite keeps a reading session from being
// handed the ability to overwrite somebody's work.
func (s *ImportDevicePublicTestSuite) TestImportNeedsASessionThatCanWrite() {
	err := slots.ImportWith(
		s.T().Context(), &bytes.Buffer{}, mocks.NewMockEditor(s.ctrl),
		slots.ImportOptions{File: s.preset(), Slot: 7})

	s.Require().ErrorContains(err, "cannot write")
}

// TestImportReportsAFailingWriter covers somewhere for the result to go.
func (s *ImportDevicePublicTestSuite) TestImportReportsAFailingWriter() {
	s.dev.MockWriter.EXPECT().
		WriteNamedPreset(
			gomock.Any(), gomock.Any(), gomock.Any(),
			gomock.Any(), gomock.Any()).
		Return(nil)

	s.Require().Error(slots.ImportWith(
		s.T().Context(), &brokenWriter{}, s.dev,
		slots.ImportOptions{File: s.preset(), Slot: 7}))
}

// TestImportFindsItsOwnDevice covers the entry point somebody actually runs.
//
// One line: find a session, hand it on, release it. The only line in this
// file that needs hardware, so the session is stood in for.
func (s *ImportDevicePublicTestSuite) TestImportFindsItsOwnDevice() {
	s.dev.MockWriter.EXPECT().
		WriteNamedPreset(
			gomock.Any(), gomock.Any(), gomock.Any(),
			gomock.Any(), gomock.Any()).
		Return(nil)

	restore := *slots.OpenDevice
	*slots.OpenDevice = func(context.Context) (sdk.Editor, error) {
		return s.dev, nil
	}

	defer func() { *slots.OpenDevice = restore }()

	var out bytes.Buffer

	s.Require().NoError(slots.ImportDevice(
		s.T().Context(), &out,
		slots.ImportOptions{File: s.preset(), Slot: 7}))

	s.Require().Contains(out.String(), "written")
}

// TestImportReportsNoDeviceAttached covers finding none.
func (s *ImportDevicePublicTestSuite) TestImportReportsNoDeviceAttached() {
	restore := *slots.OpenDevice
	*slots.OpenDevice = func(context.Context) (sdk.Editor, error) {
		return nil, errors.New("nothing on the bus")
	}

	defer func() { *slots.OpenDevice = restore }()

	s.Require().ErrorContains(slots.ImportDevice(
		s.T().Context(), &bytes.Buffer{},
		slots.ImportOptions{File: s.preset(), Slot: 7}), "nothing on the bus")
}

// TestImportReportsAPresetItCannotWrite covers a file naming gear the
// catalog's model table does not carry.
func (s *ImportDevicePublicTestSuite) TestImportReportsAPresetItCannotWrite() {
	path := filepath.Join(s.T().TempDir(), "unknown.hlx")
	s.Require().NoError(os.WriteFile(path, []byte(`{
	  "version": 6,
	  "data": {"device": 2162689, "tone": {"dsp0": {"block0": {
	    "@model": "HD2_NoSuchThing", "@position": 1, "@enabled": true
	  }}}, "meta": {"name": "Nowhere"}},
	  "schema": "L6Preset"
	}`), 0o600))

	err := slots.ImportWith(
		s.T().Context(), &bytes.Buffer{}, s.dev,
		slots.ImportOptions{File: path, Slot: 7})

	s.Require().ErrorContains(err, "does not carry")
}

// TestImportReportsAChainThatRunsIntoTheRouting covers a file whose blocks
// reach past the eight positions a device gives a path.
func (s *ImportDevicePublicTestSuite) TestImportReportsAChainThatRunsIntoTheRouting() {
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

	err := slots.ImportWith(
		s.T().Context(), &bytes.Buffer{}, s.dev,
		slots.ImportOptions{File: path, Slot: 7})

	s.Require().ErrorContains(err, "the device keeps its routing there")
}

type brokenWriter struct{}

func (*brokenWriter) Write([]byte) (int, error) { return 0, errors.New("boom") }

func TestImportDevicePublicTestSuite(t *testing.T) {
	suite.Run(t, new(ImportDevicePublicTestSuite))
}
