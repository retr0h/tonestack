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
package resolve_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/internal/resolve"
	"github.com/retr0h/tonestack/pkg/catalog"
	"github.com/retr0h/tonestack/pkg/chain"
	recipegen "github.com/retr0h/tonestack/pkg/recipe/gen"
)

type ResolvePublicTestSuite struct {
	suite.Suite

	cat *catalog.Catalog
}

func (s *ResolvePublicTestSuite) SetupSuite() {
	f, err := os.Open(filepath.Join("testdata", "catalog.json"))
	s.Require().NoError(err)

	defer func() { s.Require().NoError(f.Close()) }()

	s.cat, err = catalog.Load(f)
	s.Require().NoError(err)
}

// recipe returns a bass recipe naming amp, with optional cab and pedals.
func recipe(amp string, cab string, pedals ...string) *recipegen.Recipe {
	r := &recipegen.Recipe{
		ID:             "test",
		Kind:           recipegen.KindArtist,
		Name:           "Test Player",
		InstrumentType: recipegen.InstrumentBass,
		Rig:            recipegen.Rig{Amp: amp},
	}

	if cab != "" {
		r.Rig.Cab = &cab
	}

	if len(pedals) > 0 {
		r.Rig.Pedals = &pedals
	}

	return r
}

func (s *ResolvePublicTestSuite) TestResolvesGearToModels() {
	spec, _, err := resolve.Resolve(recipe("Ampeg SVT", ""), s.cat, nil)

	s.Require().NoError(err)
	s.Require().Equal("Test Player", spec.Name)
	s.Require().Len(spec.Blocks, 2, "an amp and the cabinet it is paired with")
	s.Require().Contains(string(spec.Blocks[0].Model), "SVBeast")
	s.Require().Equal(catalog.ModelID("HD2_Cab8x10SVBeast"), spec.Blocks[1].Model)
}

func (s *ResolvePublicTestSuite) TestAnAmbiguousRequestResolvesTheSameEveryTime() {
	// "Ampeg SVT" names neither the normal nor the bright channel. Whichever
	// is chosen, it must not change because the catalog was regenerated.
	first, _, err := resolve.Resolve(recipe("Ampeg SVT", ""), s.cat, nil)
	s.Require().NoError(err)

	for range 20 {
		again, _, err := resolve.Resolve(recipe("Ampeg SVT", ""), s.cat, nil)
		s.Require().NoError(err)
		s.Require().Equal(first.Blocks[0].Model, again.Blocks[0].Model)
	}
}

func (s *ResolvePublicTestSuite) TestTheIdentifierBreaksATie() {
	// Two pedals, descriptions of equal length. The identifier decides, so the
	// answer does not depend on map iteration order.
	spec, _, err := resolve.Resolve(recipe("Ampeg SVT", "", "Tied Pedal"), s.cat, nil)

	s.Require().NoError(err)
	s.Require().Equal(catalog.ModelID("HD2_TieA"), spec.Blocks[0].Model)
}

func (s *ResolvePublicTestSuite) TestTheCloserDescriptionWins() {
	// "Fuzz Face" matches both the Fuzz Face and the Fuzz Face Germanium
	// Reissue. The shorter description is the closer answer.
	spec, _, err := resolve.Resolve(recipe("Ampeg SVT", "", "Fuzz Face"), s.cat, nil)

	s.Require().NoError(err)
	s.Require().Equal(catalog.ModelID("HD2_Short"), spec.Blocks[0].Model)
}

func (s *ResolvePublicTestSuite) TestKeepsARequestInsideItsInstrument() {
	// A bass request must not reach a guitar amp, however well the name matches.
	_, _, err := resolve.Resolve(recipe("Marshall JCM-800", ""), s.cat, nil)

	s.Require().ErrorIs(err, resolve.ErrNoSuchGear)
	s.Require().Contains(err.Error(), "bass amps")
}

func (s *ResolvePublicTestSuite) TestOrdersPedalsBeforeTheAmp() {
	spec, _, err := resolve.Resolve(recipe("Ampeg SVT", "", "Klon Centaur"), s.cat, nil)

	s.Require().NoError(err)
	s.Require().Len(spec.Blocks, 3)
	s.Require().Equal(catalog.ModelID("HD2_DistMinotaur"), spec.Blocks[0].Model)
	s.Require().Contains(string(spec.Blocks[1].Model), "SVBeast")
	s.Require().Equal(catalog.ModelID("HD2_Cab8x10SVBeast"), spec.Blocks[2].Model)
}

