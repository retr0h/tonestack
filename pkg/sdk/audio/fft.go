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

import "math"

// Turning a window of samples into how much of each frequency is in it.
//
// Written here rather than taken as a dependency. It is one well-known
// algorithm in forty lines, this project carries no other signal processing,
// and a measurement nobody can read the arithmetic of is a measurement nobody
// should trust.

// spectrum is the magnitude of each frequency bin in one window.
//
// Only the first half of a transform is kept. A real signal's transform is
// symmetric, so the upper half repeats the lower and counting it twice would
// put the middle of the spectrum in the wrong place.
type spectrum struct {
	// mag is the magnitude per bin, from nothing up to half the sample rate.
	mag []float64
	// rate is how many samples a second the window was taken at, which is
	// what turns a bin number into a frequency.
	rate int
}

// hzPerBin is how far apart two neighbouring bins are.
func (s spectrum) hzPerBin() float64 {
	if len(s.mag) == 0 {
		return 0
	}

	// len(mag) is half the window, so the window is twice it.
	return float64(s.rate) / float64(len(s.mag)*2)
}

// hz is the frequency at the middle of one bin.
func (s spectrum) hz(
	bin int,
) float64 {
	return float64(bin) * s.hzPerBin()
}

// window is how many samples one transform covers.
//
// At 44.1kHz this is about 93 milliseconds, putting the bins roughly 10.8Hz
// apart. Fine enough to tell a bass fundamental from its second harmonic, and
// long enough to resolve a low note at all: an open E at 41Hz needs several
// of its cycles inside the window before it has a frequency to report.
const window = 4096

// hop is how far the window moves each time, at half its length.
//
// Overlapping, so a note that starts in the middle of one window is whole in
// the next. Without it a note landing on a boundary is split between two
// windows and clearly present in neither.
const hop = window / 2

// spectra cuts a recording into overlapping windows and takes each one's
// spectrum.
//
// Only where the instrument is sounding. A window of silence has a spectrum
// like any other, and letting the rests vote is how a measurement ends up
// describing the room instead of the playing.
//
// A recording shorter than one window is measured whole, which is what the
// synthetic signals in the tests are.
func spectra(
	samples []float64,
	rate int,
) []spectrum {
	if len(samples) < window {
		return []spectrum{analyse(samples, rate)}
	}

	starts := make([]int, 0, len(samples)/hop)
	levels := make([]float64, 0, len(samples)/hop)

	for at := 0; at+window <= len(samples); at += hop {
		starts = append(starts, at)
		levels = append(levels, loudness(samples[at:at+window]))
	}

	held := playing(levels)
	out := make([]spectrum, 0, len(starts))

	for i, at := range starts {
		if held[i] {
			out = append(out, analyse(samples[at:at+window], rate))
		}
	}

	return out
}

// analyse takes the spectrum of one window of samples.
//
// The window is padded up to a power of two, because that is the only length
// this transform divides evenly. Padding with silence spreads each peak a
// little; it does not move it, which is what the measurements read.
func analyse(
	samples []float64,
	rate int,
) spectrum {
	if len(samples) == 0 {
		return spectrum{rate: rate}
	}

	n := 1
	for n < len(samples) {
		n *= 2
	}

	re := make([]float64, n)
	im := make([]float64, n)

	// A raised cosine over the samples that are really there. Reading a window
	// with square edges reports the edges as well as the sound, and those show
	// up as high frequencies nobody played.
	for i, v := range samples {
		re[i] = v * hann(i, len(samples))
	}

	transform(re, im)

	half := n / 2
	out := spectrum{mag: make([]float64, half), rate: rate}

	for i := range half {
		out.mag[i] = math.Hypot(re[i], im[i])
	}

	return out
}

// hann is the window shape, tapering to nothing at both ends.
func hann(
	i, n int,
) float64 {
	if n <= 1 {
		return 1
	}

	return 0.5 * (1 - math.Cos(2*math.Pi*float64(i)/float64(n-1)))
}

// transform runs an in-place radix-2 fast Fourier transform.
//
// re and im must be the same length and that length a power of two, which
// analyse is what guarantees.
func transform(
	re, im []float64,
) {
	n := len(re)
	if n <= 1 {
		return
	}

	// Reorder into bit-reversed positions, so the combining below can walk
	// pairs that are already neighbours.
	for i, j := 1, 0; i < n; i++ {
		bit := n >> 1
		for ; j&bit != 0; bit >>= 1 {
			j ^= bit
		}

		j ^= bit

		if i < j {
			re[i], re[j] = re[j], re[i]
			im[i], im[j] = im[j], im[i]
		}
	}

	// Combine pairs, then pairs of pairs, doubling the span each time.
	for span := 2; span <= n; span *= 2 {
		angle := -2 * math.Pi / float64(span)
		wRe, wIm := math.Cos(angle), math.Sin(angle)

		for start := 0; start < n; start += span {
			curRe, curIm := 1.0, 0.0

			for k := range span / 2 {
				i, j := start+k, start+k+span/2

				tRe := curRe*re[j] - curIm*im[j]
				tIm := curRe*im[j] + curIm*re[j]

				re[j], im[j] = re[i]-tRe, im[i]-tIm
				re[i], im[i] = re[i]+tRe, im[i]+tIm

				curRe, curIm = curRe*wRe-curIm*wIm, curRe*wIm+curIm*wRe
			}
		}
	}
}
