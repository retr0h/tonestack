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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/chain"
	"github.com/retr0h/tonestack/pkg/sdk/internal/compile"
	riggen "github.com/retr0h/tonestack/pkg/sdk/rig/gen"
)

type ResolvePublicTestSuite struct {
	suite.Suite

	cat *catalog.Catalog
}

func (s *ResolvePublicTestSuite) SetupSuite() {
	s.cat = loadCatalog(&s.Suite)
}

// loadCatalog reads the fixture catalog every suite in this package builds
// against.
func loadCatalog(s *suite.Suite) *catalog.Catalog {
	f, err := os.Open(filepath.Join("testdata", "catalog.json"))
	s.Require().NoError(err)

	defer func() { s.Require().NoError(f.Close()) }()

	cat, err := catalog.Load(f)
	s.Require().NoError(err)

	return cat
}

// recipe returns a bass rig naming amp, with optional cab and pedals.
//
// Pedals are written ahead of the amp because that is where the tests mean
// them to be: a chain is ordered by what the signal does, and nothing
// downstream reorders it.
//
// They carry the `other` role rather than a guess, because these fixtures do
// not say what the pedals are and stating a role they do not have would test
// the wrong thing.
func recipe(amp string, cab string, pedals ...string) riggen.RigSpec {
	spec := riggen.RigSpec{
		Schema:     riggen.RigSpecSchemaRigSpec,
		ID:         "test",
		Subject:    riggen.Subject{Kind: riggen.KindArtist, Name: "Test Player"},
		Instrument: riggen.InstrumentBass,
	}

	for _, p := range pedals {
		spec.Chain = append(spec.Chain,
			riggen.ChainEntry{Role: riggen.RoleOther, Gear: p})
	}

	spec.Chain = append(spec.Chain,
		riggen.ChainEntry{Role: riggen.RoleAmp, Gear: amp})

	if cab != "" {
		spec.Chain = append(spec.Chain,
			riggen.ChainEntry{Role: riggen.RoleCab, Gear: cab})
	}

	return spec
}

// TestResolve turns gear a person names into models a device has.
//
// "Ampeg SVT" names neither the normal nor the bright channel, so which of
// the two comes back is arbitrary. It is pinned here because a row wants a
// value, and TestResolveIsDeterministic is what guards that it stays put.
// substituting names gear this catalog has no model for, and says what to put
// there instead.
func substituting(gear, instead string) riggen.RigSpec {
	spec := recipe("Ampeg SVT", "")
	spec.Chain[len(spec.Chain)-1] = riggen.ChainEntry{
		Role: riggen.RoleAmp,
		Gear: gear,
	}

	if instead != "" {
		spec.Chain[len(spec.Chain)-1].Substitute = &riggen.Substitute{Gear: instead}
	}

	return spec
}

