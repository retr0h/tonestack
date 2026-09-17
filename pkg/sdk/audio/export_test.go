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

// Exposed to this package's external tests. The transform and the signals it
// is checked against are steps underneath what a caller asks for, and the
// tests that hold them are the reason anything above can be believed.
type Spectrum = spectrum

var (
	Analyse   = analyse
	Transform = transform

	Sine     = sine
	Silence  = silence
	Square   = square
	Noise    = noise
	Plucked  = plucked
	Swell    = swell
	Loudness = loudness

	Playing = playing
	Frames  = frames
)

// Quietest is how far under the loudest moment a frame may sit and still
// count as playing.
const Quietest = quietest

// Reading a spectrum from outside the package, for the tests that hold the
// transform. A caller works in measurements, not in bins.
func (s spectrum) Mag() []float64 { return s.mag }

func (s spectrum) Hz(
	bin int,
) float64 {
	return s.hz(bin)
}

func (s spectrum) HzPerBin() float64 { return s.hzPerBin() }
