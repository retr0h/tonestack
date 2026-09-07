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
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/internal/lift"
	"github.com/retr0h/tonestack/pkg/catalog"
	"github.com/retr0h/tonestack/pkg/chain"
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

func (s *LiftPublicTestSuite) TestNamesGearTheWayAPersonWould() {
	tests := []struct {
		name  string
		model catalog.ModelID
		want  string
	}{
		{
			"the real gear a model emulates",
			"HD2_AmpSVBeastNrm", "Ampeg SVT® (normal channel)",
		},
		{
			"a Line 6 original by its own name, since it emulates nothing",
			"HD2_AmpLine6Litigator", "Line 6 Litigator",
		},
		{
			"a model the catalog has never heard of, by identifier",
			"HD2_NotInThisCatalog", "HD2_NotInThisCatalog",
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			got, err := lift.Lift(s.preset("Test", tc.model), s.cat)

			s.Require().NoError(err)
			s.Require().Equal(tc.want, got.Chain[0].Gear)
		})
	}
}

func (s *LiftPublicTestSuite) TestDescribesAnUnknownModelAsOther() {
	// A rig has to say what every block is, and "something this device
	// carries and we do not recognise" is a truthful answer.
	got, err := lift.Lift(s.preset("Test", "HD2_NotInThisCatalog"), s.cat)

	s.Require().NoError(err)
	s.Require().Equal(riggen.RoleOther, got.Chain[0].Role)
}

func (s *LiftPublicTestSuite) TestReadsTheInstrumentFromTheAmp() {
	tests := []struct {
		name  string
		model catalog.ModelID
		want  riggen.Instrument
	}{
		{"a bass amp", "HD2_AmpSVBeastNrm", riggen.InstrumentBass},
		{"a guitar amp", "HD2_AmpLine6Litigator", riggen.InstrumentGuitar},
		{"no amp at all", "HD2_DistMinotaur", riggen.InstrumentGuitar},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			got, err := lift.Lift(s.preset("Test", tc.model), s.cat)

			s.Require().NoError(err)
			s.Require().Equal(tc.want, got.Instrument)
		})
	}
}

func (s *LiftPublicTestSuite) TestMakesAnIdentifierOutOfTheName() {
	tests := []struct {
		name string
		want string
	}{
		{"Mike Dirnt", "mike-dirnt"},
		{"CT-Blackend", "ct-blackend"},
		{"  Lots   of   Space  ", "lots-of-space"},
		{"!!!", "untitled"},
		{"", "untitled"},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			got, err := lift.Lift(s.preset(tc.name, "HD2_AmpSVBeastNrm"), s.cat)

			s.Require().NoError(err)
			s.Require().Equal(tc.want, got.ID)
			s.Require().NoError(rig.Validate(got), "and it must still validate")
		})
	}
}

func (s *LiftPublicTestSuite) TestAPresetWithNoBlocksIsNotARig() {
	doc, err := preset.New(s.cat.DeviceID, chain.Chain{Name: "Empty"})
	s.Require().NoError(err)

	_, err = lift.Lift(doc, s.cat)

	s.Require().ErrorIs(err, rig.ErrInvalid)
}

func (s *LiftPublicTestSuite) TestLowerRefusesARigThatIsNotOne() {
	doc := s.preset("Test", "HD2_AmpSVBeastNrm")

	err := lift.Lower(doc, riggen.RigSpec{}, s.cat)

	s.Require().ErrorIs(err, rig.ErrInvalid)
}

func (s *LiftPublicTestSuite) TestLowerRefusesGearNothingModels() {
	err := lift.Lower(s.preset("Test", "HD2_AmpSVBeastNrm"), riggen.RigSpec{
		Schema:     riggen.RigSpecSchemaRigSpec,
		ID:         "nope",
		Subject:    riggen.Subject{Kind: riggen.KindSound, Name: "Nope"},
		Instrument: riggen.InstrumentGuitar,
		Chain:      []riggen.ChainEntry{{Role: riggen.RoleAmp, Gear: "Nonesuch 900"}},
	}, s.cat)

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "nothing on this device is")
}

func (s *LiftPublicTestSuite) TestARigWithNoParametersTakesTheDefaults() {
	// A rig describing gear rather than a block gets Line 6's own defaults,
	// which are never invalid.
	doc := s.preset("Test", "HD2_AmpSVBeastNrm")

	s.Require().NoError(lift.Lower(doc, riggen.RigSpec{
		Schema:     riggen.RigSpecSchemaRigSpec,
		ID:         "plain",
		Subject:    riggen.Subject{Kind: riggen.KindSound, Name: "Plain"},
		Instrument: riggen.InstrumentBass,
		Chain: []riggen.ChainEntry{
			{Role: riggen.RoleAmp, Gear: "Ampeg SVT (normal"},
		},
	}, s.cat))

	c, err := doc.Spec()
	s.Require().NoError(err)
	s.Require().NotEmpty(c.Blocks[0].Params, "every knob is set")
}

func (s *LiftPublicTestSuite) TestARigStatingParametersMeansExactlyThose() {
	doc := s.preset("Test", "HD2_AmpSVBeastNrm")
	params := map[string]any{"Drive": 0.8, "MidFreq": 2.0, "Bright": true, "Voicing": "Modern"}

	s.Require().NoError(lift.Lower(doc, riggen.RigSpec{
		Schema:     riggen.RigSpecSchemaRigSpec,
		ID:         "exact",
		Subject:    riggen.Subject{Kind: riggen.KindSound, Name: "Exact"},
		Instrument: riggen.InstrumentBass,
		Chain: []riggen.ChainEntry{
			{Role: riggen.RoleAmp, Gear: "Ampeg SVT (normal", Params: &params},
		},
	}, s.cat))

	c, err := doc.Spec()
	s.Require().NoError(err)
	s.Require().Len(c.Blocks[0].Params, len(params),
		"stated parameters are the whole truth, with no defaults added")

	// A switch stays a switch: a device given 1.5 for a three-position
	// control refuses the preset rather than rounding.
	s.Require().Equal(catalog.ParamInt, c.Blocks[0].Params["MidFreq"].Type())
	s.Require().Equal(catalog.ParamBool, c.Blocks[0].Params["Bright"].Type())
	s.Require().Equal(catalog.ParamEnum, c.Blocks[0].Params["Voicing"].Type())
}

