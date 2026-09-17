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
package cli_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/cli"
	"github.com/retr0h/tonestack/pkg/sdk/audio"
)

// PlayersPublicTestSuite covers the table of what each player's records earn.
type PlayersPublicTestSuite struct {
	suite.Suite
}

// render draws a set of players and hands back what was written.
func (s *PlayersPublicTestSuite) render(
	of []audio.Player,
) string {
	var buf bytes.Buffer

	s.Require().NoError(cli.Players(&buf, of))

	return buf.String()
}

// TestAPlayerAndWhatEarnedTheirWord is the whole contract: the word, and the
// figures on both sides of the comparison that produced it.
func (s *PlayersPublicTestSuite) TestAPlayerAndWhatEarnedTheirWord() {
	got := s.render([]audio.Player{
		{
			ID:      "pino-palladino",
			Records: 3,
			Terms: []audio.Derived{
				{Term: "dark", Key: audio.KeyCentroid, Mine: 96, Others: 170, Of: 5},
			},
		},
	})

	s.Require().Contains(got, "pino-palladino")
	s.Require().Contains(got, "dark")
	s.Require().Contains(got, "96 Hz against 170 Hz",
		"a centroid is hertz, and a share printed as hertz reads as nonsense")
}

// TestAShareIsPrintedAsAShare covers the other unit.
func (s *PlayersPublicTestSuite) TestAShareIsPrintedAsAShare() {
	got := s.render([]audio.Player{
		{
			ID:      "flea",
			Records: 3,
			Terms: []audio.Derived{
				{Term: "clean", Key: audio.KeyHarmonics, Mine: 0.12, Others: 0.24, Of: 5},
			},
		},
	})

	s.Require().Contains(got, "12% against 24%")
}

// TestSeveralWordsAreAllShown covers a player clear on more than one axis.
func (s *PlayersPublicTestSuite) TestSeveralWordsAreAllShown() {
	got := s.render([]audio.Player{
		{
			ID:      "somebody",
			Records: 2,
			Terms: []audio.Derived{
				{Term: "mid-forward", Key: audio.KeyMid, Mine: 0.09, Others: 0.02, Of: 3},
				{Term: "bright", Key: audio.KeyCentroid, Mine: 400, Others: 150, Of: 3},
			},
		},
	})

	s.Require().Contains(got, "mid-forward, bright")
	s.Require().Contains(got, "9% against 2%")
	s.Require().Contains(got, "400 Hz against 150 Hz")
}

// TestAPlayerWhoEarnedNothingSaysSo covers the ordinary answer.
//
// Mixed evidence is not a failure, and a blank cell reads as one.
func (s *PlayersPublicTestSuite) TestAPlayerWhoEarnedNothingSaysSo() {
	got := s.render([]audio.Player{{ID: "les-claypool", Records: 3}})

	s.Require().Contains(got, "nothing")
}

// TestOnePlayerIsNobodyToCompareAgainst covers the count reading as the
// reason nothing was earned.
func (s *PlayersPublicTestSuite) TestOnePlayerIsNobodyToCompareAgainst() {
	got := s.render([]audio.Player{{ID: "mike-dirnt", Records: 3}})

	s.Require().Contains(got, "1 player, which is nobody to compare against")
}

// TestSeveralPlayersAreCounted covers the ordinary heading.
func (s *PlayersPublicTestSuite) TestSeveralPlayersAreCounted() {
	got := s.render([]audio.Player{
		{ID: "flea", Records: 3},
		{ID: "mike-dirnt", Records: 3},
	})

	s.Require().Contains(got, "2 players")
}

// TestNoPlayersAtAll covers the empty table saying what to do about it.
func (s *PlayersPublicTestSuite) TestNoPlayersAtAll() {
	got := s.render(nil)

	s.Require().Contains(got, "no players to compare")
}

func TestPlayersPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(PlayersPublicTestSuite))
}
