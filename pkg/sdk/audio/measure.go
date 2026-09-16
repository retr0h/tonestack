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

// Package audio measures what a recording sounds like, as numbers.
//
// Not as words. A rig says "warm" or "percussive" and two people mean
// different things by either; a spectral centroid of 410Hz means one thing.
// The point of this package is to put a number where an adjective was, so a
// preset can be compared to a recording rather than to somebody's memory of
// one.
//
// Nothing here decides what a measurement means. Turning 410Hz into "warm" is
// a judgement, it belongs where judgements are made, and mixing the two would
// leave nowhere to stand when they disagree.
package audio

import "math"

// Profile is what a recording measures as.
//
// Every field is a number somebody else could compute from the same audio and
// get the same answer. Nothing here is an opinion.
type Profile struct {
	// Seconds is how long the measured audio runs.
	Seconds float64
	// Rate is the sample rate it was measured at.
	Rate int

	// Low, Mid and High are the share of energy in each band, summing to one.
	//
	// For a bass the interesting one is Low: where the fundamental sits. Mid
	// carries the note's body and the growl of any distortion, and High is
	// string noise, pick attack and air.
	Low, Mid, High float64

	// Centroid is the centre of gravity of the spectrum, in hertz.
	//
	// Where the sound sits. A dub bass and a Rickenbacker played with a pick
	// can hold the same note and land hundreds of hertz apart here.
	Centroid float64

	// Transient is how sharply notes start, from zero to one.
	//
	// The share of the signal's rises that are sudden rather than gradual.
	// Fingers are lower, a pick is higher, a slap higher still.
	Transient float64

	// Decay is how long a note takes to fall to a quarter of its peak, in
	// seconds. Muted playing is short, a ringing open string is long.
	Decay float64

	// DynamicRange is the gap between the loudest and the typical moment, in
	// decibels. A heavily compressed part is a small number.
	DynamicRange float64

	// Harmonics is the share of energy above the fundamental, from zero to
	// one. A clean note is low, a distorted one is high.
	Harmonics float64
	// EvenOdd leans positive when even harmonics dominate and negative when
	// odd ones do. Even is the warmth of a valve; odd is the edge of a fuzz.
	EvenOdd float64
}

// Band edges, in hertz.
//
// Chosen for a bass guitar rather than for a full mix. A four-string's open E
// is about 41Hz and its highest fretted note is under 400Hz, so lowMid sits
// above the fundamentals and below the body, and midHigh above anything the
// instrument sounds without a pick or a fret buzzing.
const (
	lowMid  = 250.0
	midHigh = 2000.0
)

// Measure reads a recording into the numbers that describe it.
//
// Samples are single-channel, between -1 and 1, at rate samples a second.
// Anything shorter than one window measures as an empty Profile rather than
// as a guess.
func Measure(
	samples []float64,
	rate int,
) Profile {
	out := Profile{Rate: rate}
	if len(samples) == 0 || rate <= 0 {
		return out
	}

	out.Seconds = float64(len(samples)) / float64(rate)

	sp := analyse(samples, rate)
	out.Low, out.Mid, out.High = bands(sp)
	out.Centroid = centroid(sp)
	out.Harmonics, out.EvenOdd = harmonics(sp)

	out.Transient = transient(samples, rate)
	out.Decay = decay(samples, rate)
	out.DynamicRange = dynamicRange(samples, rate)

	return out
}

// bands is the share of energy below lowMid, between the two, and above
// midHigh.
func bands(
	sp spectrum,
) (float64, float64, float64) {
	var low, mid, high float64

	for i, m := range sp.mag {
		// Energy rather than magnitude: doubling a signal's amplitude puts
		// four times the energy in it, and shares that ignored that would
		// overstate everything quiet.
		e := m * m

		switch hz := sp.hz(i); {
		case hz < lowMid:
			low += e
		case hz < midHigh:
			mid += e
		default:
			high += e
		}
	}

	total := low + mid + high
	if total == 0 {
		return 0, 0, 0
	}

	return low / total, mid / total, high / total
}

// centroid is the energy-weighted average frequency.
func centroid(
	sp spectrum,
) float64 {
	var weighted, total float64

	for i, m := range sp.mag {
		e := m * m
		weighted += sp.hz(i) * e
		total += e
	}

	if total == 0 {
		return 0
	}

	return weighted / total
}

