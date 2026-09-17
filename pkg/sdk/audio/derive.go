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

import "slices"

// Turning a measurement into a word, by where an artist sits among artists
// measured the same way.
//
// A figure on its own names nothing. 91% of the energy below 250Hz is not
// "scooped", because every isolated bass stem is mostly low: that figure
// describes the instrument. What names something is a player sitting clear of
// the others on a measure all of them were read for.

// Axis is one measure that can become a word, and the two words it becomes.
type Axis struct {
	// Key is the measured figure this axis reads, from MeasuredKeys.
	Key string
	// More is the term for an artist above the rest, Less for one below.
	More, Less string
	// Why says what the figure is, for the evidence a derived term carries.
	Why string

	// of is the range this axis reads from a gathered measurement.
	//
	// Carried here rather than looked up by Key, so there is no branch for a
	// measure no axis names. A switch over every key would have four arms
	// nothing can reach, which is a thing to delete rather than to test
	// around.
	of func(Across) Spread
}

// Axes is every measure a term may be derived from, and only those.
//
// Three of the six axes a term can move. The other three are left alone for
// reasons that are not oversights:
//
// Sag and reverb Mix have no measurement at all. Touch response and how much
// room is on a recording are not a band share or a centroid, and picking the
// nearest available figure because both happen to exist is how a number ends
// up deciding a control it has nothing to do with.
//
// The compressor's Attack has a measurement that is not trusted yet. transient
// reads the largest rise against the peak, so it is largest where there is
// silence to rise from, and a player who leaves gaps between phrases scores
// higher than one who plays continuously. Measured across three artists the
// one whose rig says `percussive` scored lowest of the three, which is either
// a wrong word or a measure of note density wearing the name of attack.
// Nothing derives from it until that is settled.
var Axes = []Axis{
	{
		Key: KeyMid, More: "mid-forward", Less: "scooped",
		Why: "share of energy between 250Hz and 2kHz",
		of:  func(a Across) Spread { return a.Mid },
	},
	{
		Key: KeyCentroid, More: "bright", Less: "dark",
		Why: "the spectrum's centre of gravity, in hertz",
		of:  func(a Across) Spread { return a.Centroid },
	},
	{
		Key: KeyHarmonics, More: "saturated", Less: "clean",
		Why: "share of energy above the fundamental",
		of:  func(a Across) Spread { return a.Harmonics },
	},
}

// Derived is one term an artist earned, and what earned it.
type Derived struct {
	// Term is the word, spelled as the compiler's own table spells it.
	Term string
	// Key is the measure it came from, and Why describes that measure.
	Key, Why string
	// Mine is where this artist sat, Others the middle of everyone else.
	Mine, Others float64
	// Of is how many artists the comparison was against, this one included.
	Of int
}

// Derive is what an artist's measurements say about them, against others
// measured the same way.
//
// An artist earns a term only where their whole range sits clear of what the
// other artists' middles cover. The middle alone is not enough: one of the
// three artists measured so far reads 0% mid across two records and 15% on a
// third, so his median is a figure he never actually plays, and a term drawn
// from it would assert something his own records contradict.
//
// Nothing is derived where the evidence is mixed. With few artists that will
// be most of the time, and saying nothing is the right answer rather than a
// failure: a rig that names no term keeps the words a person chose.
func Derive(
	mine Across,
	others map[string]Across,
) []Derived {
	if mine.Tracks == 0 || len(others) == 0 {
		return nil
	}

	var out []Derived

	for _, ax := range Axes {
		rest := middles(others, ax.Key)
		if len(rest) == 0 {
			continue
		}

		span := ax.of(mine)
		low, high := slices.Min(rest), slices.Max(rest)

		switch {
		case span.Low > high:
			out = append(out, made(ax, ax.More, mine, rest, len(others)+1))
		case span.High < low:
			out = append(out, made(ax, ax.Less, mine, rest, len(others)+1))
		}
	}

	return out
}

// made is one derived term, carrying what it was derived from.
//
// rest is never empty here: Derive returns early when it is.
func made(
	ax Axis,
	term string,
	mine Across,
	rest []float64,
	of int,
) Derived {
	sorted := slices.Clone(rest)
	slices.Sort(sorted)

	return Derived{
		Term: term,
		Key:  ax.Key,
		Why:  ax.Why,
		Mine: mine.Measured()[ax.Key],
		// The middle of the others, so a report can say what this was
		// clear of rather than only that it was.
		Others: quantile(sorted, 0.5),
		Of:     of,
	}
}

// middles is where each of the other artists sits on one measure.
//
// An artist with no recordings is not a vote. Their Across is empty, and
// counting a zero as a position would drag every comparison toward it.
func middles(
	others map[string]Across,
	key string,
) []float64 {
	out := make([]float64, 0, len(others))

	for _, a := range others {
		if a.Tracks == 0 {
			continue
		}

		out = append(out, a.Measured()[key])
	}

	return out
}
