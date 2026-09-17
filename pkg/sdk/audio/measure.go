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

import (
	"math"
	"slices"
)

// Spread is one measurement across a recording, kept as a range.
//
// Some of what a recording measures as is the same in every window and some
// of it is not. Harmonic content is the second kind: on four bass stems it
// ran from a few percent in the quietest tenth of windows to nearly half in
// the loudest, and reporting only the middle one hid the fact. Whether what
// separates two players is the middle or the width is not known yet, so both
// are kept.
type Spread struct {
	// Low is the tenth percentile: all but the lowest tenth of windows are
	// above this.
	Low float64
	// Mid is the median, the middle window.
	Mid float64
	// High is the ninetieth percentile: all but the highest tenth are below
	// it.
	High float64
}

// Reading is a measurement a recording may not have allowed.
//
// Some questions have no answer for some audio. A tone that starts at full
// level never rises, so there is no attack to measure; a note held through a
// whole bar never falls to a quarter of itself, so there is no decay. Both
// used to come back as a number anyway: zero for the first, and the length of
// the recording for the second.
//
// A number nobody could take is worse than no number, because it is
// indistinguishable from one somebody did. Known is what separates them.
type Reading struct {
	// Value is the measurement, when there was one.
	Value float64
	// Known says whether the recording answered at all. Value means nothing
	// when it is false.
	Known bool
}

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
	// Fingers are lower, a pick is higher, a slap higher still. Unknown when
	// the recording starts at its loudest, because nothing rose.
	Transient Reading

	// Decay is how long a note takes to fall to a quarter of its peak, in
	// seconds. Muted playing is short, a ringing open string is long. Unknown
	// when it never falls that far before the recording ends.
	Decay Reading

	// DynamicRange is the gap between the loudest and the typical moment, in
	// decibels. A heavily compressed part is a small number.
	DynamicRange float64

	// Harmonics is the share of energy above the fundamental, from zero to
	// one. A clean note is low, a distorted one is high.
	Harmonics Spread
	// EvenOdd leans positive when even harmonics dominate and negative when
	// odd ones do. Even is the warmth of a valve; odd is the edge of a fuzz.
	EvenOdd Spread
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

// startedAtPeak is how close to its loudest a recording's first frame has to
// be before there is no attack left in it to measure.
//
// Measured rather than chosen. Across the generators, everything already at
// level reads 0.92 to 1.00 of its peak in the first frame: a sine 0.92, noise
// 0.94, a struck note with no silence before it 0.99, a square 1.00. Anything
// with a start to find reads under 0.01: a swell 0.002, silence before a
// pluck 0.000. Nothing lands between 0.003 and 0.917, so the figure in the
// middle is not delicate.
const startedAtPeak = 0.5

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

	// One window per short slice of the recording rather than one over all of
	// it. A single transform of a three-minute track answers "what is the
	// average of everything at once", which is not a question anybody asked.
	windows := spectra(samples, rate)

	out.Low, out.Mid, out.High = bands(windows)
	out.Centroid = centroid(windows)
	out.Harmonics, out.EvenOdd = harmonics(windows)

	out.Transient = transient(samples, rate)
	out.Decay = decay(samples, rate)
	out.DynamicRange = dynamicRange(samples, rate)

	return out
}

