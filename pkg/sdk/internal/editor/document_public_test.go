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
	riggen "github.com/retr0h/tonestack/pkg/sdk/internal/gen"
	"github.com/retr0h/tonestack/pkg/sdk/internal/wire"
)

// DocumentPublicTestSuite covers the rig fragments a device's answer carries.
type DocumentPublicTestSuite struct {
	suite.Suite

	cat *catalog.Catalog
}

func (s *DocumentPublicTestSuite) SetupSuite() {
	var err error

	s.cat, err = catalog.BuiltIn()
	s.Require().NoError(err)
}

// TestControllers covers naming what a controller moves, and the cases
// where it cannot be named.
func (s *DocumentPublicTestSuite) TestControllers() {
	// The block at grid position 2 in the captured preset is a volume pedal,
	// whose parameters are Pedal and VolumeTaper in that order.
	block := wire.DeviceBlock{Index: 2, Model: 261}

	tests := []struct {
		name  string
		got   wire.DevicePreset
		want  *[]riggen.Controller
		named string
	}{
		{
			name: "an expression pedal on a parameter",
			got: wire.DevicePreset{
				Blocks: []wire.DeviceBlock{block},
				Controllers: []wire.DeviceController{
					{Controller: 2, Block: 2, Param: 0, Min: 0, Max: 1},
				},
			},
			named: "Pedal",
		},
		{
			name: "one snapshots leave alone",
			got: wire.DevicePreset{
				Blocks: []wire.DeviceBlock{block},
				Controllers: []wire.DeviceController{
					{Controller: 2, Block: 2, Param: 1, NoSnapshot: true},
				},
			},
			named: "VolumeTaper",
		},
		{
			name: "a preset assigning none",
			got:  wire.DevicePreset{Blocks: []wire.DeviceBlock{block}},
		},
		{
			name: "one naming a block the chain does not hold",
			got: wire.DevicePreset{
				Blocks: []wire.DeviceBlock{block},
				Controllers: []wire.DeviceController{
					{Controller: 2, Block: 9, Param: 0},
				},
			},
		},
		{
			name: "one naming a parameter past what the model has",
			got: wire.DevicePreset{
				Blocks: []wire.DeviceBlock{block},
				Controllers: []wire.DeviceController{
					{Controller: 2, Block: 2, Param: 40},
				},
			},
		},
		{
			name: "one on a model this catalog cannot name",
			got: wire.DevicePreset{
				Blocks: []wire.DeviceBlock{{Index: 2, Model: 999999}},
				Controllers: []wire.DeviceController{
					{Controller: 2, Block: 2, Param: 0},
				},
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got := editor.Controllers(tt.got, s.cat)

			if tt.named == "" {
				s.Require().Nil(got, "nothing that cannot be named is written")

				return
			}

			s.Require().NotNil(got)
			s.Require().Len(*got, 1)
			s.Require().Equal(tt.named, (*got)[0].Parameter)

			// A grid position is not a place along the path, and a rig counts
			// the way the chain does.
			s.Require().Equal(1, (*got)[0].Block)
		})
	}
}

// capture returns one slot as an HX Stomp actually answered, decoded.
//
// pkg/sdk/device/wire/testdata/preset.bin is what the hardware handed back, so the
// path from a real answer to a preset runs here without a device on the bus.
func (s *DocumentPublicTestSuite) capture() wire.DevicePreset {
	raw, err := os.ReadFile(
		filepath.Join("..", "wire", "testdata", "preset.bin"))
	s.Require().NoError(err)

	got, err := wire.DecodePreset(raw)
	s.Require().NoError(err)

	return got
}

