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

// EvidencePublicTestSuite covers naming a measurement for a rig.
type EvidencePublicTestSuite struct {
	suite.Suite
}

// keys is every name a measurement is written under.
func (s *EvidencePublicTestSuite) keys() []string {
	return audio.MeasuredKeys()
}

// TestEveryKeyIsMeasured covers the order naming exactly what is produced.
//
// A key in the order that nothing measures would write an absent figure into
// a rig; a key measured but missing from the order would be dropped silently.
func (s *EvidencePublicTestSuite) TestEveryKeyIsMeasured() {
	got := audio.Profile{}.Measured()

	s.Require().Len(audio.MeasuredKeys(), len(got))

	for _, key := range audio.MeasuredKeys() {
		s.Require().Contains(got, key)
	}
}

// TestTheOrderIsStable covers evidence written twice reading the same way.
func (s *EvidencePublicTestSuite) TestTheOrderIsStable() {
	s.Require().Equal(audio.MeasuredKeys(), audio.MeasuredKeys())
}

// TestOneRecordingIsNamedInFull covers a profile writing every key.
func (s *EvidencePublicTestSuite) TestOneRecordingIsNamedInFull() {
	got := audio.Profile{
		Low: 0.9134, Mid: 0.0812, High: 0.0054,
		Centroid: 175.4, Transient: 0.7431, Decay: 0.8249,
		DynamicRange: 7.84,
		Harmonics:    audio.Spread{Mid: 0.3512},
		EvenOdd:      audio.Spread{Mid: -0.6049},
	}.Measured()

	for _, key := range s.keys() {
		s.Require().Contains(got, key)
	}

	s.Require().Len(got, len(s.keys()), "and nothing nobody measures")
}

// TestGatheredRecordingsAreNamedTheSameWay covers the two agreeing.
//
// A rig cannot compare a figure from one recording against a figure from four
// unless both are written under the same name.
func (s *EvidencePublicTestSuite) TestGatheredRecordingsAreNamedTheSameWay() {
	one := audio.Profile{Centroid: 175}.Measured()
	many := audio.Together([]audio.Profile{{Centroid: 175}}).Measured()

	s.Require().Equal(len(one), len(many))

	for key := range one {
		s.Require().Contains(many, key, "%s is named by one and not the other", key)
	}
}

// TestFiguresAreRoundedToWhatTheyCanClaim covers the numbers being readable.
//
// The bins are about 10Hz apart, so a centroid to six places claims a
// precision the transform does not have.
func (s *EvidencePublicTestSuite) TestFiguresAreRoundedToWhatTheyCanClaim() {
	got := audio.Profile{
		Low: 0.913471, Centroid: 175.4382, Decay: 0.824913, DynamicRange: 7.8449,
	}.Measured()

	s.Require().InDelta(0.91, got[audio.KeyLow], 1e-9)
	s.Require().InDelta(175, got[audio.KeyCentroid], 1e-9)
	s.Require().InDelta(0.82, got[audio.KeyDecay], 1e-9)
	s.Require().InDelta(7.8, got[audio.KeyDynamics], 1e-9)
}

// TestTheMiddleIsWhatIsCarried covers a gathered measurement writing the
// middle rather than an end.
func (s *EvidencePublicTestSuite) TestTheMiddleIsWhatIsCarried() {
	got := audio.Together([]audio.Profile{
		{Centroid: 135}, {Centroid: 175}, {Centroid: 195},
	}).Measured()

	s.Require().InDelta(175, got[audio.KeyCentroid], 1e-9)
}

// TestNothingMeasuredIsStillNamed covers an empty measurement.
//
// Zeroes rather than absent keys. A rig reading this can tell the difference
// between a figure of zero and a figure nobody took, and dropping the key
// would remove that.
func (s *EvidencePublicTestSuite) TestNothingMeasuredIsStillNamed() {
	got := audio.Across{}.Measured()

	s.Require().Len(got, len(s.keys()))
}

func TestEvidencePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(EvidencePublicTestSuite))
}