// bands is the share of energy below lowMid, between the two, and above
// midHigh.
func bands(
	windows []spectrum,
) (float64, float64, float64) {
	var low, mid, high float64

	// Summed across every window, so a loud passage counts for more than a
	// quiet one. Averaging the per-window shares instead would let two
	// seconds of near-silence outvote a chorus.
	for _, sp := range windows {
		for i, m := range sp.mag {
			// Energy rather than magnitude: doubling a signal's amplitude
			// puts four times the energy in it, and shares that ignored that
			// would overstate everything quiet.
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
	}

	total := low + mid + high
	if total == 0 {
		return 0, 0, 0
	}

	return low / total, mid / total, high / total
}

// centroid is the energy-weighted average frequency.
func centroid(
	windows []spectrum,
) float64 {
	var weighted, total float64

	for _, sp := range windows {
		for i, m := range sp.mag {
			e := m * m
			weighted += sp.hz(i) * e
			total += e
		}
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
	windows []spectrum,
) (Spread, Spread) {
	shares := make([]float64, 0, len(windows))
	leans := make([]float64, 0, len(windows))

	for _, sp := range windows {
		share, lean, ok := harmonicsOf(sp)
		if !ok {
			continue
		}

		shares = append(shares, share)
		leans = append(leans, lean)
	}

	// A range rather than the average of them. One window catching a cymbal
	// or a key change should not move the answer, which is why neither end is
	// the largest or the smallest window.
	return spreadOf(shares), spreadOf(leans)
}

// harmonicsOf reads one window: how much energy sits above its loudest
// frequency, and whether the even multiples of it or the odd ones carry more.
//
// One window rather than a whole recording, because this takes the loudest
// bin as the fundamental and that is only true of a short slice. Over three
// minutes the loudest bin is whichever note was played most, and every other
// note is then measured as though it were a harmonic of that one. Measured on
// a real track, reading it that way swung between 38% and 61% depending on
// which ten seconds were looked at.
func harmonicsOf(
	sp spectrum,
) (float64, float64, bool) {
	peak, at := 0.0, 0

	for i, m := range sp.mag {
		if m > peak {
			peak, at = m, i
		}
	}

	if at == 0 || peak == 0 {
		return 0, 0, false
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

	return above / total, lean, true
}

// spreadOf is what the windows measured, as a range rather than a number.
//
// Nothing here is the largest or the smallest window. Both ends are a tenth
// of the way in, so a single window catching something the player did not do
// moves neither.
func spreadOf(
	of []float64,
) Spread {
	if len(of) == 0 {
		return Spread{}
	}

	sorted := slices.Clone(of)
	slices.Sort(sorted)

	return Spread{
		Low:  quantile(sorted, 0.1),
		Mid:  quantile(sorted, 0.5),
		High: quantile(sorted, 0.9),
	}
}

// quantile is the value that share of the way through a sorted list.
//
// share is never one, so the index it lands on is always inside the list:
// even at 0.9 of a single value the index is zero.
func quantile(
	sorted []float64,
	share float64,
) float64 {
	return sorted[int(share*float64(len(sorted)))]
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
) Reading {
	frame := rate / 200
	if frame < 1 || len(samples) < frame*4 {
		return Reading{}
	}

	var largest, peak float64

	prev := loudness(samples[:frame])
	first := prev
	peak = prev

	for at := frame; at+frame <= len(samples); at += frame {
		cur := loudness(samples[at : at+frame])

		largest = math.Max(largest, cur-prev)
		peak = math.Max(peak, cur)

		prev = cur
	}

	// Nothing to rise from. A recording already at its loudest in the first
	// frame has no attack in it to be sharp or gradual, and the largest rise
	// it does contain is whatever the level wandered by afterwards.
	//
	// The size of that rise cannot be what decides this. Measured across the
	// generators, a swell's largest rise is 2.9% of its peak and a held sine's
	// is 2.5%: near enough identical, while one is a real attack and the other
	// is not. Where the signal starts separates them cleanly.
	if peak == 0 || first >= peak*startedAtPeak {
		return Reading{}
	}

	return Reading{Value: math.Min(1, largest/peak), Known: true}
}

// onsetRise is how far a frame has to rise above the one before it, as a share
// of the recording's loudest, to count as a new note rather than the same one
// continuing.
//
// Swept from 0.02 to 0.40 across thirteen bass stems. The sweep did not find a
// best value, because the thing worth optimising turned out to be unobtainable
// at any threshold: see decay. What it did find is where the detector starts
// missing notes, above about 0.30, where one player's decay collapses from
// 0.26s to 0.05s as onsets stop being counted.
//
// 0.06 sits well below that, and the synthetic signals whose answers are known
// measure correctly there.
const onsetRise = 0.06

// decay is how long a note takes to fall to a quarter of itself, across the
// notes in a recording.
//
// A quarter rather than silence, because a recording's noise floor never
// reaches silence and waiting for it would measure the room.
//
// Per note, and this is the whole point. An earlier version took the single
// loudest frame in the recording and timed that one fall. On a dense line the
// loudest frame is one accent, and the level drops back to the ongoing playing
// immediately after it: nothing stopped ringing, the loudest moment merely
// passed. Measured that way one bassist read 0.01s to 0.06s across three
// records, and no note dies in thirty milliseconds.
//
// The figure it produced was not stable either. Measured across two different
// selections of the same player's records it moved 0.43s, which is as far
// apart as the artists are from each other, so it could not support a
// comparison between players at all.
//
// Per note fixes the first problem and not the second. A player whose notes
// plainly ring now reads 0.26s rather than 0.03s, which is a believable ring
// time. But measured across two selections of one player's records the figure
// still moves 0.28s to 0.46s, at every onset threshold from 0.02 to 0.40,
// against 0.56s to 0.77s between the artists themselves. There is no threshold
// where the drift becomes a small enough share of the separation to compare
// players by.
//
// So this is a measurement of a recording rather than of a player, and it
// stays out of Axes. Centroid moves 1Hz across the same two selections;
// that is what a figure has to do before a word is derived from it.
func decay(
	samples []float64,
	rate int,
) Reading {
	frame := rate / 100
	if frame < 1 || len(samples) < frame*2 {
		return Reading{}
	}

	levels := frames(samples, frame)
	held := playing(levels)

	var loudest float64
	for _, l := range levels {
		loudest = math.Max(loudest, l)
	}

	if loudest == 0 {
		return Reading{}
	}

	rang := make([]float64, 0, len(levels)/4)

	for i := range levels {
		if !held[i] {
			continue
		}

		// A note starts where the level rises into it, where the playing
		// resumes after a rest, or at the very beginning. The last two matter
		// as much as the first: a recording that opens with a note already
		// sounding has nothing to rise from, and looking only for a rise finds
		// no notes in it at all.
		if i > 0 && held[i-1] && levels[i]-levels[i-1] < loudest*onsetRise {
			continue
		}

		// This note's own peak, which is at the onset or just after it.
		peak := levels[i]
		for j := i + 1; j < len(levels) && held[j] && levels[j] > peak; j++ {
			peak = levels[j]
		}

		for j := i + 1; j < len(levels); j++ {
			// Either the note has fallen far enough, or the playing stopped
			// while it was still sounding. Both end this note.
			//
			// A rest is not a note dying away: reaching one means the player
			// stopped, and timing the silence after it would report the gap
			// rather than the ring. What can honestly be said then is that it
			// rang for at least this long.
			if levels[j] <= peak/4 || !held[j] {
				rang = append(rang,
					float64(j-i)*float64(frame)/float64(rate))

				break
			}
		}
	}

	// Nothing rose far enough to be a note, or no note ever fell. Returning
	// what is left of the recording would give back its length rather than a
	// decay, and a sustained note or a generated tone would read as ringing
	// for exactly as long as somebody happened to record.
	if len(rang) == 0 {
		return Reading{}
	}

	slices.Sort(rang)

	return Reading{Value: quantile(rang, 0.5), Known: true}
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

	sorted := slices.Clone(levels)
	slices.Sort(sorted)

	// Both are above zero: the loop above keeps only frames that are.
	middle := quantile(sorted, 0.5)
	loudest := sorted[len(sorted)-1]

	return 20 * math.Log10(loudest/middle)
}
