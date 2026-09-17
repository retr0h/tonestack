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

package cli

import (
	"fmt"
	"io"

	"github.com/charmbracelet/lipgloss"

	"github.com/retr0h/tonestack/pkg/cli/internal/paint"
	"github.com/retr0h/tonestack/pkg/sdk/audio"
)

// Profile prints what a recording measures as.
//
// Numbers and the units they are in, and nothing that reads as a verdict. A
// centroid of 410Hz is a fact; calling it warm is an argument, and one this
// does not join.
func Profile(
	w io.Writer,
	p audio.Profile,
) error {
	rows := [][]string{
		{
			paint.Accent(w, "energy"),
			fmt.Sprintf("%.0f%% low · %.0f%% mid · %.0f%% high",
				p.Low*100, p.Mid*100, p.High*100),
			paint.Mute(w, "where the sound sits, by band"),
		},
		{
			paint.Accent(w, "centroid"),
			fmt.Sprintf("%.0f Hz", p.Centroid),
			paint.Mute(w, "its centre of gravity"),
		},
		{
			paint.Accent(w, "transient"),
			fmt.Sprintf("%.2f", p.Transient),
			paint.Mute(w, attackReads(p.Transient)),
		},
		{
			paint.Accent(w, "decay"),
			fmt.Sprintf("%.2f s", p.Decay),
			paint.Mute(w, "to a quarter of the loudest moment"),
		},
		{
			paint.Accent(w, "dynamics"),
			fmt.Sprintf("%.1f dB", p.DynamicRange),
			paint.Mute(w, compressionReads(p.DynamicRange)),
		},
		{
			paint.Accent(w, "harmonics"),
			fmt.Sprintf("%.0f%%", p.Harmonics.Mid*100),
			paint.Mute(w, leanReads(p.EvenOdd.Mid)),
		},
		{
			paint.Accent(w, "spread"),
			fmt.Sprintf("%.0f–%.0f%%", p.Harmonics.Low*100, p.Harmonics.High*100),
			paint.Mute(w, "the same measure, a tenth in from either end"),
		},
	}

	return paint.Section{
		Title:   "Measured",
		Detail:  fmt.Sprintf("%.1fs · %d Hz", p.Seconds, p.Rate),
		Headers: []string{"measure", "value", "what it counts"},
		Rows:    rows,
		Align: []lipgloss.Position{
			lipgloss.Left, lipgloss.Right, lipgloss.Left,
		},
		Empty: "nothing to measure",
	}.Render(w)
}

// attackReads says what a transient number is counting, not what it means.
//
// The boundaries come from measuring signals whose answers were known: a
// struck note reads above 0.9 and a note swelled in reads below 0.1. They
// describe how the number was arrived at rather than ruling on the playing.
func attackReads(
	v float64,
) string {
	switch {
	case v >= 0.9:
		return "the level arrives in one step"
	case v <= 0.1:
		return "the level climbs gradually"
	default:
		return "the largest single step up, against the peak"
	}
}

// compressionReads says what the decibel gap is between.
func compressionReads(
	v float64,
) string {
	if v < 6 {
		return "loudest against typical; little room between them"
	}

	return "loudest against typical"
}

// leanReads says which harmonics carry more, without calling it a sound.
//
// Even is a valve's, odd is a fuzz's, and saying so here would be the verdict
// this file is careful not to reach.
func leanReads(
	lean float64,
) string {
	switch {
	case lean > 0.2:
		return "above the fundamental, leaning even"
	case lean < -0.2:
		return "above the fundamental, leaning odd"
	default:
		return "above the fundamental"
	}
}
