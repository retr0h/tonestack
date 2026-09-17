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

// MeasurePublicTestSuite covers the numbers a recording reads as.
//
// Every case here is a signal whose answer can be stated before it is
// measured. These definitions are this project's own rather than a known
// algorithm, so they are worth less than the transform underneath them until
// something predictable has been pointed at them.
type MeasurePublicTestSuite struct {
	suite.Suite
}

// TestASineIsAllFundamental covers the simplest thing there is.
//
// One frequency, so its energy is in whichever band holds it, its centroid is
// that frequency, and there is nothing above it.
func (s *MeasurePublicTestSuite) TestASineIsAllFundamental() {
	got := audio.Measure(audio.Sine(100, 1.0, rate, 0.8), rate)

	s.Require().InDelta(1.0, got.Seconds, 0.001)
	s.Require().Equal(rate, got.Rate)

	s.Require().Greater(got.Low, 0.99, "a 100Hz tone is entirely low")
	s.Require().Less(got.Mid, 0.01)
	s.Require().Less(got.High, 0.01)

	s.Require().InDelta(100, got.Centroid, 15, "and sits where it sounds")
	s.Require().Less(got.Harmonics.Mid, 0.05, "with nothing above it")
}

// TestBandsSumToOne is what makes the three shares readable.
func (s *MeasurePublicTestSuite) TestBandsSumToOne() {
	for _, hz := range []float64{80, 500, 5000} {
		s.Run("", func() {
			got := audio.Measure(audio.Sine(hz, 0.5, rate, 0.7), rate)

			s.Require().InDelta(1.0, got.Low+got.Mid+got.High, 0.001)
		})
	}
}

// TestEachBandCatchesItsOwn covers the edges being where they are said to be.
func (s *MeasurePublicTestSuite) TestEachBandCatchesItsOwn() {
	low := audio.Measure(audio.Sine(80, 0.5, rate, 0.7), rate)
	mid := audio.Measure(audio.Sine(800, 0.5, rate, 0.7), rate)
	high := audio.Measure(audio.Sine(5000, 0.5, rate, 0.7), rate)

	s.Require().Greater(low.Low, 0.95)
	s.Require().Greater(mid.Mid, 0.95)
	s.Require().Greater(high.High, 0.95)
}

// TestAHigherNoteSitsHigher covers the centroid tracking the sound.
func (s *MeasurePublicTestSuite) TestAHigherNoteSitsHigher() {
	deep := audio.Measure(audio.Sine(60, 0.5, rate, 0.7), rate)
	bright := audio.Measure(audio.Sine(900, 0.5, rate, 0.7), rate)

	s.Require().Greater(bright.Centroid, deep.Centroid*5)
}

// TestASquareIsOddHarmonics covers distortion's own signature.
//
// A square holds every odd multiple of its fundamental and no even ones, so
// it should read as harmonically rich and leaning odd. That lean is the
// difference between a valve's warmth and a fuzz's edge.
func (s *MeasurePublicTestSuite) TestASquareIsOddHarmonics() {
	clean := audio.Measure(audio.Sine(200, 0.5, rate, 0.5), rate)
	dirty := audio.Measure(audio.Square(200, 0.5, rate, 0.5), rate)

	s.Require().Greater(dirty.Harmonics.Mid, clean.Harmonics.Mid*3,
		"a square carries far more above its fundamental than a sine")
	s.Require().Negative(dirty.EvenOdd.Mid, "and what it carries is odd")
}

// TestASpreadIsOrdered covers the three numbers being a range.
func (s *MeasurePublicTestSuite) TestASpreadIsOrdered() {
	got := audio.Measure(audio.Plucked(110, 2.0, rate, 0.9, 4), rate)

	s.Require().LessOrEqual(got.Harmonics.Low, got.Harmonics.Mid)
	s.Require().LessOrEqual(got.Harmonics.Mid, got.Harmonics.High)

	s.Require().LessOrEqual(got.EvenOdd.Low, got.EvenOdd.Mid)
	s.Require().LessOrEqual(got.EvenOdd.Mid, got.EvenOdd.High)
}

