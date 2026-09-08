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

package wire_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/vmihailenco/msgpack/v5"

	"github.com/retr0h/tonestack/pkg/sdk/wire"
)

// PresetPublicTestSuite reads what an HX Stomp actually answered.
//
// testdata/preset.bin is one slot as the hardware handed it back, captured
// over USB. Nothing else in this repository describes the wire format, so a
// real answer is the only thing that can say whether the reading is right.
type PresetPublicTestSuite struct {
	suite.Suite
}

func (s *PresetPublicTestSuite) capture() []byte {
	return s.captureNamed("preset.bin")
}

// captureNamed returns one slot as an HX Stomp actually sent it.
func (s *PresetPublicTestSuite) captureNamed(name string) []byte {
	raw, err := os.ReadFile(filepath.Join("testdata", name))
	s.Require().NoError(err)

	return raw
}

func (s *PresetPublicTestSuite) TestReadsTheChainADeviceSent() {
	// The slot is a factory preset called "BAS:SVT Nrm", so the models it
	// names are known before decoding: a volume pedal, an LA-2A, two pitch
	// blocks, the SVT's normal channel and the 8x10 it is voiced with.
	got, err := wire.DecodePreset(s.capture())

	s.Require().NoError(err)
	s.Require().Len(got.Blocks, 6)

	s.Require().Equal(261, got.Blocks[0].Model)

	// Where the device put them, not where they fall in the chain: a preset
	// leaves gaps in its layout, and footswitches address blocks by this.
	s.Require().Equal(2, got.Blocks[0].Index)
	s.Require().Equal(5, got.Blocks[4].Model, "the SVT normal channel")
	s.Require().Equal(74, got.Blocks[5].Model, "the 8x10 it is paired with")
}

func (s *PresetPublicTestSuite) TestParametersArriveByPosition() {
	got, err := wire.DecodePreset(s.capture())
	s.Require().NoError(err)

	// An amp has twelve, and their names come from the catalog's model table
	// rather than from anything the device sent.
	s.Require().Len(got.Blocks[4].Values, 12)
}

func (s *PresetPublicTestSuite) TestCarriesWhetherABlockIsOn() {
	got, err := wire.DecodePreset(s.capture())
	s.Require().NoError(err)

	s.Require().True(got.Blocks[0].Enabled)
	s.Require().False(got.Blocks[2].Enabled, "a bypassed block is still a block")
}

func (s *PresetPublicTestSuite) TestRefusesWhatIsNotAPreset() {
	for _, tc := range []struct {
		name string
		body []byte
	}{
		{"nothing at all", nil},
		{"something else entirely", []byte{0xc0}},
		{"the header and no more", []byte("\xa9l6-helix\x00")},
		{"a header and offsets but no document", []byte("\xa9l6-helix\x00\xa1x")},
	} {
		s.Run(tc.name, func() {
			_, err := wire.DecodePreset(tc.body)

			s.Require().ErrorIs(err, wire.ErrNotAPreset)
		})
	}
}

func (s *PresetPublicTestSuite) TestADocumentThatIsNotAMap() {
	_, err := wire.DecodePreset([]byte("\xa9l6-helix\x00\xa1x\xc3"))

	s.Require().ErrorIs(err, wire.ErrNotAPreset)
}

func (s *PresetPublicTestSuite) TestReadsTheCabinetAnAmplifierCarries() {
	// switches.bin is a slot holding two amplifiers, each with a cabinet.
	// A device stores the pair as one block.
	raw, err := os.ReadFile(filepath.Join("testdata", "switches.bin"))
	s.Require().NoError(err)

	got, err := wire.DecodePreset(raw)
	s.Require().NoError(err)

	var paired int

	for _, b := range got.Blocks {
		if len(b.Cab) == 0 {
			continue
		}

		paired++

		// Five settings the cabinet model has names for, and a microphone
		// after them.
		s.Require().Equal(5, b.CabNamed)
		s.Require().Len(b.Cab, 6)
	}

	s.Require().Equal(2, paired, "two amplifiers, two cabinets")
}

func TestPresetPublicTestSuite(t *testing.T) {
	suite.Run(t, new(PresetPublicTestSuite))
}