// harmonics is how much energy sits above the loudest frequency, and whether
// the even multiples of it or the odd ones carry more.
//
// The loudest bin is taken as the fundamental. True of a bass note played
// alone, which is what this measures; a chord has no single fundamental and
// the answer is meaningless rather than wrong.
func harmonics(
	sp spectrum,
) (float64, float64) {
	peak, at := 0.0, 0

	for i, m := range sp.mag {
		if m > peak {
			peak, at = m, i
		}
	}

	if at == 0 || peak == 0 {
		return 0, 0
	}

	var fundamental, above, even, odd float64

	for i, m := range sp.mag {
		e := m * m

		switch {
		case i < at+at/2:
			fundamental += e
		default:
			above += e
		}

		// Which multiple of the fundamental this bin is nearest, and how far
		// off it is. Anything not close to a whole multiple is neither even
		// nor odd; it is noise between the harmonics.
		mult := float64(i) / float64(at)
		nearest := math.Round(mult)

		if nearest >= 2 && math.Abs(mult-nearest) < 0.1 {
			if int(nearest)%2 == 0 {
				even += e
			} else {
				odd += e
			}
		}
	}

	// No total to divide by is impossible here: reaching this point needs a
	// peak above zero, and a peak above zero is energy.
	total := fundamental + above

	lean := 0.0
	if even+odd > 0 {
		lean = (even - odd) / (even + odd)
	}

	return above / total, lean
}

// transient is how much of a note's level arrives in one step.
//
// The largest rise between neighbouring frames, against the loudest the
// signal gets. A struck string goes from nothing to everything in a single
// frame and reads near one; a note swelled in, or a tone simply held, climbs
// in many small steps and reads near zero.
//
// An earlier version measured how much the rises varied among themselves,
// which is a different thing and got the answer backwards: a cleanly struck
// note has exactly one rise, so there was no variation and it scored zero.
// Measured across a struck note, a swell, a held tone, a square and noise,
// the two struck cases land above 0.94 and everything gradual below 0.1.
func transient(
	samples []float64,
	rate int,
) float64 {
	frame := rate / 200
	if frame < 1 || len(samples) < frame*4 {
		return 0
	}

	var largest, peak float64

	prev := loudness(samples[:frame])
	peak = prev

	for at := frame; at+frame <= len(samples); at += frame {
		cur := loudness(samples[at : at+frame])

		largest = math.Max(largest, cur-prev)
		peak = math.Max(peak, cur)

		prev = cur
	}

	if peak == 0 {
		return 0
	}

	return math.Min(1, largest/peak)
}

// decay is how long the loudest moment takes to fall to a quarter of itself.
//
// A quarter rather than silence, because a recording's noise floor never
// reaches silence and waiting for it would measure the room.
func decay(
	samples []float64,
	rate int,
) float64 {
	frame := rate / 100
	if frame < 1 || len(samples) < frame*2 {
		return 0
	}

	levels := frames(samples, frame)
	held := playing(levels)

	peak, at := 0.0, 0

	for i, l := range levels {
		if l > peak {
			peak, at = l, i
		}
	}

	if peak == 0 {
		return 0
	}

	for i := at; i < len(levels); i++ {
		// A rest is not a note dying away. Reaching one means the player
		// stopped, and timing that would report the gap rather than the
		// ring, which is what a track full of rests did before this.
		if !held[i] {
			break
		}

		if levels[i] <= peak/4 {
			return float64(i-at) * float64(frame) / float64(rate)
		}
	}

	// It was still sounding when the playing stopped, so all that can be said
	// is that it rang for at least this long.
	for i := at; i < len(levels); i++ {
		if !held[i] {
			return float64(i-at) * float64(frame) / float64(rate)
		}
	}

	return float64(len(levels)-at) * float64(frame) / float64(rate)
}

// dynamicRange is the gap between the loudest frame and the median one, in
// decibels.
//
// The median rather than the average, so a part that is mostly silence with
// occasional notes is not reported as wildly dynamic because of the gaps.
func dynamicRange(
	samples []float64,
	rate int,
) float64 {
	frame := rate / 50
	if frame < 1 || len(samples) < frame*2 {
		return 0
	}

	all := frames(samples, frame)
	held := playing(all)

	// Only where the instrument is sounding. The median of a part that rests
	// half the time lands in the noise floor, and the gap between the loudest
	// note and the noise floor is not how dynamic the playing is: one stem
	// measured 55dB that way, against 6 to 8dB for the same player elsewhere.
	levels := make([]float64, 0, len(all))

	for i, l := range all {
		if held[i] && l > 0 {
			levels = append(levels, l)
		}
	}

	if len(levels) == 0 {
		return 0
	}

	sorted := append([]float64(nil), levels...)
	for i := 1; i < len(sorted); i++ {
		for j := i; j > 0 && sorted[j] < sorted[j-1]; j-- {
			sorted[j], sorted[j-1] = sorted[j-1], sorted[j]
		}
	}

	// Both are above zero: the loop above keeps only frames that are.
	median := sorted[len(sorted)/2]
	loudest := sorted[len(sorted)-1]

	return 20 * math.Log10(loudest/median)
}
