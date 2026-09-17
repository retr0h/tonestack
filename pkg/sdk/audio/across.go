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

// Across is what several recordings measure as together.
//
// One record is one engineer's decisions on one day, and a number taken from
// it describes that day as much as the player. Several records disagree, and
// the disagreement is the useful part: the middle is what they have in
// common, and the width is how much the choice of record moved it.
//
// Measured on four bass stems by one player, the centroid's middle sat at
// 150Hz and its width ran 135Hz to 191Hz. The same four measured as full
// mixes ran 842Hz to 1338Hz, which is the band rather than the bass.
type Across struct {
	// Tracks is how many recordings went into it.
	Tracks int

	// Low, Mid and High are each band's share, across the recordings.
	Low, Mid, High Spread

	// Centroid is the spectrum's centre of gravity, in hertz.
	Centroid Spread
	// Transient is how sharply notes start, from zero to one. Only the
	// recordings whose level rose at all are in it.
	Transient Ranged
	// Decay is how long a note takes to fall to a quarter, in seconds. Only
	// the recordings whose notes fell that far are in it.
	Decay Ranged
	// DynamicRange is the gap between loudest and typical, in decibels.
	DynamicRange Spread
	// Harmonics is the share of energy above the fundamental.
	Harmonics Spread
	// EvenOdd leans positive for even harmonics and negative for odd.
	EvenOdd Spread
}

// Together gathers several recordings into one measurement.
//
// Each recording contributes one number per measure, and the spread of those
// numbers is what comes back. Where a recording's own measure is already a
// range, its middle is the number it contributes: a track is one voice here,
// not a distribution of them.
//
// With few enough recordings the ends of the spread are the extreme records
// rather than a tenth in from them, because a tenth of four is none. That is
// worth knowing when reading a width taken from four songs: one unusual
// record is the whole of one end.
func Together(
	of []Profile,
) Across {
	out := Across{Tracks: len(of)}
	if len(of) == 0 {
		return out
	}

	out.Low = each(of, func(p Profile) float64 { return p.Low })
	out.Mid = each(of, func(p Profile) float64 { return p.Mid })
	out.High = each(of, func(p Profile) float64 { return p.High })

	out.Centroid = each(of, func(p Profile) float64 { return p.Centroid })
	out.DynamicRange = each(of, func(p Profile) float64 { return p.DynamicRange })

	out.Transient = whichever(of, func(p Profile) Reading { return p.Transient })
	out.Decay = whichever(of, func(p Profile) Reading { return p.Decay })

	out.Harmonics = each(of, func(p Profile) float64 { return p.Harmonics.Mid })
	out.EvenOdd = each(of, func(p Profile) float64 { return p.EvenOdd.Mid })

	return out
}

// Ranged is what several recordings measured, and how many of them could be.
//
// Not every recording answers every question. A note held through a whole bar
// never falls to a quarter of itself, so it has no decay to contribute, and a
// spread taken over the recordings that do plus the ones that do not is a
// spread over two different things.
type Ranged struct {
	Spread

	// From is how many recordings contributed a reading. Zero means none
	// could, and the spread inside is empty rather than small.
	From int
}

// whichever is the spread of one measure across the recordings that had it.
//
// Silence from the ones that did not, rather than a zero. A zero is a
// measurement somebody took and got nothing; an absent recording is one
// nobody could take, and averaging the second into the first drags every
// answer toward zero for a reason that has nothing to do with the playing.
func whichever(
	of []Profile,
	pick func(Profile) Reading,
) Ranged {
	vals := make([]float64, 0, len(of))

	for _, p := range of {
		if r := pick(p); r.Known {
			vals = append(vals, r.Value)
		}
	}

	return Ranged{Spread: spreadOf(vals), From: len(vals)}
}

// each is the spread of one measure across the recordings.
func each(
	of []Profile,
	pick func(Profile) float64,
) Spread {
	vals := make([]float64, 0, len(of))

	for _, p := range of {
		vals = append(vals, pick(p))
	}

	return spreadOf(vals)
}
