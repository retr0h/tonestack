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
// The compressor's Attack has a measurement, and it measures a different
// thing from the word. transient reads how sharply a note's level rises.
// `percussive` on slap bass describes the character of the attack, the click
// of a string against a fret, which is spectral rather than a rise in level.
//
// Measured across three artists, the one whose rig says `percussive` scored
// lowest of the three. That was suspected to mean transient was really
// reading note density, since it is largest where there is silence to rise
// from. It is not: across 582 chunks of three seconds, with density running
// from nothing to continuous, the correlation is +0.026. transient is sound,
// and Flea rising less sharply than a picked punk bassist is true.
//
// So neither the word nor the figure is wrong, and no threshold between them
// would mean anything. The axis stays out until something measures the click.
//
// Measuring the click was tried, and it cannot be done from these stems. The
// measure was fixed before any music was run: the share of a note's energy
// above 1kHz in its first 30ms. It separates a plain synthetic pluck (0.000)
// from one with a click added (0.095). On five artists' separated bass stems
// every figure sat at the noise floor, 0.0000 to 0.0060, because the stems
// carry almost nothing above 1kHz; the high band reads 0 to 2% for every
// artist measured. The click lives in exactly what separation takes out.
// Recordings that keep the bass's top end, a DI or a separation that does not
// band-limit, would be needed first. A lower cutoff was not tried, because
// choosing one after seeing these figures would fit the measure to them.
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
	// Margin is how far past the line this player sits, in the measure's own
	// units: how much the other players' quartile would have to move to take
	// the word away.
	//
	// Two words that read alike are not alike. 39% of the energy in the mid
	// band against 4% is a word nothing will overturn; 8% against 4% flipped
	// the first time somebody else was measured, and nothing in the word
	// said which kind it was.
	//
	// This is the only stability figure there is, and it is about the other
	// players rather than this one. Dropping one of a player's own records
	// and deriving again cannot take a word away: the test is their whole
	// range against a quartile, the range is read as a tenth and a ninetieth
	// percentile, and removing a record can only raise the tenth or lower
	// the ninetieth. It was built, run against the nine players here, and
	// reported nothing for every word, which is what the arithmetic says it
	// must do.
	Margin float64
}

// Derive is what an artist's measurements say about them, against others
// measured the same way.
//
// An artist earns a term only where their whole range sits outside the middle
// half of the other artists: above the others' upper quartile, or below their
// lower quartile. So a word means more than most of the population, not more
// than every member of it.
//
// It was more than every member of it at first, and that rule got stricter
// with every artist added. Each one could only raise the highest middle or
// lower the lowest, so with five artists at most one could clear an axis in
// each direction. One player earned three terms against two other artists and
// none against four, which is more data giving fewer answers.
//
// The quartiles are interpolated between neighbours. Taken on a single
// neighbour, the upper quartile of four artists is simply the highest of them,
// which is the old rule back again.
//
// The whole range rather than the middle, because the middle alone is not
// enough: one artist reads 0% mid across two records and 15% on a third, so
// his median is a figure he never actually plays, and a term drawn from it
// would assert something his own records contradict.
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

		slices.Sort(rest)
		span := ax.of(mine)

		switch {
		case span.Low > between(rest, 0.75):
			out = append(out, made(
				ax, ax.More, mine, rest, len(others)+1,
				span.Low-between(rest, 0.75)))
		case span.High < between(rest, 0.25):
			out = append(out, made(
				ax, ax.Less, mine, rest, len(others)+1,
				between(rest, 0.25)-span.High))
		}
	}

	return out
}

// made is one derived term, carrying what it was derived from.
//
// rest arrives sorted, and is never empty: Derive returns early when it is.
func made(
	ax Axis,
	term string,
	mine Across,
	rest []float64,
	of int,
	margin float64,
) Derived {
	return Derived{
		Term: term,
		Key:  ax.Key,
		Why:  ax.Why,
		Mine: mine.Measured()[ax.Key],
		// The middle of the others, so a report can say what this was
		// clear of rather than only that it was.
		Others: between(rest, 0.5),
		Of:     of,
		Margin: margin,
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

// between is the value a share of the way through a sorted list, interpolated
// between the two entries either side of that point.
//
// quantile in measure.go takes the entry the point lands on, which is right
// for a spread over hundreds of windows and wrong for a handful of artists:
// there, the upper quartile of four lands on the highest of them.
func between(
	sorted []float64,
	share float64,
) float64 {
	at := share * float64(len(sorted)-1)
	lo := int(at)

	// Only a list of one has nothing above its only entry. For anything
	// longer, share below one keeps lo below the last index.
	if lo+1 >= len(sorted) {
		return sorted[lo]
	}

	return sorted[lo] + (at-float64(lo))*(sorted[lo+1]-sorted[lo])
}
