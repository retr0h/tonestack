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
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/catalog"
	"github.com/retr0h/tonestack/pkg/chain"
	"github.com/retr0h/tonestack/pkg/compile"
	"github.com/retr0h/tonestack/pkg/preset"
	"github.com/retr0h/tonestack/pkg/rig"
	riggen "github.com/retr0h/tonestack/pkg/rig/gen"
)

type LiftPublicTestSuite struct {
	suite.Suite

	cat *catalog.Catalog
}

func (s *LiftPublicTestSuite) SetupSuite() {
	var err error

	s.cat, err = catalog.BuiltIn()
	s.Require().NoError(err)
}

// preset builds a document holding one block of the given model.
func (s *LiftPublicTestSuite) preset(name string, models ...catalog.ModelID) *preset.Document {
	blocks := make([]chain.Block, 0, len(models))
	for i, m := range models {
		blocks = append(blocks, chain.Block{Model: m, Pos: i, Enabled: true})
	}

	doc, err := preset.New(s.cat.DeviceID, chain.Chain{Name: name, Blocks: blocks})
	s.Require().NoError(err)

	return doc
}

// catalogOf builds a catalog holding nothing but the given blocks.
func (s *LiftPublicTestSuite) catalogOf(
	blocks map[catalog.ModelID]catalog.Block,
) *catalog.Catalog {
	if blocks == nil {
		return s.cat
	}

	return &catalog.Catalog{
		Device: "Test", DeviceID: s.cat.DeviceID, Blocks: blocks,
	}
}

// rigOf returns a valid rig naming one piece of gear.
func rigOf(id, gear string, inst riggen.Instrument, params *map[string]any) riggen.RigSpec {
	return riggen.RigSpec{
		Schema:     riggen.RigSpecSchemaRigSpec,
		ID:         id,
		Subject:    riggen.Subject{Kind: riggen.KindSound, Name: id},
		Instrument: inst,
		Chain: []riggen.ChainEntry{
			{Role: riggen.RoleAmp, Gear: gear, Params: params},
		},
	}
}

// TestLift reads a preset as a rig.
func (s *LiftPublicTestSuite) TestLift() {
	tests := []struct {
		name string
		// what the preset is called, which is where a rig's identifier comes
		// from.
		title  string
		model  catalog.ModelID
		blocks map[catalog.ModelID]catalog.Block
		// a document written by hand, for what preset.New cannot build.
		raw string
		// a preset holding no blocks at all.
		bare bool

		wantGear       string
		wantRole       riggen.Role
		wantInstrument riggen.Instrument
		wantID         string
		err            error
		errText        string
	}{
		{
			name:           "an amp emulating real gear, named the way a person would",
			model:          "HD2_AmpSVBeastNrm",
			wantGear:       "Ampeg SVT® (normal channel)",
			wantInstrument: riggen.InstrumentBass,
		},
		{
			name:           "a Line 6 original, by its own name, since it emulates nothing",
			model:          "HD2_AmpLine6Litigator",
			wantGear:       "Line 6 Litigator",
			wantInstrument: riggen.InstrumentGuitar,
		},
		{
			// A rig has to say what every block is, and "something this
			// device carries and we do not recognise" is a truthful answer.
			name:           "a model the catalog has never heard of, by identifier",
			model:          "HD2_NotInThisCatalog",
			wantGear:       "HD2_NotInThisCatalog",
			wantRole:       riggen.RoleOther,
			wantInstrument: riggen.InstrumentGuitar,
		},
		{
			name:           "a preset with no amp in it",
			model:          "HD2_DistMinotaur",
			wantInstrument: riggen.InstrumentGuitar,
		},
		{
			// A handful of catalog entries carry an empty name and no gear,
			// so neither handle is available and the identifier is all there
			// is.
			name:  "a model with no name",
			model: "HD2_Nameless",
			blocks: map[catalog.ModelID]catalog.Block{
				"HD2_Nameless": {ID: "HD2_Nameless", Category: catalog.CategoryDrive},
			},
			wantGear: "HD2_Nameless",
		},
		{
			name:  "a category this project does not know",
			model: "HD2_Odd",
			blocks: map[catalog.ModelID]catalog.Block{
				"HD2_Odd": {
					ID: "HD2_Odd", Name: "Odd",
					Category: catalog.Category("nonsense"),
				},
			},
			wantRole: riggen.RoleOther,
		},
		{
			name:   "a name that is already an identifier",
			title:  "Mike Dirnt",
			model:  "HD2_AmpSVBeastNrm",
			wantID: "mike-dirnt",
		},
		{
			name:   "a name carrying punctuation",
			title:  "CT-Blackend",
			model:  "HD2_AmpSVBeastNrm",
			wantID: "ct-blackend",
		},
		{
			name:   "a name somebody spaced out",
			title:  "  Lots   of   Space  ",
			model:  "HD2_AmpSVBeastNrm",
			wantID: "lots-of-space",
		},
		{
			name:   "a name of nothing but punctuation",
			title:  "!!!",
			model:  "HD2_AmpSVBeastNrm",
			wantID: "untitled",
		},
		{
			name:   "no name at all",
			model:  "HD2_AmpSVBeastNrm",
			wantID: "untitled",
		},
		{name: "a preset holding no blocks", bare: true, err: rig.ErrInvalid},
		{
			name: "a document it cannot read",
			raw: `{"schema":"L6Preset","version":6,"data":{"device":2162694,` +
				`"meta":{"name":"Bad"},"tone":{"dspX":{"block0":{"@model":"x"}}}}}`,
			errText: "reading the chain",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var doc *preset.Document

			switch {
			case tt.raw != "":
				var err error

				doc, err = preset.Read(bytes.NewReader([]byte(tt.raw)))
				s.Require().NoError(err)
			case tt.bare:
				var err error

				doc, err = preset.New(s.cat.DeviceID, chain.Chain{Name: "Empty"})
				s.Require().NoError(err)
			default:
				doc = s.preset(tt.title, tt.model)
			}

			got, err := compile.Lift(doc, s.catalogOf(tt.blocks))

			if tt.err != nil || tt.errText != "" {
				s.Require().Error(err)

				if tt.err != nil {
					s.Require().ErrorIs(err, tt.err)
				}

				if tt.errText != "" {
					s.Require().Contains(err.Error(), tt.errText)
				}

				return
			}

			s.Require().NoError(err)
			s.Require().NoError(rig.Validate(got), "a lifted rig must validate")

			if tt.wantGear != "" {
				s.Require().Equal(tt.wantGear, got.Chain[0].Gear)
			}

			if tt.wantRole != "" {
				s.Require().Equal(tt.wantRole, got.Chain[0].Role)
			}

			if tt.wantInstrument != "" {
				s.Require().Equal(tt.wantInstrument, got.Instrument)
			}

			if tt.wantID != "" {
				s.Require().Equal(tt.wantID, got.ID)
			}
		})
	}
}

