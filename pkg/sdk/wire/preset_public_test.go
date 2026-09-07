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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

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
	raw, err := os.ReadFile(filepath.Join("testdata", "preset.bin"))
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

func TestPresetPublicTestSuite(t *testing.T) {
	suite.Run(t, new(PresetPublicTestSuite))
}
