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

package lift_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/internal/lift"
	"github.com/retr0h/tonestack/pkg/catalog"
	"github.com/retr0h/tonestack/pkg/preset"
	riggen "github.com/retr0h/tonestack/pkg/rig/gen"
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
func (s *FootswitchesPublicTestSuite) presetWith(body string) *preset.Document {
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
			spec, err := lift.Lift(s.presetWith(tt.body), s.cat)
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
			s.Require().NoError(lift.Lower(back, spec, s.cat))

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
	spec := riggen.RigSpec{
		Schema:     riggen.RigSpecSchemaRigSpec,
		ID:         "test",
		Subject:    riggen.Subject{Kind: riggen.KindSound, Name: "Test"},
		Instrument: riggen.InstrumentBass,
		Chain:      []riggen.ChainEntry{{Role: riggen.RoleAmp, Gear: "Ampeg SVT"}},
		Footswitches: &[]riggen.Footswitch{
			{Label: &label},
		},
	}

	doc, err := preset.Blank()
	s.Require().NoError(err)
	s.Require().NoError(lift.Lower(doc, spec, s.cat))

	var out bytes.Buffer
	s.Require().NoError(preset.Write(&out, doc))
	s.Require().NotContains(out.String(), "orphan")
}

func TestFootswitchesPublicTestSuite(t *testing.T) {
	suite.Run(t, new(FootswitchesPublicTestSuite))
}
