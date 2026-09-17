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

package audio

import (
	"math"
	"math/rand/v2"
)

// Signals whose answers are known before they are measured.
//
// A measurement is only worth pointing at music once it has been pointed at
// something predictable. A sine at 100Hz has its energy at 100Hz and nowhere
// else, silence has no loudness, and a plucked note decays. Anything that gets
// those wrong is wrong about a bass guitar too, and this is where that is
// found out rather than argued about.

// sine is one frequency at one level, and nothing else.
func sine(
	hz float64,
	seconds float64,
	rate int,
	amplitude float64,
) []float64 {
	out := make([]float64, int(seconds*float64(rate)))
	for i := range out {
		out[i] = amplitude * math.Sin(2*math.Pi*hz*float64(i)/float64(rate))
	}

	return out
}

// silence is nothing at all, which several measurements have to survive.
func silence(
	seconds float64,
	rate int,
) []float64 {
	return make([]float64, int(seconds*float64(rate)))
}

// square is one frequency and every odd harmonic above it.
//
// Built by adding those harmonics rather than by switching between two values,
// so the harmonics it holds are known exactly instead of reaching up forever
// and folding back down as something that was never played.
func square(
	hz float64,
	seconds float64,
	rate int,
	amplitude float64,
) []float64 {
	out := make([]float64, int(seconds*float64(rate)))
	limit := float64(rate) / 2

	for h := 1; float64(h)*hz < limit; h += 2 {
		level := amplitude * (4 / math.Pi) / float64(h)
		for i := range out {
			out[i] += level * math.Sin(2*math.Pi*float64(h)*hz*float64(i)/float64(rate))
		}
	}

	return out
}

// noise is every frequency at once, which is what a spectrum with no shape
// looks like.
//
// Seeded fixed, because a measurement that changes between two runs of the
// same test teaches nothing about the measurement.
func noise(
	seconds float64,
	rate int,
	amplitude float64,
	seed uint64,
) []float64 {
	r := rand.New(rand.NewPCG(seed, seed))
	out := make([]float64, int(seconds*float64(rate)))

	for i := range out {
		out[i] = amplitude * (r.Float64()*2 - 1)
	}

	return out
}

// plucked is a note that starts loud and dies away, which is what a string
// does and what a sine never does.
//
// A single frequency under a falling envelope. Crude next to a real string,
// and enough to tell a measurement that reads decay from one that does not.
func plucked(
	hz float64,
	seconds float64,
	rate int,
	amplitude float64,
	decay float64,
) []float64 {
	out := make([]float64, int(seconds*float64(rate)))

	for i := range out {
		t := float64(i) / float64(rate)
		out[i] = amplitude * math.Exp(-decay*t) *
			math.Sin(2*math.Pi*hz*t)
	}

	return out
}

// swell is a note faded in rather than struck, the opposite of a pluck.
//
// The case that keeps a transient measurement honest: it reaches the same
// level as a struck note and takes the whole recording to get there.
func swell(
	hz float64,
	seconds float64,
	rate int,
	amplitude float64,
) []float64 {
	out := sine(hz, seconds, rate, amplitude)
	for i := range out {
		out[i] *= float64(i) / float64(len(out))
	}

	return out
}

// quietest is how far below a recording's own loudest moment a frame may sit
// and still count as playing, in decibels.
//
// Measured rather than chosen. Four bass stems from one band: three carry
// rests over 4% to 23% of their length, and one is silent for 51% of its
// running time across 56 separate gaps. In every one of them the silent
// stretches sit more than 40dB under the loudest note, and the playing never
// does.
//
// Relative to the recording rather than absolute, because a quiet master and
// a loud one are the same performance.
const quietest = -40.0

// playing reports which frames hold the instrument rather than the gap
// between notes.
//
// Anything measured over time has to ask this first. A rest is not quiet
// playing, and counting it as though it were reports the silence between the
// notes instead of the notes.
func playing(
	levels []float64,
) []bool {
	out := make([]bool, len(levels))

	var loudest float64
	for _, l := range levels {
		loudest = math.Max(loudest, l)
	}

	if loudest == 0 {
		return out
	}

	// Decibels are a ratio, so the floor is a fraction of the loudest frame.
	floor := loudest * math.Pow(10, quietest/20)

	for i, l := range levels {
		out[i] = l >= floor
	}

	return out
}

// frames cuts a signal into level readings, one per window.
func frames(
	samples []float64,
	frame int,
) []float64 {
	out := make([]float64, 0, len(samples)/frame)
	for at := 0; at+frame <= len(samples); at += frame {
		out = append(out, loudness(samples[at:at+frame]))
	}

	return out
}

// loudness is the root mean square of a run of samples.
//
// What "how loud is this" means for a signal rather than for one sample: the
// level a steady tone would have to sit at to carry the same energy.
func loudness(
	samples []float64,
) float64 {
	if len(samples) == 0 {
		return 0
	}

	var sum float64
	for _, v := range samples {
		sum += v * v
	}

	return math.Sqrt(sum / float64(len(samples)))
}