// TestDecodeLoaded covers what a device answers when asked what it is
// playing, which is the only honest signal that a select has finished.
func (s *PresetPublicTestSuite) TestDecodeLoaded() {
	tests := []struct {
		name   string
		result any
		want   wire.Loaded
		err    bool
	}{
		{
			name: "a device naming what it plays",
			result: map[any]any{
				int8(107): uint8(0),
				int8(108): uint8(99),
				int8(109): "Chunky Monkey",
			},
			want: wire.Loaded{Setlist: 0, Slot: 99, Name: "Chunky Monkey"},
		},
		{
			name: "one that names no preset",
			result: map[any]any{
				int8(107): uint8(0),
				int8(108): uint8(7),
			},
			want: wire.Loaded{Setlist: 0, Slot: 7},
		},
		{
			name:   "an answer that is not a map",
			result: "nonsense",
			err:    true,
		},
		{
			name:   "one naming no setlist",
			result: map[any]any{int8(108): uint8(7)},
			err:    true,
		},
		{
			name:   "one naming no slot",
			result: map[any]any{int8(107): uint8(0)},
			err:    true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := wire.DecodeLoaded(tt.result)

			if tt.err {
				s.Require().Error(err)

				return
			}

			s.Require().NoError(err)
			s.Require().Equal(tt.want, got)
		})
	}
}

// TestDecodesControllerAssignments covers what an expression pedal moves.
//
// The captured preset carries exactly one, and it is what established the
// layout: slot 27B read off an HX Stomp beside the same slot exported from
// HX Edit, which names it as block0's Pedal with min 0, max 1, controller 2.
func (s *PresetPublicTestSuite) TestDecodesControllerAssignments() {
	got, err := wire.DecodePreset(s.capture())

	s.Require().NoError(err)
	s.Require().Equal([]wire.DeviceController{{
		Controller: 2,
		Block:      2,
		Param:      0,
		Min:        0,
		Max:        1,
	}}, got.Controllers)
}

// TestAPresetWithNoControllers covers the ordinary case: nine of the ten
// slots hold nothing on any real preset, and most presets assign none at all.
func (s *PresetPublicTestSuite) TestAPresetWithNoControllers() {
	for _, name := range []string{"switches.bin", "empty.bin"} {
		got, err := wire.DecodePreset(s.captureNamed(name))

		s.Require().NoError(err)
		s.Require().Empty(got.Controllers, name)
	}
}

// TestControllerSectionsNoDeviceWouldSend covers every shape the decoder
// refuses, since a preset carrying one assignment exercises none of them.
func (s *PresetPublicTestSuite) TestControllerSectionsNoDeviceWouldSend() {
	tests := []struct {
		name    string
		section any
		want    []wire.DeviceController
	}{
		{
			name:    "a section that is not an array",
			section: 42,
		},
		{
			name:    "a controller holding something that is not a list",
			section: []any{42},
		},
		{
			name:    "an assignment that is not a map",
			section: []any{[]any{"nonsense"}},
		},
		{
			name:    "one naming no parameter",
			section: []any{[]any{map[any]any{int8(1): map[any]any{}}}},
		},
		{
			name: "one with no body",
			section: []any{[]any{map[any]any{
				int8(0): int8(0),
			}}},
		},
		{
			name: "one naming no block",
			section: []any{[]any{map[any]any{
				int8(0): int8(0),
				int8(1): map[any]any{int8(2): 0.0},
			}}},
		},
		{
			name: "travel a device wrote as whole numbers",
			section: []any{nil, nil, []any{map[any]any{
				int8(0): int8(1),
				int8(1): map[any]any{
					int8(0): int8(5),
					int8(2): int8(0),
					int8(3): int8(1),
					int8(6): map[any]any{int8(41): true},
				},
			}}},
			want: []wire.DeviceController{{
				Controller: 2, Block: 5, Param: 1, Min: 0, Max: 1,
				NoSnapshot: true,
			}},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := wire.DecodePreset(s.withControllers(tt.section))

			s.Require().NoError(err)
			s.Require().Equal(tt.want, got.Controllers)
		})
	}
}

// withControllers rebuilds the captured preset with a different controller
// section, so shapes a device never sends can still be read.
func (s *PresetPublicTestSuite) withControllers(section any) []byte {
	doc, err := wire.DecodeDocument(s.capture())
	s.Require().NoError(err)

	var buf bytes.Buffer

	enc := msgpack.NewEncoder(&buf)
	enc.SetCustomStructTag("msgpack")
	s.Require().NoError(enc.Encode(section))

	doc.SetSection(4, buf.Bytes())

	return doc.Encode()
}