// TestDocument covers building the preset a device's answer describes.
func (s *DocumentPublicTestSuite) TestDocument() {
	tests := []struct {
		name    string
		got     func() wire.DevicePreset
		empty   bool
		wantErr string
	}{
		{
			name: "a slot the device answered with",
			got:  s.capture,
		},
		{
			name:  "a slot holding nothing",
			got:   func() wire.DevicePreset { return wire.DevicePreset{} },
			empty: true,
		},
		{
			name: "a model this catalog cannot name",
			got: func() wire.DevicePreset {
				return wire.DevicePreset{
					Blocks: []wire.DeviceBlock{{Index: 2, Model: 999999}},
				}
			},
			wantErr: "does not reach",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			doc, empty, err := editor.Document(tt.got(), s.cat, "slot 01A")

			if tt.wantErr != "" {
				s.Require().ErrorContains(err, tt.wantErr)

				return
			}

			s.Require().NoError(err)
			s.Require().Equal(tt.empty, empty)

			if tt.empty {
				s.Require().Nil(doc)

				return
			}

			s.Require().Equal("slot 01A", doc.Data.Meta.Name)
			s.Require().Equal(s.cat.DeviceID, doc.Data.Device)

			// The chain the device described, not the blank it was written
			// into.
			c, err := doc.Spec()
			s.Require().NoError(err)
			s.Require().NotEmpty(c.Blocks)
		})
	}
}

// TestSnapshots covers what the device recalls on a footswitch.
func (s *DocumentPublicTestSuite) TestSnapshots() {
	tests := []struct {
		name  string
		got   wire.DevicePreset
		want  int
		named string
		tempo bool
	}{
		{
			name: "a preset with none",
			got:  wire.DevicePreset{},
		},
		{
			name: "one the player named, with a tempo",
			got: wire.DevicePreset{Snapshots: []wire.DeviceSnapshot{
				{Name: "Verse", Tempo: 120, LED: 2, Valid: true},
			}},
			want: 1, named: "Verse", tempo: true,
		},
		{
			name: "one carrying neither",
			got: wire.DevicePreset{Snapshots: []wire.DeviceSnapshot{
				{LED: 1, Valid: false},
			}},
			want: 1,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got := editor.Snapshots(tt.got)

			if tt.want == 0 {
				s.Require().Nil(got)

				return
			}

			s.Require().NotNil(got)
			s.Require().Len(*got, tt.want)

			one := (*got)[0]
			s.Require().NotNil(one.Led)
			s.Require().NotNil(one.Valid)

			if tt.named == "" {
				s.Require().Nil(one.Name)
			} else {
				s.Require().Equal(tt.named, *one.Name)
			}

			s.Require().Equal(tt.tempo, one.Tempo != nil)
		})
	}
}

// TestFootswitches covers what the pedal prints under each switch.
func (s *DocumentPublicTestSuite) TestFootswitches() {
	tests := []struct {
		name     string
		got      wire.DevicePreset
		want     int
		coloured bool
	}{
		{
			name: "a preset assigning none",
			got:  wire.DevicePreset{},
		},
		{
			name: "one on a colour the catalog names",
			got: wire.DevicePreset{Footswitches: []wire.DeviceFootswitch{
				{Switch: 1, Label: "Drive", Gear: "Minotaur", Block: 3, LED: 0},
			}},
			want: 1, coloured: true,
		},
		{
			name: "one on a colour this catalog does not know",
			got: wire.DevicePreset{Footswitches: []wire.DeviceFootswitch{
				{Switch: 2, Label: "Verb", Gear: "Hall", Block: 4, LED: 999},
			}},
			want: 1,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got := editor.Footswitches(tt.got, s.cat)

			if tt.want == 0 {
				s.Require().Nil(got)

				return
			}

			s.Require().NotNil(got)
			s.Require().Len(*got, tt.want)

			one := (*got)[0]

			// A grid position is not a place along the path, and a rig counts
			// the way the chain does.
			s.Require().Equal(
				tt.got.Footswitches[0].Block-wire.GridOffset, *one.Block)
			s.Require().Equal(tt.got.Footswitches[0].Label, *one.Label)

			// A colour a device knows and this catalog does not gets no name
			// rather than a wrong one.
			s.Require().Equal(tt.coloured, one.Led != nil)
		})
	}
}

func TestDocumentPublicTestSuite(t *testing.T) {
	suite.Run(t, new(DocumentPublicTestSuite))
}
