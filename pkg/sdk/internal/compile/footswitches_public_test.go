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

package compile_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/internal/compile"
	"github.com/retr0h/tonestack/pkg/sdk/preset"
	"github.com/retr0h/tonestack/pkg/sdk/rig"
)

// FootswitchesPublicTestSuite covers what a preset says about the pedal.
//
// A preset keys footswitches by the block each acts on, so a rig has to carry
// both the block and the switch or it cannot be written back. Everything here
// is about that going both ways, and about a file somebody edited by hand not
// producing a broken preset.
type FootswitchesPublicTestSuite struct {
	suite.Suite

	cat *catalog.Catalog
}

func (s *FootswitchesPublicTestSuite) SetupSuite() {
	var err error

	s.cat, err = catalog.BuiltIn()
	s.Require().NoError(err)
}

// presetWith returns a document whose footswitch section is the given JSON.
func (s *FootswitchesPublicTestSuite) presetWith(
	body string,
) *preset.Document {
	doc, err := preset.Blank()
	s.Require().NoError(err)

	var entry preset.Tone
	s.Require().NoError(json.Unmarshal([]byte(body), &entry))

	doc.Data.Tone["footswitch"] = entry

	// A rig holds at least one thing, so the preset it is read from has to.
	doc.Data.Tone["dsp0"]["block0"] = json.RawMessage(
		`{"@model": "HD2_AmpSVBeastNrm", "@position": 0, "@enabled": true}`)

	return doc
}

// TestLiftFootswitches reads what a preset says about the pedal.
func (s *FootswitchesPublicTestSuite) TestLiftFootswitches() {
	tests := []struct {
		name string
		body string
		// what the one switch must say.
		wantSwitch int
		wantBlock  int
		wantPath   int
		wantLabel  string
		wantColour int
		primary    bool
		// fields this does not model, which it must not drop either.
		rest bool
		// what writing the rig back must put in the preset.
		writes []string
		// a switch naming no block at all.
		none bool
	}{
		{
			name: "a switch on the first processor",
			body: `{"dsp0": {"block1": {
				"@fs_index": 2, "@fs_label": "Fuzz", "@fs_ledcolor": 16711683,
				"@fs_enabled": true, "@fs_momentary": false, "@fs_primary": true,
				"@fs_unknown": 7
			}}}`,
			wantSwitch: 2,
			wantBlock:  1,
			wantLabel:  "Fuzz",
			wantColour: 16711683,
			primary:    true,
			rest:       true,
			writes: []string{
				`"@fs_label": "Fuzz"`,
				`"@fs_index": 2`,
				`"@fs_unknown": 7`,
			},
		},
		{
			name:       "one on the second",
			body:       `{"dsp1": {"block3": {"@fs_index": 1, "@fs_label": "X"}}}`,
			wantSwitch: 1,
			wantBlock:  3,
			wantPath:   1,
			wantLabel:  "X",
		},
		{
			name: "a processor no preset has",
			body: `{"variax": {"block1": {"@fs_index": 1}}}`,
			none: true,
		},
		{
			name: "a block key nothing can number",
			body: `{"dsp0": {"nonsense": {"@fs_index": 1}}}`,
			none: true,
		},
		{
			name: "a processor that is not an object",
			body: `{"dsp0": "not an object"}`,
			none: true,
		},
		{
			name: "a switch that is not an object",
			body: `{"dsp0": {"block1": "not an object"}}`,
			none: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			spec, err := compile.Lift(s.presetWith(tt.body), s.cat)
			s.Require().NoError(err)

			if tt.none {
				s.Require().Nil(spec.Footswitches)

				return
			}

			s.Require().NotNil(spec.Footswitches)
			s.Require().Len(*spec.Footswitches, 1)

			fs := (*spec.Footswitches)[0]
			s.Require().Equal(tt.wantSwitch, *fs.Switch)
			s.Require().Equal(tt.wantBlock, *fs.Block,
				"a switch acts on a block, and says which")
			s.Require().Equal(tt.wantLabel, *fs.Label)

			if tt.wantPath > 0 {
				s.Require().Equal(tt.wantPath, *fs.Path)
			}

			if tt.wantColour > 0 {
				s.Require().Equal(tt.wantColour, *fs.Colour)
			}

			if tt.primary {
				s.Require().True(*fs.Primary)
			}

			if tt.rest {
				s.Require().NotNil(fs.Rest,
					"a field this does not model is not one it drops")
			}

			if tt.writes == nil {
				return
			}

			back, err := preset.Blank()
			s.Require().NoError(err)
			s.Require().NoError(compile.Lower(back, spec, s.cat))

			var out bytes.Buffer
			s.Require().NoError(preset.Write(&out, back))

			for _, want := range tt.writes {
				s.Require().Contains(out.String(), want)
			}
		})
	}
}