func (s *ResolvePublicTestSuite) TestResolve() {
	tests := []struct {
		name   string
		spec   riggen.RigSpec
		models []catalog.ModelID
		err    string
	}{
		{
			name:   "an amp brings the cabinet it was voiced with",
			spec:   recipe("Ampeg SVT", ""),
			models: []catalog.ModelID{"HD2_AmpSVBeastBrt", "HD2_Cab8x10SVBeast"},
		},
		{
			name: "the chain keeps the order it was written in",
			spec: recipe("Ampeg SVT", "", "Klon Centaur"),
			models: []catalog.ModelID{
				"HD2_DistMinotaur",
				"HD2_AmpSVBeastBrt",
				"HD2_Cab8x10SVBeast",
			},
		},
		{
			// Two pedals, descriptions of equal length. The identifier
			// decides, so the answer does not depend on map iteration order.
			name:   "the identifier breaks a tie",
			spec:   recipe("Ampeg SVT", "", "Tied Pedal"),
			models: []catalog.ModelID{"HD2_TieA", "HD2_AmpSVBeastBrt", "HD2_Cab8x10SVBeast"},
		},
		{
			// "Fuzz Face" matches both the Fuzz Face and the Fuzz Face
			// Germanium Reissue. The shorter description is the closer answer.
			name:   "the closer description wins",
			spec:   recipe("Ampeg SVT", "", "Fuzz Face"),
			models: []catalog.ModelID{"HD2_Short", "HD2_AmpSVBeastBrt", "HD2_Cab8x10SVBeast"},
		},
		{
			name:   "a cabinet the recipe names beats the amp's own",
			spec:   recipe("Ampeg SVT", "Ampeg SVT 410HLF"),
			models: []catalog.ModelID{"HD2_AmpSVBeastBrt", "HD2_CabNamed"},
		},
		{
			// A rig outlives any one device, so it goes on naming what was
			// really played and says separately what this device can do.
			name:   "gear nobody models, with a stand-in the rig names",
			spec:   substituting("Orange AD200B", "Ampeg SVT"),
			models: []catalog.ModelID{"HD2_AmpSVBeastBrt", "HD2_Cab8x10SVBeast"},
		},
		{
			name: "a stand-in nobody models either",
			spec: substituting("Orange AD200B", "Also Not A Thing"),
			err:  `"Also Not A Thing" stands in for "Orange AD200B"`,
		},
		{
			// Substituting an amplifier is not a detail, so without one the
			// build fails rather than picking something.
			name: "gear nobody models and no stand-in",
			spec: substituting("Orange AD200B", ""),
			err:  `no amp in this device's bass amps emulates "Orange AD200B"`,
		},
		{
			// Line 6 does not describe every cabinet in terms of real gear,
			// so one it cannot name is not a reason to refuse to build.
			name:   "a cabinet nobody models falls back to the amp's",
			spec:   recipe("Ampeg SVT", "Some Cabinet Nobody Models"),
			models: []catalog.ModelID{"HD2_AmpSVBeastBrt", "HD2_Cab8x10SVBeast"},
		},
		{
			name:   "a partial name reaching the other channel",
			spec:   recipe("Ampeg SVT (bright", ""),
			models: []catalog.ModelID{"HD2_AmpSVBeastBrt", "HD2_Cab8x10SVBeast"},
		},
		{
			name:   "an amp that names none at all",
			spec:   recipe("Cabless Bass Head", ""),
			models: []catalog.ModelID{"HD2_AmpNoCab"},
		},
		{
			name:   "an amp naming a cabinet this device lacks",
			spec:   recipe("Dangling Bass Head", ""),
			models: []catalog.ModelID{"HD2_AmpDanglingCab"},
		},
		{
			// A bass request must not reach a guitar amp, however well the
			// name matches.
			name: "a request stays inside its instrument",
			spec: recipe("Marshall JCM-800", ""),
			err:  "bass amps",
		},
		{
			// A cabinet miss is only recoverable because the amplifier names
			// the one it was voiced with. One that names none leaves nothing
			// to substitute.
			name: "a cabinet miss with nothing to fall back to",
			spec: recipe("Cabless Bass Head", "Some Cabinet Nobody Models"),
			err:  "Some Cabinet Nobody Models",
		},
		{
			name: "gear no model emulates",
			spec: recipe("Orange Rockerverb", ""),
			err:  "Orange Rockerverb",
		},
		{
			name: "a pedal no model emulates",
			spec: recipe("Ampeg SVT", "", "Nonexistent Fuzz"),
			err:  "Nonexistent Fuzz",
		},
		{
			// A user IR block carries a slot index, not audio. Generating one
			// would point at whatever happened to be loaded in that slot.
			name: "a block needing the owner's own impulse response",
			spec: recipe("Slotted Cab", ""),
			err:  "Slotted Cab",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, _, err := compile.Resolve(tt.spec, s.cat, nil)

			if tt.err != "" {
				s.Require().ErrorIs(err, compile.ErrNoSuchGear)
				s.Require().Contains(err.Error(), tt.err)

				return
			}

			s.Require().NoError(err)
			s.Require().Equal("Test Player", got.Name)
			s.Require().Equal(tt.models, models(got))
		})
	}
}

// TestResolveIsDeterministic is a property rather than a case.
//
// "Ampeg SVT" names neither the normal nor the bright channel. Whichever is
// chosen, it must not change because the catalog was regenerated or because
// a map iterated in a different order.
// TestGear resolves one name, the way both halves of this project now do.
//
// Lowering a rig into a preset used to answer this question itself, with a map
// range that took whatever matched first. Two resolvers cannot both be right
// about which Ampeg SVT is meant, and the one that ignored the role would
// answer with a cabinet.
// TestResolveChecksWhatTheRigClaims covers the check reaching the caller.
//
// Resolve builds the chain and then asks whether this catalog can supply what
// the rig says beside it, so a recipe naming another device's hardware fails
// here rather than at the pedal.
func (s *ResolvePublicTestSuite) TestResolveChecksWhatTheRigClaims() {
	spec := recipe("Ampeg SVT (normal", "")
	device := "Kemper Profiler"
	spec.Target = &riggen.Target{Device: &device}

	_, _, err := compile.Resolve(spec, s.cat, nil)

	s.Require().ErrorIs(err, compile.ErrNoSuchValue)
}