func (s *ResolvePublicTestSuite) TestPrefersACabinetTheRecipeNames() {
	spec, _, err := resolve.Resolve(recipe("Ampeg SVT", "Ampeg SVT 410HLF"), s.cat, nil)

	s.Require().NoError(err)
	s.Require().Equal(catalog.ModelID("HD2_CabNamed"), spec.Blocks[1].Model)
}

func (s *ResolvePublicTestSuite) TestFallsBackToTheAmpsOwnCabinet() {
	// Line 6 does not describe every cabinet in terms of real gear, so a
	// cabinet it cannot name is not a reason to refuse to build.
	spec, _, err := resolve.Resolve(recipe("Ampeg SVT", "Some Cabinet Nobody Models"), s.cat, nil)

	s.Require().NoError(err)
	s.Require().Equal(catalog.ModelID("HD2_Cab8x10SVBeast"), spec.Blocks[1].Model)
}

func (s *ResolvePublicTestSuite) TestAnAmpWithNoPairedCabinet() {
	spec, _, err := resolve.Resolve(recipe("Ampeg SVT (bright", ""), s.cat, nil)

	s.Require().NoError(err)
	s.Require().NotEmpty(spec.Blocks)
}

func (s *ResolvePublicTestSuite) TestAnAmpThatNamesNoCabinet() {
	spec, _, err := resolve.Resolve(recipe("Cabless Bass Head", ""), s.cat, nil)

	s.Require().NoError(err)
	s.Require().Len(spec.Blocks, 1, "a chain without a cabinet is still a chain")
}

func (s *ResolvePublicTestSuite) TestAnAmpNamingACabinetThisDeviceLacks() {
	spec, _, err := resolve.Resolve(recipe("Dangling Bass Head", ""), s.cat, nil)

	s.Require().NoError(err)
	s.Require().Len(spec.Blocks, 1)
}

func (s *ResolvePublicTestSuite) TestABlockNeedingTheOwnersOwnIRIsNeverChosen() {
	// A user IR block carries a slot index, not audio. Generating one would
	// point at whatever happened to be loaded in that slot, so the resolver
	// must not reach for it even when the name matches.
	_, _, err := resolve.Resolve(recipe("Slotted Cab", ""), s.cat, nil)

	s.Require().Error(err)
}

func (s *ResolvePublicTestSuite) TestFitLeavesOverflowAloneOnASingleChipDevice() {
	// Regression: overflow was moved to dsp1 unconditionally. An HX Stomp has
	// one signal path — no preset in a corpus of 714 has a second — so a
	// block put there produces a file the device cannot load. A chain that
	// does not fit must stay put and be rejected by validation instead.
	spec, _, err := resolve.Resolve(recipe("Ampeg SVT", "", "Wide Thing"), s.cat, nil)
	s.Require().NoError(err)

	fitted := resolve.Fit(spec, s.cat, oneChip(80.0))

	for _, b := range fitted.Blocks {
		s.Require().Zero(b.DSP,
			"a device with one signal path has nowhere to put overflow")
	}
}

func (s *ResolvePublicTestSuite) TestFitChargesAStereoBlockItsStereoCost() {
	spec, _, err := resolve.Resolve(recipe("Ampeg SVT", "", "Wide Thing"), s.cat, nil)
	s.Require().NoError(err)

	// 50 stereo + 26.67 + 7.2 overflows 80; 5 mono + the rest would not.
	fitted := resolve.Fit(spec, s.cat, twoChips(80.0))

	var second int

	for _, b := range fitted.Blocks {
		if b.DSP == 1 {
			second++
		}
	}

	s.Require().Positive(second, "a stereo block costs its stereo figure")
}

func (s *ResolvePublicTestSuite) TestReportsGearNoModelEmulates() {
	_, _, err := resolve.Resolve(recipe("Orange Rockerverb", ""), s.cat, nil)

	s.Require().ErrorIs(err, resolve.ErrNoSuchGear)
	s.Require().Contains(err.Error(), "Orange Rockerverb")
}

