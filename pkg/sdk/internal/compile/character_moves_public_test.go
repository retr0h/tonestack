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
	"github.com/retr0h/tonestack/pkg/sdk/rig"
)

// CharacterMovesPublicTestSuite covers the words a rig uses reaching the
// amplifier, which is the whole reason the vocabulary exists.
type CharacterMovesPublicTestSuite struct {
	suite.Suite

	cat *catalog.Catalog
}

func (s *CharacterMovesPublicTestSuite) SetupTest() {
	s.cat = loadCatalog(&s.Suite)
}

// described returns a rig that says how it should sound.
func described(amp string, terms ...string) rig.Spec {
	spec := recipe(amp, "")

	got := make([]rig.CharacterTerm, 0, len(terms))
	for _, t := range terms {
		got = append(got, rig.CharacterTerm{Term: t})
	}

	spec.Character = &got

	return spec
}

// TestCharacterReachesTheAmplifier covers a word turning a real knob.
//
// Drive, because it is the one control every amplifier in this fixture
// carries. Which parameter each axis moves is covered against a block built
// for it in move_test.go; what this proves is that a rig's words reach the
// amplifier at all.
func (s *CharacterMovesPublicTestSuite) TestCharacterReachesTheAmplifier() {
	tests := []struct {
		name string
		term string
		up   bool
	}{
		{name: "pushed harder", term: "saturated", up: true},
		{name: "backed off", term: "clean"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			plain, _, _, err := compile.Resolve(recipe("Ampeg SVT", ""), s.cat, nil)
			s.Require().NoError(err)

			got, _, moved, err := compile.Resolve(
				described("Ampeg SVT", tt.term), s.cat, nil)
			s.Require().NoError(err)

			s.Require().Len(moved, 1)
			s.Require().True(moved[0].Acted())
			s.Require().Equal("Drive", moved[0].Param)

			was := s.paramOf(plain, "Drive")
			now := s.paramOf(got, "Drive")

			if tt.up {
				s.Require().Greater(now, was)

				return
			}

			s.Require().Less(now, was)
		})
	}
}

// paramOf reads one control off whichever block is the amplifier.
func (s *CharacterMovesPublicTestSuite) paramOf(
	built chain.Chain,
	key string,
) float64 {
	for _, b := range built.Blocks {
		blk, ok := s.cat.Block(b.Model)
		if !ok || blk.Category != catalog.CategoryAmp {
			continue
		}

		v, ok := b.Params[key].Float()
		s.Require().True(ok, "the amplifier has no %s", key)

		return v
	}

	s.Require().Fail("no amplifier in the chain")

	return 0
}

// TestARigThatSaysNothingMovesNothing covers the silent case.
func (s *CharacterMovesPublicTestSuite) TestARigThatSaysNothingMovesNothing() {
	_, _, moved, err := compile.Resolve(recipe("Ampeg SVT", ""), s.cat, nil)

	s.Require().NoError(err)
	s.Require().Empty(moved)
}

// TestWordsSurviveAChainWithNoAmplifier covers a rig describing a sound it
// has nowhere to make.
//
// The words are still what the rig said, so they are reported as moving
// nothing rather than dropped.
func (s *CharacterMovesPublicTestSuite) TestWordsSurviveAChainWithNoAmplifier() {
	spec := rig.Spec{
		Schema:     rig.SchemaName,
		ID:         "test",
		Subject:    rig.Subject{Kind: rig.KindArtist, Name: "Test Player"},
		Instrument: rig.InstrumentBass,
		Chain: []rig.ChainEntry{
			{Role: rig.RoleOther, Gear: "Klon Centaur"},
		},
		Character: &[]rig.CharacterTerm{{Term: "mid-forward"}},
	}

	_, _, moved, err := compile.Resolve(spec, s.cat, nil)

	s.Require().NoError(err)
	s.Require().Len(moved, 1)
	s.Require().False(moved[0].Acted())
}

// TestTwoWordsForOneAxisMoveNothing covers a rig answering one question twice.
func (s *CharacterMovesPublicTestSuite) TestTwoWordsForOneAxisMoveNothing() {
	plain, _, _, err := compile.Resolve(recipe("Ampeg SVT", ""), s.cat, nil)
	s.Require().NoError(err)

	got, _, moved, err := compile.Resolve(
		described("Ampeg SVT", "minimal-drive", "grit-on-attack"), s.cat, nil)
	s.Require().NoError(err)

	s.Require().Len(moved, 2)

	for _, m := range moved {
		s.Require().True(m.Contested())
		s.Require().Equal("drive", m.Against)
	}

	s.Require().InDelta(
		s.paramOf(plain, "Drive"), s.paramOf(got, "Drive"), 1e-9)
}

func TestCharacterMovesPublicTestSuite(t *testing.T) {
	suite.Run(t, new(CharacterMovesPublicTestSuite))
}
