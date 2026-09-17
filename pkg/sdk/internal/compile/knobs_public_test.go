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
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/chain"
	"github.com/retr0h/tonestack/pkg/sdk/internal/compile"
	"github.com/retr0h/tonestack/pkg/sdk/preset"
	"github.com/retr0h/tonestack/pkg/sdk/rig"
)

// KnobsPublicTestSuite covers a rig's musical words reaching real controls.
type KnobsPublicTestSuite struct {
	suite.Suite

	cat *catalog.Catalog
}

func (s *KnobsPublicTestSuite) SetupSuite() { s.cat = loadCatalog(&s.Suite) }

// dial returns a control counted from lo to hi.
func dial(
	kind catalog.ParamType,
	lo, hi float64,
) catalog.Param {
	def := catalog.Float(lo)
	if kind == catalog.ParamInt {
		def = catalog.Int(int64(lo))
	}

	return catalog.Param{Type: kind, Min: lo, Max: hi, Default: def}
}

// knob returns a pointer to one of a rig's settings.
func knob(
	v float64,
) *rig.Knob {
	out := rig.Knob(v)

	return &out
}

// TestSetKnobs covers which control a word lands on, and what it reads.
func (s *KnobsPublicTestSuite) TestSetKnobs() {
	tests := []struct {
		name   string
		params map[string]catalog.Param
		set    *rig.Settings
		want   chain.Params
		errs   []string
	}{
		{
			name:   "a rig that says nothing",
			params: map[string]catalog.Param{"Drive": dial(catalog.ParamFloat, 0, 1)},
			want:   chain.Params{},
		},
		{
			name: "the words an amplifier answers to",
			params: map[string]catalog.Param{
				"Drive": dial(catalog.ParamFloat, 0, 1),
				"Bass":  dial(catalog.ParamFloat, 0, 1),
				"Mid":   dial(catalog.ParamFloat, 0, 1),
			},
			set: &rig.Settings{Drive: knob(0.47), Bass: knob(0.52), Mid: knob(0.71)},
			want: chain.Params{
				"Drive": catalog.Float(0.47),
				"Bass":  catalog.Float(0.52),
				"Mid":   catalog.Float(0.71),
			},
		},
		{
			// One manufacturer calls the same control different things on
			// different models, so a word reaches whichever of them is there.
			name: "controls this model calls something else",
			params: map[string]catalog.Param{
				"Low":   dial(catalog.ParamFloat, 0, 1),
				"High":  dial(catalog.ParamFloat, 0, 1),
				"ChVol": dial(catalog.ParamFloat, 0, 1),
			},
			set: &rig.Settings{Bass: knob(0.25), Treble: knob(0.75), Level: knob(1)},
			want: chain.Params{
				"Low":   catalog.Float(0.25),
				"High":  catalog.Float(0.75),
				"ChVol": catalog.Float(1),
			},
		},
		{
			// A rig counts every word from 0 to 1 whatever the device counts
			// the control in, so halfway up a control from -12 to 12 is 0.
			name:   "a control counted in something other than 0 to 1",
			params: map[string]catalog.Param{"Level": dial(catalog.ParamFloat, -12, 12)},
			set:    &rig.Settings{Level: knob(0.5)},
			want:   chain.Params{"Level": catalog.Float(0)},
		},
		{
			// A device handed 4.7 for a control counting whole steps refuses
			// the preset rather than rounding it.
			name:   "a control counted in whole steps",
			params: map[string]catalog.Param{"Level": dial(catalog.ParamInt, 0, 10)},
			set:    &rig.Settings{Level: knob(0.47)},
			want:   chain.Params{"Level": catalog.Int(5)},
		},
		{
			// Halfway up a switch is not a position, so the switch is not a
			// control this word can reach and the model is treated as having
			// none.
			name: "a switch by that name",
			params: map[string]catalog.Param{
				"Bright": {Type: catalog.ParamBool, Default: catalog.Bool(false)},
				"Drive":  dial(catalog.ParamFloat, 0, 1),
			},
			set:  &rig.Settings{Treble: knob(1)},
			errs: []string{`no "treble"`, "it has: drive"},
		},
		{
			// Every complaint at once. A rig with two words this model has no
			// control for took two runs to fix when this reported the first.
			name:   "words this model has no control for",
			params: map[string]catalog.Param{"Drive": dial(catalog.ParamFloat, 0, 1)},
			set:    &rig.Settings{Presence: knob(0.4), Mix: knob(0.2)},
			errs:   []string{`no "presence"`, `no "mix"`, "it has: drive"},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			blk := catalog.Block{ID: "HD2_Test", Params: tt.params}
			got := chain.Params{}

			err := compile.SetKnobs(got, blk, tt.set, "chain[0].settings")

			if len(tt.errs) > 0 {
				s.Require().Error(err)
				s.Require().ErrorIs(err, compile.ErrNoSuchValue)

				for _, want := range tt.errs {
					s.Require().Contains(err.Error(), want)
				}

				return
			}

			s.Require().NoError(err)
			s.Require().Equal(tt.want, got)
		})
	}
}

// TestLowerSetsWhatTheRigSaid covers the words reaching a compiled preset.
func (s *KnobsPublicTestSuite) TestLowerSetsWhatTheRigSaid() {
	spec := recipe("Ampeg SVT", "")
	spec.Chain[0].Settings = &rig.Settings{Drive: knob(0.47)}

	doc, err := preset.Blank()
	s.Require().NoError(err)
	s.Require().NoError(compile.Lower(doc, spec, s.cat))

	built, err := doc.Spec()
	s.Require().NoError(err)
	s.Require().Equal(catalog.Float(0.47), built.Blocks[0].Params["Drive"])
}

// TestLowerRefusesAWordTheGearHasNoControlFor covers the other outcome.
func (s *KnobsPublicTestSuite) TestLowerRefusesAWordTheGearHasNoControlFor() {
	spec := recipe("Ampeg SVT", "")
	spec.Chain[0].Settings = &rig.Settings{Presence: knob(0.4)}

	doc, err := preset.Blank()
	s.Require().NoError(err)
	s.Require().ErrorIs(compile.Lower(doc, spec, s.cat), compile.ErrNoSuchValue)
}

// TestResolveSetsWhatTheRigSaid covers the build path, where the words are
// the last thing to move a control.
func (s *KnobsPublicTestSuite) TestResolveSetsWhatTheRigSaid() {
	spec := recipe("Ampeg SVT", "")
	spec.Chain[0].Settings = &rig.Settings{Drive: knob(0.47)}

	built, _, _, err := compile.Resolve(spec, s.cat, nil)
	s.Require().NoError(err)
	s.Require().Equal(catalog.Float(0.47), built.Blocks[0].Params["Drive"])
}

// TestResolveRefusesAWordTheGearHasNoControlFor covers the same refusal on
// the build path, where the block is one the catalog chose.
func (s *KnobsPublicTestSuite) TestResolveRefusesAWordTheGearHasNoControlFor() {
	spec := recipe("Ampeg SVT", "")
	spec.Chain[0].Settings = &rig.Settings{Presence: knob(0.4)}

	_, _, _, err := compile.Resolve(spec, s.cat, nil)
	s.Require().ErrorIs(err, compile.ErrNoSuchValue)
}

func TestKnobsPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(KnobsPublicTestSuite))
}