func (s *ResolvePublicTestSuite) TestGear() {
	tests := []struct {
		name       string
		gear       string
		role       riggen.Role
		instrument string
		want       catalog.ModelID
		err        error
	}{
		{
			name:       "an amplifier by name",
			gear:       "Ampeg SVT (normal",
			role:       riggen.RoleAmp,
			instrument: "bass",
			want:       "HD2_AmpSVBeastNrm",
		},
		{
			// The role is half the question: this catalog holds a cabinet
			// whose name matches an amplifier's, and asking for one must not
			// answer with the other.
			name:       "a cabinet by the name it shares",
			gear:       "Ampeg SVT",
			role:       riggen.RoleCab,
			instrument: "bass",
			want:       "HD2_Cab8x10SVBeast",
		},
		{
			name:       "gear for the other instrument",
			gear:       "Guitar Only",
			role:       riggen.RoleAmp,
			instrument: "bass",
			err:        compile.ErrNoSuchGear,
		},
		{
			name:       "gear nothing emulates",
			gear:       "Nonesuch 900",
			role:       riggen.RoleAmp,
			instrument: "bass",
			err:        compile.ErrNoSuchGear,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			// Twice, because the answer used to depend on which way a map
			// ranged.
			for range 2 {
				got, err := compile.Gear(s.cat, tt.gear, tt.role, tt.instrument)

				if tt.err != nil {
					s.Require().ErrorIs(err, tt.err)

					continue
				}

				s.Require().NoError(err)
				s.Require().Equal(tt.want, got.ID)
			}
		})
	}
}

func (s *ResolvePublicTestSuite) TestResolveIsDeterministic() {
	first, _, err := compile.Resolve(recipe("Ampeg SVT", ""), s.cat, nil)
	s.Require().NoError(err)

	for range 20 {
		again, _, err := compile.Resolve(recipe("Ampeg SVT", ""), s.cat, nil)

		s.Require().NoError(err)
		s.Require().Equal(models(first), models(again))
	}
}

// TestResolveNamesWhatItChoseForYou covers the second return, which is what
// a person is told about decisions made on their behalf.
func (s *ResolvePublicTestSuite) TestResolveNamesWhatItChoseForYou() {
	_, added, err := compile.Resolve(
		recipe("Ampeg SVT", "Some Cabinet Nobody Models"), s.cat, nil)

	s.Require().NoError(err)
	s.Require().NotEmpty(added, "a substitution is a choice made for somebody")
	s.Require().Contains(added[0].Reason, "Some Cabinet Nobody Models")
	s.Require().Zero(added[0].Share, "a substitution is not a measurement")
}

// TestResolveSetsParameters covers the values a block starts at.
func (s *ResolvePublicTestSuite) TestResolveSetsParameters() {
	tests := []struct {
		name  string
		spec  riggen.RigSpec
		check func(chain.Block)
	}{
		{
			name: "every parameter starts at what Line 6 states",
			spec: recipe("Ampeg SVT", ""),
			check: func(b chain.Block) {
				blk, ok := s.cat.Block(b.Model)
				s.Require().True(ok)

				want, ok := blk.Params["Drive"].Default.Float()
				s.Require().True(ok)

				got, ok := b.Params["Drive"].Float()
				s.Require().True(ok)
				s.Require().InDelta(want, got, 1e-9)
			},
		},
		{
			// A value with no kind produces a preset the device rejects.
			name: "a parameter with no stated default is skipped",
			spec: recipe("Ampeg SVT", "", "Nothing Real"),
			check: func(b chain.Block) {
				s.Require().Empty(b.Params)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, _, err := compile.Resolve(tt.spec, s.cat, nil)

			s.Require().NoError(err)
			s.Require().NotEmpty(got.Blocks)

			tt.check(got.Blocks[0])
		})
	}
}