// TestANoteThatChangesSpreadsWider is the reason the range is kept.
//
// A tone that never changes measures the same in every window, so its three
// numbers sit almost on top of each other. A note that is struck and then
// dies away does not, and a median alone would report the two as the same
// kind of measurement.
func (s *MeasurePublicTestSuite) TestANoteThatChangesSpreadsWider() {
	steady := audio.Measure(audio.Sine(110, 2.0, rate, 0.8), rate)
	plucked := audio.Measure(audio.Plucked(110, 2.0, rate, 0.9, 4), rate)

	s.Require().Greater(
		plucked.Harmonics.High-plucked.Harmonics.Low,
		steady.Harmonics.High-steady.Harmonics.Low,
	)
}

// TestAPluckedNoteDecaysFasterThanAToneHeld covers decay separating them.
func (s *MeasurePublicTestSuite) TestAPluckedNoteDecaysFasterThanAToneHeld() {
	held := audio.Measure(audio.Sine(110, 2.0, rate, 0.8), rate)
	plucked := audio.Measure(audio.Plucked(110, 2.0, rate, 0.9, 4), rate)

	s.Require().Less(plucked.Decay, held.Decay,
		"a note that dies away decays sooner than one that does not")
	s.Require().Less(plucked.Decay, 1.0, "and does it within the recording")
}

// TestAHarderPluckDecaysSooner covers decay tracking the envelope.
func (s *MeasurePublicTestSuite) TestAHarderPluckDecaysSooner() {
	slow := audio.Measure(audio.Plucked(110, 2.0, rate, 0.8, 2), rate)
	fast := audio.Measure(audio.Plucked(110, 2.0, rate, 0.8, 8), rate)

	s.Require().Less(fast.Decay, slow.Decay)
}

// TestASteadyToneIsNotDynamic covers dynamic range on something unchanging.
func (s *MeasurePublicTestSuite) TestASteadyToneIsNotDynamic() {
	got := audio.Measure(audio.Sine(220, 1.0, rate, 0.7), rate)

	s.Require().Less(got.DynamicRange, 1.0,
		"a tone that never changes has almost no range")
}

// TestAPluckedNoteIsDynamic covers the other end of the same measurement.
func (s *MeasurePublicTestSuite) TestAPluckedNoteIsDynamic() {
	steady := audio.Measure(audio.Sine(110, 2.0, rate, 0.8), rate)
	plucked := audio.Measure(audio.Plucked(110, 2.0, rate, 0.9, 3), rate)

	s.Require().Greater(plucked.DynamicRange, steady.DynamicRange+5,
		"a note that starts loud and dies away covers real ground")
}

// TestASuddenStartReadsAsTransient covers what separates a pick from a swell.
//
// The ordering here was measured before it was asserted. Across a struck
// note, a swelled one, a held tone, a square and noise, the struck cases land
// above 0.94 and everything gradual below 0.1. An earlier definition measured
// how much the rises varied among themselves and put a cleanly struck note at
// zero, because one rise has nothing to vary against.
func (s *MeasurePublicTestSuite) TestASuddenStartReadsAsTransient() {
	// Silence, then a note starting at once: the sharpest start there is.
	struck := audio.Measure(
		append(audio.Silence(0.3, rate), audio.Plucked(110, 0.7, rate, 0.9, 6)...), rate)
	ringing := audio.Measure(
		append(audio.Silence(0.3, rate), audio.Plucked(110, 0.7, rate, 0.9, 1)...), rate)

	held := audio.Measure(audio.Sine(110, 1.0, rate, 0.7), rate)
	swelled := audio.Measure(audio.Swell(110, 1.0, rate, 0.8), rate)

	s.Require().Greater(struck.Transient, 0.9, "a struck note arrives at once")
	s.Require().Greater(ringing.Transient, 0.9, "however long it then rings")

	s.Require().Less(held.Transient, 0.1, "a held tone never arrives")
	s.Require().Less(swelled.Transient, 0.1, "and a swell arrives gradually")

	s.Require().Greater(struck.Transient, swelled.Transient*5,
		"the two are not close")
}

// TestTransientIgnoresHowLoud covers it measuring the start, not the level.
func (s *MeasurePublicTestSuite) TestTransientIgnoresHowLoud() {
	quiet := audio.Measure(
		append(audio.Silence(0.3, rate), audio.Plucked(110, 0.7, rate, 0.2, 6)...), rate)
	loud := audio.Measure(
		append(audio.Silence(0.3, rate), audio.Plucked(110, 0.7, rate, 0.9, 6)...), rate)

	s.Require().InDelta(quiet.Transient, loud.Transient, 0.05)
}

