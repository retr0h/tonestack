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
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/retr0h/tonestack/pkg/cli/internal/paint"
	"github.com/retr0h/tonestack/pkg/sdk/audio"
)

// Players prints what each player's records earn them against the others.
//
// The comparison is the whole method, so the table says what everybody else
// read as well as what this player read. A term with no numbers beside it is
// an assertion; with them it is an argument somebody can check.
func Players(
	w io.Writer,
	of []audio.Player,
) error {
	rows := make([][]string, 0, len(of))

	for _, p := range of {
		rows = append(rows, []string{
			paint.Accent(w, p.ID),
			fmt.Sprintf("%d", p.Records),
			termsOf(w, p),
			holdsOf(p),
			paint.Mute(w, againstOf(p)),
		})
	}

	return paint.Section{
		Title:   "What the records say",
		Detail:  playersRead(len(of)),
		Headers: []string{"player", "records", "earns", "holds", "against the others"},
		Rows:    rows,
		Align: []lipgloss.Position{
			lipgloss.Left, lipgloss.Right, lipgloss.Left, lipgloss.Left, lipgloss.Left,
		},
		Empty: "no players to compare: one directory of recordings each",
		Summary: "a word is earned by sitting clear of the other players, " +
			"and the margin is how far the rest of them would have to move to take it",
	}.Render(w)
}

// termsOf is the words a player earned, or what it means that they earned
// none.
func termsOf(
	w io.Writer,
	p audio.Player,
) string {
	if len(p.Terms) == 0 {
		return paint.Mute(w, "nothing")
	}

	words := make([]string, 0, len(p.Terms))
	for _, t := range p.Terms {
		words = append(words, t.Term)
	}

	return strings.Join(words, ", ")
}

// holdsOf says how well each word stands up, as the margin past the line it
// had to clear.
//
// A word that flips when the population changes and one that never will read
// the same until this is beside them: 39% of the energy in the mid band
// against 4% is not the same claim as 8% against 4%, and both print as
// `mid-forward`.
func holdsOf(
	p audio.Player,
) string {
	if len(p.Terms) == 0 {
		return ""
	}

	out := make([]string, 0, len(p.Terms))

	for _, t := range p.Terms {
		out = append(out, fmt.Sprintf("%s: clear by %s", t.Term, figure(t.Key, t.Margin)))
	}

	return strings.Join(out, "; ")
}

// againstOf is the figures behind each word: this player, then the middle of
// everybody else.
func againstOf(
	p audio.Player,
) string {
	if len(p.Terms) == 0 {
		return ""
	}

	out := make([]string, 0, len(p.Terms))
	for _, t := range p.Terms {
		out = append(out, fmt.Sprintf("%s %s against %s",
			t.Key, figure(t.Key, t.Mine), figure(t.Key, t.Others)))
	}

	return strings.Join(out, "; ")
}

// figure prints one measure in the unit it is measured in.
//
// A centroid is hertz and the rest are shares of the whole, and printing a
// share as 264 or a frequency as 26400% is how a table stops being read.
//
// A share under one percent keeps a decimal. Whole percents are enough for
// where a player sits and not for how far past the line they sit: Geddy Lee
// clears `mid-forward` by four tenths of a percent, and rounding that to "0%"
// hides the one thing the column is for.
func figure(
	key string,
	v float64,
) string {
	if key == audio.KeyCentroid {
		return fmt.Sprintf("%.0f Hz", v)
	}

	if share := v * 100; share < 1 {
		return fmt.Sprintf("%.1f%%", share)
	}

	return fmt.Sprintf("%.0f%%", v*100)
}

// playersRead says how many players were compared.
func playersRead(
	n int,
) string {
	if n == 1 {
		return "1 player, which is nobody to compare against"
	}

	return fmt.Sprintf("%d players", n)
}
