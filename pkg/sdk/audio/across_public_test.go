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

package audio_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/audio"
)

// AcrossPublicTestSuite covers several recordings read as one measurement.
type AcrossPublicTestSuite struct {
	suite.Suite
}

// rang is a decay the recording actually gave up.
func rang(
	v float64,
) audio.Reading {
	return audio.Reading{Value: v, Known: true}
}

// records is four profiles that disagree by a known amount.
func (s *AcrossPublicTestSuite) records() []audio.Profile {
	return []audio.Profile{
		{Centroid: 135, Low: 0.97, Decay: rang(0.15), Harmonics: audio.Spread{Mid: 0.18}},
		{Centroid: 148, Low: 0.97, Decay: rang(1.72), Harmonics: audio.Spread{Mid: 0.23}},
		{Centroid: 152, Low: 0.94, Decay: rang(0.82), Harmonics: audio.Spread{Mid: 0.35}},
		{Centroid: 191, Low: 0.91, Decay: rang(0.88), Harmonics: audio.Spread{Mid: 0.20}},
	}
}

// TestItCountsWhatWentIn covers the number of recordings reaching the answer.
func (s *AcrossPublicTestSuite) TestItCountsWhatWentIn() {
	s.Require().Equal(4, audio.Together(s.records()).Tracks)
}

// TestTheWidthIsTheRecordsDisagreeing is the whole point of gathering them.
//
// Four records is too few for the ends to be a tenth in from anything, so the
// ends are the extreme records. That is stated rather than hidden, and this
// holds it to it.
func (s *AcrossPublicTestSuite) TestTheWidthIsTheRecordsDisagreeing() {
	got := audio.Together(s.records())

	s.Require().InDelta(135, got.Centroid.Low, 0.001, "the lowest record")
	s.Require().InDelta(191, got.Centroid.High, 0.001, "and the highest")
	s.Require().GreaterOrEqual(got.Centroid.Mid, 135.0)
	s.Require().LessOrEqual(got.Centroid.Mid, 191.0)
}

// TestARecordThatDisagreesDoesNotBecomeTheAnswer covers the middle holding.
//
// One stem measured a decay of 0.15s where the others rang for about a
// second, because its take is half silence. Gathering four records should
// leave that as one voice rather than the result.
func (s *AcrossPublicTestSuite) TestARecordThatDisagreesDoesNotBecomeTheAnswer() {
	got := audio.Together(s.records())

	s.Require().Greater(got.Decay.Mid, 0.5,
		"one short take does not decide what the player's notes do")
}

// TestATracksOwnRangeContributesItsMiddle covers spreads folding into one
// number each.
func (s *AcrossPublicTestSuite) TestATracksOwnRangeContributesItsMiddle() {
	got := audio.Together([]audio.Profile{
		{Harmonics: audio.Spread{Low: 0.01, Mid: 0.20, High: 0.90}},
		{Harmonics: audio.Spread{Low: 0.02, Mid: 0.40, High: 0.95}},
	})

	s.Require().InDelta(0.20, got.Harmonics.Low, 0.001,
		"the ends come from the middles, not from the widest window anywhere")
	s.Require().InDelta(0.40, got.Harmonics.High, 0.001)
}

// TestOneRecordingIsItsOwnMiddle covers a corpus of one.
func (s *AcrossPublicTestSuite) TestOneRecordingIsItsOwnMiddle() {
	got := audio.Together([]audio.Profile{{Centroid: 150}})

	s.Require().Equal(1, got.Tracks)
	s.Require().InDelta(150, got.Centroid.Low, 0.001)
	s.Require().InDelta(150, got.Centroid.Mid, 0.001)
	s.Require().InDelta(150, got.Centroid.High, 0.001)
}

// TestNoRecordingsAtAll is an empty answer rather than a guess.
func (s *AcrossPublicTestSuite) TestNoRecordingsAtAll() {
	got := audio.Together(nil)

	s.Require().Equal(0, got.Tracks)
	s.Require().Equal(audio.Spread{}, got.Centroid)
	s.Require().Equal(audio.Spread{}, got.Harmonics)
}

// TestEveryMeasureIsGathered covers nothing being quietly dropped.
func (s *AcrossPublicTestSuite) TestEveryMeasureIsGathered() {
	got := audio.Together([]audio.Profile{
		{
			Low: 0.6, Mid: 0.3, High: 0.1,
			Centroid: 410, Transient: rang(0.8), Decay: rang(0.4),
			DynamicRange: 4.2,
			Harmonics:    audio.Spread{Mid: 0.18},
			EvenOdd:      audio.Spread{Mid: -0.6},
		},
	})

	for name, sp := range map[string]audio.Spread{
		"low": got.Low, "mid": got.Mid, "high": got.High,
		"centroid": got.Centroid, "dynamics": got.DynamicRange,
		"harmonics": got.Harmonics, "lean": got.EvenOdd,
	} {
		s.Require().NotZero(sp.Mid, "%s never reached the answer", name)
	}

	// The two a recording can decline to answer are gathered separately, and
	// carry how many of them did.
	for name, r := range map[string]audio.Ranged{
		"transient": got.Transient, "decay": got.Decay,
	} {
		s.Require().Equal(1, r.From, "%s did not count the record it came from", name)
		s.Require().NotZero(r.Mid, "%s never reached the answer", name)
	}
}

// TestARecordingThatDeclinesIsLeftOut covers a corpus where only some records
// answer.
//
// The middle has to come from the records that had the measurement. Counting
// the ones that did not as zero drags every answer down for a reason that has
// nothing to do with the playing.
func (s *AcrossPublicTestSuite) TestARecordingThatDeclinesIsLeftOut() {
	got := audio.Together([]audio.Profile{
		{Decay: rang(0.8)},
		{Decay: audio.Reading{}},
		{Decay: rang(1.0)},
	})

	s.Require().Equal(2, got.Decay.From, "two of the three had a decay")
	s.Require().InDelta(0.8, got.Decay.Low, 0.001)
	s.Require().InDelta(1.0, got.Decay.High, 0.001)
}

// TestNoRecordingAnswered covers a corpus where none could.
func (s *AcrossPublicTestSuite) TestNoRecordingAnswered() {
	got := audio.Together([]audio.Profile{{Centroid: 150}, {Centroid: 160}})

	s.Require().Equal(2, got.Tracks, "the records are still there")
	s.Require().Equal(0, got.Decay.From, "but none of them had a decay")
	s.Require().Equal(audio.Spread{}, got.Decay.Spread)
}

func TestAcrossPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(AcrossPublicTestSuite))
}