func (s *ResolvePublicTestSuite) TestReportsAPedalNoModelEmulates() {
	_, _, err := resolve.Resolve(recipe("Ampeg SVT", "", "Nonexistent Fuzz"), s.cat, nil)

	s.Require().ErrorIs(err, resolve.ErrNoSuchGear)
	s.Require().Contains(err.Error(), "Nonexistent Fuzz")
}

func (s *ResolvePublicTestSuite) TestSetsEveryParameterToItsStatedDefault() {
	spec, _, err := resolve.Resolve(recipe("Ampeg SVT", ""), s.cat, nil)
	s.Require().NoError(err)

	amp := spec.Blocks[0]
	blk, ok := s.cat.Block(amp.Model)
	s.Require().True(ok)

	want, ok := blk.Params["Drive"].Default.Float()
	s.Require().True(ok)

	got, ok := amp.Params["Drive"].Float()
	s.Require().True(ok)
	s.Require().InDelta(want, got, 1e-9,
		"every parameter starts at what Line 6 states")
}

func (s *ResolvePublicTestSuite) TestSkipsAParameterWithNoStatedDefault() {
	// A value with no kind produces a preset the device rejects.
	spec, _, err := resolve.Resolve(recipe("Ampeg SVT", "", "Nothing Real"), s.cat, nil)

	s.Require().NoError(err)
	s.Require().Empty(spec.Blocks[0].Params)
}

func (s *ResolvePublicTestSuite) TestFitKeepsASmallChainOnOneProcessor() {
	spec, _, _ := resolve.Resolve(recipe("Ampeg SVT", ""), s.cat, nil)

	fitted := resolve.Fit(spec, s.cat, twoChips(95.0))

	for _, b := range fitted.Blocks {
		s.Require().Equal(0, b.DSP)
	}
}

func (s *ResolvePublicTestSuite) TestFitMovesOverflowToTheSecondProcessor() {
	spec, _, _ := resolve.Resolve(recipe("Ampeg SVT", "", "Heavy Thing", "Heavy Thing"), s.cat, nil)

	// 60 + 60 + 26.67 + 7.2 cannot fit under 95 on one chip.
	fitted := resolve.Fit(spec, s.cat, twoChips(95.0))

	var second int

	for _, b := range fitted.Blocks {
		if b.DSP == 1 {
			second++
		}
	}

	s.Require().Positive(second, "something must move to the second processor")
}

func (s *ResolvePublicTestSuite) TestFitNumbersEachProcessorFromZero() {
	spec, _, _ := resolve.Resolve(recipe("Ampeg SVT", "", "Heavy Thing", "Heavy Thing"), s.cat, nil)

	fitted := resolve.Fit(spec, s.cat, twoChips(95.0))

	seen := map[int]map[int]bool{}

	for _, b := range fitted.Blocks {
		if seen[b.DSP] == nil {
			seen[b.DSP] = map[int]bool{}
		}

		s.Require().False(seen[b.DSP][b.Pos], "position %d used twice on chip %d", b.Pos, b.DSP)
		seen[b.DSP][b.Pos] = true
	}

	for dsp, positions := range seen {
		for i := range positions {
			s.Require().True(positions[i], "chip %d has a gap at %d", dsp, i)
		}
	}
}

func (s *ResolvePublicTestSuite) TestFitIgnoresABlockTheCatalogLacks() {
	spec, _, _ := resolve.Resolve(recipe("Ampeg SVT", ""), s.cat, nil)
	spec.Blocks[0].Model = "HD2_NotInThisCatalog"

	fitted := resolve.Fit(spec, s.cat, twoChips(95.0))

	s.Require().Len(fitted.Blocks, len(spec.Blocks))
}

func (s *ResolvePublicTestSuite) TestNoSuchGearErrorReadsWell() {
	withInstrument := &resolve.NoSuchGearError{Gear: "Orange", Kind: "amp", Instrument: "bass"}
	s.Require().Contains(withInstrument.Error(), "bass amps")

	anywhere := &resolve.NoSuchGearError{Gear: "Orange", Kind: "block"}
	s.Require().Contains(anywhere.Error(), "this device's catalog")
	s.Require().ErrorIs(anywhere, resolve.ErrNoSuchGear)
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