// withSwitch puts one footswitch on a rig, lit the given colour.
func withSwitch(spec riggen.RigSpec, led string) riggen.RigSpec {
	spec.Footswitches = &[]riggen.Footswitch{{Led: &led}}

	return spec
}

// TestLower writes a rig into a preset.
// substituted says what to put in place of the gear a rig names.
func substituted(spec riggen.RigSpec, instead string) riggen.RigSpec {
	spec.Chain[0].Substitute = &riggen.Substitute{Gear: instead}

	return spec
}

func (s *LiftPublicTestSuite) TestLower() {
	tests := []struct {
		name   string
		spec   riggen.RigSpec
		blocks map[catalog.ModelID]catalog.Block
		// the model the destination preset already holds.
		into catalog.ModelID
		// a rig lifted off a preset of this model, rather than one written
		// out here.
		from catalog.ModelID

		wantModel  catalog.ModelID
		wantParams []string
		absent     []string
		exact      int
		types      map[string]catalog.ParamType
		ints       map[string]int64
		err        error
		errText    string
	}{
		{name: "a rig that is not one", err: rig.ErrInvalid},
		{
			name:    "gear nothing on this device models",
			spec:    rigOf("nope", "Nonesuch 900", riggen.InstrumentGuitar, nil),
			errText: "emulates \"Nonesuch 900\"",
		},
		{
			// The rig names what was really played and says what this device
			// should put there, so building it lands on the stand-in.
			name: "gear nothing models, with a stand-in the rig names",
			spec: substituted(
				rigOf("stood-in", "Nonesuch 900", riggen.InstrumentBass, nil),
				"Ampeg SVT (normal"),
			wantModel: "HD2_AmpSVBeastNrm",
			exact:     -1,
		},
		{
			name: "a stand-in nothing models either",
			spec: substituted(
				rigOf("nope", "Nonesuch 900", riggen.InstrumentBass, nil),
				"Also Nonesuch"),
			errText: `"Also Nonesuch" stands in for "Nonesuch 900"`,
		},
		{
			// A rig describing gear rather than a block gets Line 6's own
			// defaults, which are never invalid.
			name:  "a rig stating no parameters",
			spec:  rigOf("plain", "Ampeg SVT (normal", riggen.InstrumentBass, nil),
			exact: -1,
		},
		{
			name: "a rig stating parameters, which are the whole truth",
			spec: rigOf("exact", "Ampeg SVT (normal", riggen.InstrumentBass,
				&map[string]any{
					"Drive": 0.8, "MidFreq": 2.0,
					"Bright": true, "Voicing": "Modern",
				}),
			exact: 4,
			// A switch stays a switch: a device given 1.5 for a
			// three-position control refuses the preset rather than rounding.
			types: map[string]catalog.ParamType{
				"MidFreq": catalog.ParamInt,
				"Bright":  catalog.ParamBool,
				"Voicing": catalog.ParamEnum,
			},
		},
		{
			// Written as stated. Nudging by a half and truncating cuts
			// toward zero, so this arrived as -11: an octave down turned
			// into a major seventh, in a preset nobody would think to check.
			name: "a parameter somebody set below nothing",
			spec: rigOf("octave", "Ampeg SVT (normal", riggen.InstrumentBass,
				&map[string]any{"MidFreq": -12.0}),
			types: map[string]catalog.ParamType{"MidFreq": catalog.ParamInt},
			ints:  map[string]int64{"MidFreq": -12},
		},
		{
			// "Ampeg SVT" matches both channels. The recorded identifier is
			// what makes a lifted rig rebuild into the preset it came from.
			name:      "the exact model, over the name it shares",
			from:      "HD2_AmpSVBeastBrt",
			wantModel: "HD2_AmpSVBeastBrt",
		},
		{
			// A parameter Line 6 state no default for has no kind, and
			// writing a value with no kind produces a preset the device
			// rejects.
			name: "a parameter with no stated default",
			spec: rigOf("half", "Half A Thing", riggen.InstrumentGuitar, nil),
			blocks: map[catalog.ModelID]catalog.Block{
				"HD2_Half": {
					ID: "HD2_Half", Name: "Half", BasedOn: "Half A Thing",
					Category: catalog.CategoryAmp,
					Params: map[string]catalog.Param{
						"Drive":   {Type: catalog.ParamFloat, Default: catalog.Float(0.5)},
						"Missing": {Type: catalog.ParamFloat},
					},
				},
			},
			into:       "HD2_Half",
			wantParams: []string{"Drive"},
			absent:     []string{"Missing"},
		},
		{
			// Lowering asks the same question about what a rig claims beside
			// its chain, so a colour this device cannot light fails here
			// rather than reaching a preset.
			name: "a colour the device does not have",
			spec: withSwitch(
				rigOf("lit", "Ampeg SVT (normal", riggen.InstrumentBass, nil),
				"chartruse"),
			errText: "footswitches[0].led",
		},
		{
			name: "a parameter of no known kind",
			spec: rigOf("odd", "Ampeg SVT (normal", riggen.InstrumentBass,
				&map[string]any{"Drive": 0.8, "Nonsense": []any{1, 2}}),
			wantParams: []string{"Drive"},
			absent:     []string{"Nonsense"},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			cat := s.catalogOf(tt.blocks)

			into := tt.into
			if into == "" {
				into = "HD2_AmpSVBeastNrm"
			}

			doc := s.preset("Test", into)

			spec := tt.spec

			if tt.from != "" {
				var err error

				spec, err = compile.Lift(s.preset("Test", tt.from), cat)
				s.Require().NoError(err)
			}

			err := compile.Lower(doc, spec, cat)

			if tt.err != nil || tt.errText != "" {
				s.Require().Error(err)

				if tt.err != nil {
					s.Require().ErrorIs(err, tt.err)
				}

				if tt.errText != "" {
					s.Require().Contains(err.Error(), tt.errText)
				}

				return
			}

			s.Require().NoError(err)

			c, err := doc.Spec()
			s.Require().NoError(err)

			if tt.wantModel != "" {
				s.Require().Equal(tt.wantModel, c.Blocks[0].Model)
			}

			switch {
			case tt.exact < 0:
				s.Require().NotEmpty(c.Blocks[0].Params, "every knob is set")
			case tt.exact > 0:
				s.Require().Len(c.Blocks[0].Params, tt.exact)
			}

			for _, want := range tt.wantParams {
				s.Require().Contains(c.Blocks[0].Params, want)
			}

			for _, unwanted := range tt.absent {
				s.Require().NotContains(c.Blocks[0].Params, unwanted)
			}

			for key, want := range tt.types {
				s.Require().Equal(want, c.Blocks[0].Params[key].Type())
			}

			for key, want := range tt.ints {
				got, ok := c.Blocks[0].Params[key].Int()
				s.Require().True(ok)
				s.Require().Equal(want, got, "%s", key)
			}
		})
	}
}