// TestFit places a chain across the processors a device has.
func (s *ResolvePublicTestSuite) TestFit() {
	tests := []struct {
		name    string
		spec    riggen.RigSpec
		limits  chain.Limits
		spilled bool
	}{
		{
			name:   "a small chain stays on the first processor",
			spec:   recipe("Ampeg SVT", ""),
			limits: twoChips(95.0),
		},
		{
			// 60 + 60 + 26.67 + 7.2 cannot fit under 95 on one chip.
			name:    "overflow moves to the second",
			spec:    recipe("Ampeg SVT", "", "Heavy Thing", "Heavy Thing"),
			limits:  twoChips(95.0),
			spilled: true,
		},
		{
			// 50 stereo + 26.67 + 7.2 overflows 80; 5 mono and the rest
			// would not, so this is the stereo figure being charged.
			name:    "a stereo block costs its stereo figure",
			spec:    recipe("Ampeg SVT", "", "Wide Thing"),
			limits:  twoChips(80.0),
			spilled: true,
		},
		{
			// Regression: overflow was moved to dsp1 unconditionally. An HX
			// Stomp has one signal path — no preset in a corpus of 714 has a
			// second — so a block put there produces a file the device cannot
			// load. A chain that does not fit stays put and is rejected by
			// validation instead.
			name:   "a device with one path has nowhere to put overflow",
			spec:   recipe("Ampeg SVT", "", "Wide Thing"),
			limits: oneChip(80.0),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			spec, _, err := compile.Resolve(tt.spec, s.cat, nil)
			s.Require().NoError(err)

			fitted := compile.Fit(spec, s.cat, tt.limits)

			var second int

			for _, b := range fitted.Blocks {
				if b.DSP == 1 {
					second++
				}
			}

			if tt.spilled {
				s.Require().Positive(second, "something must move")

				return
			}

			s.Require().Zero(second, "nothing should have moved")
		})
	}
}

// TestFitNumbersEachProcessorFromZero is a property of the whole result
// rather than of any one chain.
func (s *ResolvePublicTestSuite) TestFitNumbersEachProcessorFromZero() {
	spec, _, err := compile.Resolve(
		recipe("Ampeg SVT", "", "Heavy Thing", "Heavy Thing"), s.cat, nil)
	s.Require().NoError(err)

	seen := map[int]map[int]bool{}

	for _, b := range compile.Fit(spec, s.cat, twoChips(95.0)).Blocks {
		if seen[b.DSP] == nil {
			seen[b.DSP] = map[int]bool{}
		}

		s.Require().False(seen[b.DSP][b.Pos],
			"position %d used twice on chip %d", b.Pos, b.DSP)
		seen[b.DSP][b.Pos] = true
	}

	for dsp, positions := range seen {
		for i := range positions {
			s.Require().True(positions[i], "chip %d has a gap at %d", dsp, i)
		}
	}
}

// TestFitIgnoresABlockTheCatalogLacks keeps a catalog from another release
// from dropping blocks on the floor.
func (s *ResolvePublicTestSuite) TestFitIgnoresABlockTheCatalogLacks() {
	spec, _, err := compile.Resolve(recipe("Ampeg SVT", ""), s.cat, nil)
	s.Require().NoError(err)

	spec.Blocks[0].Model = "HD2_NotInThisCatalog"

	s.Require().Len(
		compile.Fit(spec, s.cat, twoChips(95.0)).Blocks, len(spec.Blocks))
}

// TestNoSuchGearError covers what somebody reads when nothing matched.
func (s *ResolvePublicTestSuite) TestNoSuchGearError() {
	tests := []struct {
		name string
		err  *compile.NoSuchGearError
		want string
	}{
		{
			name: "a miss inside one instrument's half of the catalog",
			err:  &compile.NoSuchGearError{Gear: "Orange", Kind: "amp", Instrument: "bass"},
			want: "bass amps",
		},
		{
			name: "one with nowhere left to look",
			err:  &compile.NoSuchGearError{Gear: "Orange", Kind: "block"},
			want: "this device's catalog",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().Contains(tt.err.Error(), tt.want)
			s.Require().ErrorIs(tt.err, compile.ErrNoSuchGear)
		})
	}
}

// models names what a chain resolved to, in order.
func models(c chain.Chain) []catalog.ModelID {
	out := make([]catalog.ModelID, 0, len(c.Blocks))
	for _, b := range c.Blocks {
		out = append(out, b.Model)
	}

	return out
}

func TestResolvePublicTestSuite(t *testing.T) {
	suite.Run(t, new(ResolvePublicTestSuite))
}

// twoChips is a device with somewhere to put overflow.
func twoChips(ceiling float64) chain.Limits {
	return chain.Limits{MaxBlocks: 8, Paths: 2, ChipCeiling: ceiling}
}

// oneChip is a device with a single signal path, like the HX Stomp.
func oneChip(ceiling float64) chain.Limits {
	return chain.Limits{MaxBlocks: 8, Paths: 1, ChipCeiling: ceiling}
}
