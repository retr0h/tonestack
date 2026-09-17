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

// The keys a measurement is written under in a rig's evidence.
//
// A rig's `measured` block exists so a figure can be compared against the same
// figure taken from somewhere else. That comparison is only possible when both
// sides agree what to call things, so these names are fixed here rather than
// chosen per rig. The contract keeps the field open to any key, the way
// EvidenceKind is open, and this is what the tool writes.
//
// Every one is a number this package produces. A key naming something nothing
// measures would be a promise with nobody behind it.
const (
	// KeyLow, KeyMid and KeyHigh are each band's share of the energy, 0 to 1.
	KeyLow  = "low"
	KeyMid  = "mid"
	KeyHigh = "high"
	// KeyCentroid is the spectrum's centre of gravity, in hertz.
	KeyCentroid = "centroid"
	// KeyTransient is how sharply notes start, 0 to 1.
	KeyTransient = "transient"
	// KeyDecay is how long a note takes to fall to a quarter, in seconds.
	KeyDecay = "decay"
	// KeyDynamics is the gap between loudest and typical, in decibels.
	KeyDynamics = "dynamics"
	// KeyHarmonics is the share of energy above the fundamental, 0 to 1.
	KeyHarmonics = "harmonics"
	// KeyLean is positive for even harmonics and negative for odd, -1 to 1.
	KeyLean = "lean"
)

// MeasuredKeys is every key a measurement is written under, in the order it
// is written.
//
// An order rather than a set, because the map a measurement comes back as has
// none, and evidence written twice from the same recording should be the same
// text both times. A diff that moves lines around hides the one line that
// changed.
func MeasuredKeys() []string {
	return []string{
		KeyLow, KeyMid, KeyHigh,
		KeyCentroid, KeyTransient, KeyDecay, KeyDynamics,
		KeyHarmonics, KeyLean,
	}
}

// Measured is what one recording measures as, keyed for a rig's evidence.
//
// Rounded to what the measurement can honestly claim. A centroid is reported
// to the hertz because the bins are about 10Hz apart, and a share to two
// places because the third moves with which windows the playing fell in.
func (p Profile) Measured() map[string]float64 {
	return map[string]float64{
		KeyLow:       to(p.Low, 2),
		KeyMid:       to(p.Mid, 2),
		KeyHigh:      to(p.High, 2),
		KeyCentroid:  to(p.Centroid, 0),
		KeyTransient: to(p.Transient, 2),
		KeyDecay:     to(p.Decay, 2),
		KeyDynamics:  to(p.DynamicRange, 1),
		KeyHarmonics: to(p.Harmonics.Mid, 2),
		KeyLean:      to(p.EvenOdd.Mid, 2),
	}
}

// Measured is what several recordings measure as together, keyed for a rig's
// evidence.
//
// The middle of each range. The width is not written into a rig: evidence is
// attached per source, and a width belongs to a set of sources rather than to
// any one of them. What the width is for is deciding whether to trust the
// middle at all, which is a judgement made while reading the report rather
// than a number to carry.
func (a Across) Measured() map[string]float64 {
	return map[string]float64{
		KeyLow:       to(a.Low.Mid, 2),
		KeyMid:       to(a.Mid.Mid, 2),
		KeyHigh:      to(a.High.Mid, 2),
		KeyCentroid:  to(a.Centroid.Mid, 0),
		KeyTransient: to(a.Transient.Mid, 2),
		KeyDecay:     to(a.Decay.Mid, 2),
		KeyDynamics:  to(a.DynamicRange.Mid, 1),
		KeyHarmonics: to(a.Harmonics.Mid, 2),
		KeyLean:      to(a.EvenOdd.Mid, 2),
	}
}

// to rounds a measurement to the places it can honestly claim.
func to(
	v float64,
	places int,
) float64 {
	scale := math.Pow(10, float64(places))

	return math.Round(v*scale) / scale
}