func (s *LiftPublicTestSuite) TestARoundTripKeepsTheExactModelOverTheName() {
	// "Ampeg SVT" matches both channels. The recorded identifier is what
	// makes a lifted rig rebuild into the preset it came from.
	doc := s.preset("Test", "HD2_AmpSVBeastBrt")

	spec, err := lift.Lift(doc, s.cat)
	s.Require().NoError(err)

	back := s.preset("Test", "HD2_AmpSVBeastNrm")
	s.Require().NoError(lift.Lower(back, spec, s.cat))

	c, err := back.Spec()
	s.Require().NoError(err)
	s.Require().Equal(catalog.ModelID("HD2_AmpSVBeastBrt"), c.Blocks[0].Model)
}

func (s *LiftPublicTestSuite) TestNamesAModelWithNoNameByIdentifier() {
	// A handful of catalog entries carry an empty name and no gear, so
	// neither handle is available and the identifier is all there is.
	cat := &catalog.Catalog{
		Device: "Test", DeviceID: s.cat.DeviceID,
		Blocks: map[catalog.ModelID]catalog.Block{
			"HD2_Nameless": {ID: "HD2_Nameless", Category: catalog.CategoryDrive},
		},
	}

	got, err := lift.Lift(s.preset("Test", "HD2_Nameless"), cat)

	s.Require().NoError(err)
	s.Require().Equal("HD2_Nameless", got.Chain[0].Gear)
}

func (s *LiftPublicTestSuite) TestDescribesACategoryItDoesNotKnowAsOther() {
	cat := &catalog.Catalog{
		Device: "Test", DeviceID: s.cat.DeviceID,
		Blocks: map[catalog.ModelID]catalog.Block{
			"HD2_Odd": {ID: "HD2_Odd", Name: "Odd", Category: catalog.Category("nonsense")},
		},
	}

	got, err := lift.Lift(s.preset("Test", "HD2_Odd"), cat)

	s.Require().NoError(err)
	s.Require().Equal(riggen.RoleOther, got.Chain[0].Role)
}

func (s *LiftPublicTestSuite) TestSkipsAParameterWithNoStatedDefault() {
	// A parameter Line 6 state no default for has no kind, and writing a
	// value with no kind produces a preset the device rejects.
	cat := &catalog.Catalog{
		Device: "Test", DeviceID: s.cat.DeviceID,
		Blocks: map[catalog.ModelID]catalog.Block{
			"HD2_Half": {
				ID: "HD2_Half", Name: "Half", BasedOn: "Half A Thing",
				Category: catalog.CategoryAmp,
				Params: map[string]catalog.Param{
					"Drive":   {Type: catalog.ParamFloat, Default: catalog.Float(0.5)},
					"Missing": {Type: catalog.ParamFloat},
				},
			},
		},
	}

	doc := s.preset("Test", "HD2_Half")
	s.Require().NoError(lift.Lower(doc, riggen.RigSpec{
		Schema:     riggen.RigSpecSchemaRigSpec,
		ID:         "half",
		Subject:    riggen.Subject{Kind: riggen.KindSound, Name: "Half"},
		Instrument: riggen.InstrumentGuitar,
		Chain:      []riggen.ChainEntry{{Role: riggen.RoleAmp, Gear: "Half A Thing"}},
	}, cat))

	c, err := doc.Spec()
	s.Require().NoError(err)
	s.Require().Contains(c.Blocks[0].Params, "Drive")
	s.Require().NotContains(c.Blocks[0].Params, "Missing")
}

func (s *LiftPublicTestSuite) TestSkipsAParameterOfNoKnownKind() {
	doc := s.preset("Test", "HD2_AmpSVBeastNrm")
	params := map[string]any{"Drive": 0.8, "Nonsense": []any{1, 2}}

	s.Require().NoError(lift.Lower(doc, riggen.RigSpec{
		Schema:     riggen.RigSpecSchemaRigSpec,
		ID:         "odd",
		Subject:    riggen.Subject{Kind: riggen.KindSound, Name: "Odd"},
		Instrument: riggen.InstrumentBass,
		Chain: []riggen.ChainEntry{
			{Role: riggen.RoleAmp, Gear: "Ampeg SVT (normal", Params: &params},
		},
	}, s.cat))

	c, err := doc.Spec()
	s.Require().NoError(err)
	s.Require().Contains(c.Blocks[0].Params, "Drive")
	s.Require().NotContains(c.Blocks[0].Params, "Nonsense")
}

func (s *LiftPublicTestSuite) TestLiftRefusesADocumentItCannotRead() {
	doc, err := preset.Read(bytes.NewReader([]byte(
		`{"schema":"L6Preset","version":6,"data":{"device":2162694,` +
			`"meta":{"name":"Bad"},"tone":{"dspX":{"block0":{"@model":"x"}}}}}`)))
	s.Require().NoError(err)

	_, err = lift.Lift(doc, s.cat)

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "reading the chain")
}

func TestLiftPublicTestSuite(t *testing.T) {
	suite.Run(t, new(LiftPublicTestSuite))
}