// TestTransientNeverExceedsOne covers the one case that could push it past.
//
// A signal that starts at full level in its very first frame has a rise as
// large as its peak, and nothing should report more than all of it.
func (s *MeasurePublicTestSuite) TestTransientNeverExceedsOne() {
	got := audio.Measure(audio.Plucked(110, 1.0, rate, 0.9, 8), rate)

	s.Require().LessOrEqual(got.Transient, 1.0)
	s.Require().GreaterOrEqual(got.Transient, 0.0)
}

// TestSilenceMeasuresAsNothing covers every measurement surviving no signal.
func (s *MeasurePublicTestSuite) TestSilenceMeasuresAsNothing() {
	got := audio.Measure(audio.Silence(0.5, rate), rate)

	s.Require().InDelta(0.5, got.Seconds, 0.001, "it still has a length")

	s.Require().InDelta(0, got.Low, 1e-9)
	s.Require().InDelta(0, got.Mid, 1e-9)
	s.Require().InDelta(0, got.High, 1e-9)
	s.Require().InDelta(0, got.Centroid, 1e-9)
	s.Require().Equal(audio.Spread{}, got.Harmonics)
	s.Require().Equal(audio.Spread{}, got.EvenOdd)
	s.Require().InDelta(0, got.Decay, 1e-9)
	s.Require().InDelta(0, got.DynamicRange, 1e-9)
	s.Require().InDelta(0, got.Transient, 1e-9)
}

// TestNoSamplesAtAll measures as an empty profile rather than a guess.
func (s *MeasurePublicTestSuite) TestNoSamplesAtAll() {
	got := audio.Measure(nil, rate)

	s.Require().InDelta(0, got.Seconds, 1e-9)
	s.Require().Equal(rate, got.Rate)
	s.Require().InDelta(0, got.Centroid, 1e-9)
}

// TestARateOfNothing is a caller's mistake, not a division by zero.
func (s *MeasurePublicTestSuite) TestARateOfNothing() {
	got := audio.Measure(audio.Sine(100, 0.5, rate, 0.5), 0)

	s.Require().InDelta(0, got.Seconds, 1e-9)
	s.Require().InDelta(0, got.Centroid, 1e-9)
}

// TestTooShortToMeasure covers audio below one frame of anything.
func (s *MeasurePublicTestSuite) TestTooShortToMeasure() {
	got := audio.Measure([]float64{0.1, -0.1, 0.2}, rate)

	s.Require().InDelta(0, got.Transient, 1e-9)
	s.Require().InDelta(0, got.Decay, 1e-9)
	s.Require().InDelta(0, got.DynamicRange, 1e-9)
}

// TestNoiseIsSpreadAcrossEverything covers a spectrum with no shape.
//
// The opposite of a sine, and the case that catches a band measurement which
// quietly puts everything in one place.
func (s *MeasurePublicTestSuite) TestNoiseIsSpreadAcrossEverything() {
	got := audio.Measure(audio.Noise(1.0, rate, 0.5, 7), rate)

	s.Require().Greater(got.High, 0.5, "most of the spectrum is above 2kHz")
	s.Require().Greater(got.Centroid, 3000.0, "and its centre is high up")
}

// TestLoudnessDoesNotMoveTheShape covers the shares being shares.
//
// The same signal twice as loud is the same sound. Anything here that changes
// with amplitude alone is measuring the recording level, not the playing.
func (s *MeasurePublicTestSuite) TestLoudnessDoesNotMoveTheShape() {
	quiet := audio.Measure(audio.Square(150, 1.0, rate, 0.2), rate)
	loud := audio.Measure(audio.Square(150, 1.0, rate, 0.8), rate)

	s.Require().InDelta(quiet.Low, loud.Low, 0.01)
	s.Require().InDelta(quiet.Mid, loud.Mid, 0.01)
	s.Require().InDelta(quiet.Centroid, loud.Centroid, 1.0)
	s.Require().InDelta(quiet.Harmonics.Mid, loud.Harmonics.Mid, 0.01)
}

func TestMeasurePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(MeasurePublicTestSuite))
}