// TestLowerPicksTheSameModelEveryTime is a property of the resolver rather
// than a case of the call.
//
// Lowering used to range a map and take the first name that matched, so one
// rig became a different preset each run: six compiles of this one named an
// amplifier, a preamp, the bright channel and twice a cabinet. A rig lifted
// off a device was unaffected, because it carries the model identifier, which
// is why nothing caught it.
func (s *LiftPublicTestSuite) TestLowerPicksTheSameModelEveryTime() {
	spec := rigOf("stable", "Ampeg SVT", riggen.InstrumentBass, nil)

	var first catalog.ModelID

	for range 20 {
		doc, err := preset.Blank()
		s.Require().NoError(err)
		s.Require().NoError(compile.Lower(doc, spec, s.cat))

		c, err := doc.Spec()
		s.Require().NoError(err)
		s.Require().NotEmpty(c.Blocks)

		if first == "" {
			first = c.Blocks[0].Model
		}

		s.Require().Equal(first, c.Blocks[0].Model, "the same rig, a different model")
	}

	// And an amplifier, because the role is half the question.
	b, ok := s.cat.Block(first)
	s.Require().True(ok)
	s.Require().Equal(catalog.CategoryAmp, b.Category)
}

func TestLiftPublicTestSuite(t *testing.T) {
	suite.Run(t, new(LiftPublicTestSuite))
}
