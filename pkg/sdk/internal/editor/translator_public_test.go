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

package editor_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/internal/editor"
	"github.com/retr0h/tonestack/pkg/sdk/internal/wire"
)

// TranslatorPublicTestSuite covers the package's work reached as a value.
//
// Each method is the package-level function of the same name, so what is
// asserted here is that calling it through the type and calling it directly
// agree. The behaviour itself is covered by the function's own suite.
type TranslatorPublicTestSuite struct {
	suite.Suite

	cat *catalog.Catalog
	got wire.DevicePreset
}

func (s *TranslatorPublicTestSuite) SetupSuite() {
	var err error

	s.cat, err = catalog.BuiltIn()
	s.Require().NoError(err)

	raw, err := os.ReadFile(
		filepath.Join("..", "wire", "testdata", "preset.bin"))
	s.Require().NoError(err)

	s.got, err = wire.DecodePreset(raw)
	s.Require().NoError(err)
}

// TestNew covers what a caller is handed.
func (s *TranslatorPublicTestSuite) TestNew() {
	s.Require().NotNil(editor.New())
}

// TestChain covers reading what a device laid out, through the type.
func (s *TranslatorPublicTestSuite) TestChain() {
	tests := []struct {
		name string
		got  wire.DevicePreset
	}{
		{name: "a slot the device answered with", got: s.got},
		{name: "a slot holding nothing", got: wire.DevicePreset{}},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			want, wantErr := editor.Chain("one", tt.got, s.cat)
			got, err := editor.New().Chain("one", tt.got, s.cat)

			s.Require().Equal(wantErr == nil, err == nil)
			s.Require().Equal(want, got)
		})
	}
}

// TestControllers covers naming what a controller moves, through the type.
func (s *TranslatorPublicTestSuite) TestControllers() {
	s.Require().Equal(
		editor.Controllers(s.got, s.cat),
		editor.New().Controllers(s.got, s.cat))
}

// TestDeviceState covers the routing a device wraps a chain in.
func (s *TranslatorPublicTestSuite) TestDeviceState() {
	s.Require().Equal(
		editor.DeviceState(s.got, s.cat),
		editor.New().DeviceState(s.got, s.cat))
}

// TestDocument covers building the preset a device describes, through the type.
func (s *TranslatorPublicTestSuite) TestDocument() {
	tests := []struct {
		name string
		got  wire.DevicePreset
	}{
		{name: "a slot the device answered with", got: s.got},
		{name: "a slot holding nothing", got: wire.DevicePreset{}},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			want, wantEmpty, wantErr := editor.Document(tt.got, s.cat, "one")
			got, empty, err := editor.New().Document(tt.got, s.cat, "one")

			s.Require().Equal(wantErr == nil, err == nil)
			s.Require().Equal(wantEmpty, empty)
			s.Require().Equal(want, got)
		})
	}
}

// TestFootswitches covers what the pedal prints, through the type.
func (s *TranslatorPublicTestSuite) TestFootswitches() {
	s.Require().Equal(
		editor.Footswitches(s.got, s.cat),
		editor.New().Footswitches(s.got, s.cat))
}

// TestPlacements covers turning a preset into what a device lays out.
func (s *TranslatorPublicTestSuite) TestPlacements() {
	doc, empty, err := editor.Document(s.got, s.cat, "one")
	s.Require().NoError(err)
	s.Require().False(empty)

	want, wantErr := editor.Placements(doc, s.cat)
	got, err := editor.New().Placements(doc, s.cat)

	s.Require().Equal(wantErr == nil, err == nil)
	s.Require().Equal(want, got)
}

// TestSnapshots covers what the device recalls on a footswitch.
func (s *TranslatorPublicTestSuite) TestSnapshots() {
	s.Require().Equal(
		editor.Snapshots(s.got),
		editor.New().Snapshots(s.got))
}

func TestTranslatorPublicTestSuite(t *testing.T) {
	suite.Run(t, new(TranslatorPublicTestSuite))
}
