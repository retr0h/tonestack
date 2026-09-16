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
	"math"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/audio"
)

// FFTPublicTestSuite covers the transform every measurement rests on.
//
// Checked against signals whose answers are known before they are measured. A
// transform that puts a 100Hz sine anywhere but 100Hz is wrong about a bass
// guitar too, and this is where that is found out.
type FFTPublicTestSuite struct {
	suite.Suite
}

// rate is a sample rate this project will really see.
const rate = 44100

// peak is the bin holding the most energy.
func (s *FFTPublicTestSuite) peak(
	sp audio.Spectrum,
) int {
	best, at := 0.0, 0

	for i, m := range sp.Mag() {
		if m > best {
			best, at = m, i
		}
	}

	return at
}

// TestASineSitsAtItsOwnFrequency is the measurement's foundation.
func (s *FFTPublicTestSuite) TestASineSitsAtItsOwnFrequency() {
	for _, hz := range []float64{100, 440, 1000} {
		s.Run("", func() {
			sp := audio.Analyse(audio.Sine(hz, 0.5, rate, 0.8), rate)

			got := sp.Hz(s.peak(sp))

			// Within one bin. A window of this length puts the bins a few
			// hertz apart, and a peak cannot land closer than that.
			s.Require().InDelta(hz, got, sp.HzPerBin()*1.5,
				"a %gHz sine should peak at %gHz, got %g", hz, hz, got)
		})
	}
}

// TestASineHoldsNothingElse covers the rest of the spectrum being empty.
//
// A transform can put the peak in the right place and still smear energy
// everywhere, which would make every band measurement meaningless.
func (s *FFTPublicTestSuite) TestASineHoldsNothingElse() {
	sp := audio.Analyse(audio.Sine(1000, 0.5, rate, 0.8), rate)
	at := s.peak(sp)

	var elsewhere float64

	for i, m := range sp.Mag() {
		// Skip the peak and its immediate neighbours: a window always spreads
		// a peak across the bins beside it.
		if i > at-4 && i < at+4 {
			continue
		}

		elsewhere = math.Max(elsewhere, m)
	}

	s.Require().Less(elsewhere, sp.Mag()[at]*0.02,
		"everything away from the peak should be near nothing")
}

// TestASquareHoldsOddHarmonics covers a signal with known overtones.
//
// A square is its own frequency plus every odd multiple, each quieter than the
// last in proportion. Getting this right is what makes a harmonics measurement
// mean anything.
func (s *FFTPublicTestSuite) TestASquareHoldsOddHarmonics() {
	const f = 500

	sp := audio.Analyse(audio.Square(f, 0.5, rate, 0.5), rate)

	at := func(hz float64) float64 {
		bin := int(math.Round(hz / sp.HzPerBin()))
		best := 0.0

		for i := bin - 2; i <= bin+2; i++ {
			if i >= 0 && i < len(sp.Mag()) {
				best = math.Max(best, sp.Mag()[i])
			}
		}

		return best
	}

	first, third, fifth := at(f), at(f*3), at(f*5)
	second, fourth := at(f*2), at(f*4)

	s.Require().InDelta(first/3, third, first*0.1, "the third is a third as loud")
	s.Require().InDelta(first/5, fifth, first*0.1, "the fifth is a fifth as loud")

	s.Require().Less(second, first*0.05, "a square holds no even harmonics")
	s.Require().Less(fourth, first*0.05, "nor a fourth")
}

// TestSilence measures as nothing rather than failing.
func (s *FFTPublicTestSuite) TestSilence() {
	sp := audio.Analyse(audio.Silence(0.1, rate), rate)

	for _, m := range sp.Mag() {
		s.Require().InDelta(0, m, 1e-9)
	}

	s.Require().InDelta(0, audio.Loudness(audio.Silence(0.1, rate)), 1e-9)
}

// TestNothingAtAll is a window with no samples in it.
func (s *FFTPublicTestSuite) TestNothingAtAll() {
	sp := audio.Analyse(nil, rate)

	s.Require().Empty(sp.Mag())
	s.Require().InDelta(0, sp.HzPerBin(), 1e-9, "and no division by zero")
	s.Require().InDelta(0, sp.Hz(3), 1e-9)
}

// TestOneSample holds no frequencies, which is the honest answer.
//
// A window of one pads to a transform of one, and half of that is nothing. A
// single sample has a level but no frequency content: how fast something
// changes cannot be read from a signal that never gets a second reading.
func (s *FFTPublicTestSuite) TestOneSample() {
	sp := audio.Analyse([]float64{0.5}, rate)

	s.Require().Empty(sp.Mag())
	s.Require().InDelta(0, sp.HzPerBin(), 1e-9, "and no division by zero")
}

// TestTwoSamples is the shortest window that reports anything.
func (s *FFTPublicTestSuite) TestTwoSamples() {
	sp := audio.Analyse([]float64{0.5, -0.5}, rate)

	s.Require().Len(sp.Mag(), 1)
	s.Require().InDelta(float64(rate)/2, sp.HzPerBin(), 1e-9)
}

// TestLouderInLouderOut covers amplitude reaching the spectrum.
func (s *FFTPublicTestSuite) TestLouderInLouderOut() {
	quiet := audio.Analyse(audio.Sine(440, 0.5, rate, 0.2), rate)
	loud := audio.Analyse(audio.Sine(440, 0.5, rate, 0.8), rate)

	s.Require().Greater(loud.Mag()[s.peak(loud)], quiet.Mag()[s.peak(quiet)]*3,
		"four times the amplitude is far more energy")
}

// TestLoudness covers the level a run of samples carries.
func (s *FFTPublicTestSuite) TestLoudness() {
	// A sine's root mean square is its amplitude over the square root of two.
	got := audio.Loudness(audio.Sine(440, 0.5, rate, 1.0))

	s.Require().InDelta(1/math.Sqrt2, got, 0.01)
	s.Require().InDelta(0, audio.Loudness(nil), 1e-9)
}

// TestAPluckedNoteDecays is what separates a string from a tone.
func (s *FFTPublicTestSuite) TestAPluckedNoteDecays() {
	got := audio.Plucked(110, 1.0, rate, 0.9, 4)

	head := audio.Loudness(got[:rate/10])
	tail := audio.Loudness(got[len(got)-rate/10:])

	s.Require().Greater(head, tail*5, "a plucked note dies away")
}

// TestTransformIsItsOwnCheck covers the arithmetic without a signal.
//
// A transform of a single spike holds the same magnitude in every bin. It is
// the one case whose answer can be written down without trusting anything
// else in this file.
func (s *FFTPublicTestSuite) TestTransformIsItsOwnCheck() {
	re := make([]float64, 16)
	im := make([]float64, 16)
	re[0] = 1

	audio.Transform(re, im)

	for i := range re {
		s.Require().InDelta(1, math.Hypot(re[i], im[i]), 1e-9,
			"bin %d", i)
	}
}

// TestTransformOfNothing returns without touching anything.
func (s *FFTPublicTestSuite) TestTransformOfNothing() {
	re, im := []float64{}, []float64{}

	audio.Transform(re, im)

	s.Require().Empty(re)
}

func TestFFTPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(FFTPublicTestSuite))
}