func (s *FootswitchesPublicTestSuite) TestASwitchWithNoBlockIsNotWritten() {
	// A rig somebody edited can say anything. A switch that names no block
	// has nowhere to go, and dropping it beats writing a preset that will not
	// load.
	label := "orphan"
	spec := rig.Spec{
		Schema:     rig.SchemaName,
		ID:         "test",
		Subject:    rig.Subject{Kind: rig.KindSound, Name: "Test"},
		Instrument: rig.InstrumentBass,
		Chain:      []rig.ChainEntry{{Role: rig.RoleAmp, Gear: "Ampeg SVT"}},
		Footswitches: &[]rig.Footswitch{
			{Label: &label},
		},
	}

	doc, err := preset.Blank()
	s.Require().NoError(err)
	s.Require().NoError(compile.Lower(doc, spec, s.cat))

	var out bytes.Buffer
	s.Require().NoError(preset.Write(&out, doc))
	s.Require().NotContains(out.String(), "orphan")
}

// TestAChosenColourGoesBothWays covers the colour somebody picked surviving
// the trip out of a preset and back into one.
//
// The device files a colour under its place in the catalog's list, and stores
// that number rather than the light it produces. `colour` is the light, which
// the device works out from the block.
func (s *FootswitchesPublicTestSuite) TestAChosenColourGoesBothWays() {
	doc := s.presetWith(`{"dsp0": {"block1": {"@fs_index": 3, "@fs_customcolor": 3,
		"@fs_ledcolor": 525824, "@fs_label": "Dhyana Drive"}}}`)

	lifted, err := compile.Lift(doc, s.cat)
	s.Require().NoError(err)
	s.Require().NotNil(lifted.Footswitches)

	fs := (*lifted.Footswitches)[0]
	s.Require().NotNil(fs.Led)
	s.Require().Equal("dark orange", *fs.Led,
		"the third colour the catalog lists, which is what the number means")

	// Back the way it came.
	back, err := preset.Blank()
	s.Require().NoError(err)

	compile.Footswitches(back, lifted, s.cat)

	var out bytes.Buffer
	s.Require().NoError(preset.Write(&out, back))
	s.Require().Contains(out.String(), `"@fs_customcolor": 3`)
}

// TestAColourNobodyChoseIsNotWritten covers a rig that names no colour, and
// one that names something this device does not have.
func (s *FootswitchesPublicTestSuite) TestAColourNobodyChoseIsNotWritten() {
	block, switched := 1, 3
	nonsense := "burnt sienna"

	tests := []struct {
		name string
		led  *string
	}{
		{name: "a rig that names no colour"},
		{name: "a colour this device does not have", led: &nonsense},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			doc, err := preset.Blank()
			s.Require().NoError(err)

			compile.Footswitches(doc, rig.Spec{Footswitches: &[]rig.Footswitch{
				{Switch: &switched, Block: &block, Led: tt.led},
			}}, s.cat)

			var out bytes.Buffer
			s.Require().NoError(preset.Write(&out, doc))
			s.Require().NotContains(out.String(), "@fs_customcolor")
		})
	}
}

// TestAColourByNameReachesABuiltPreset covers the point of all this: a rig
// somebody typed asking for a red switch and getting one.
func (s *FootswitchesPublicTestSuite) TestAColourByNameReachesABuiltPreset() {
	block, switched := 0, 1
	red := "red"

	spec := rig.Spec{
		Schema:     rig.SchemaName,
		ID:         "test",
		Subject:    rig.Subject{Kind: rig.KindSound, Name: "Test"},
		Instrument: rig.InstrumentBass,
		Chain:      []rig.ChainEntry{{Role: rig.RoleAmp, Gear: "Ampeg SVT"}},
		Footswitches: &[]rig.Footswitch{
			{Switch: &switched, Block: &block, Led: &red},
		},
	}

	doc, err := preset.Blank()
	s.Require().NoError(err)
	s.Require().NoError(compile.Lower(doc, spec, s.cat))

	var out bytes.Buffer
	s.Require().NoError(preset.Write(&out, doc))
	s.Require().Contains(out.String(), `"@fs_customcolor": 2`)
}

func TestFootswitchesPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(FootswitchesPublicTestSuite))
}
