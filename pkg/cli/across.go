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

// Tracks prints one row per recording, before they are read together.
//
// The rows come first because the gathered answer hides which record was
// unusual, and which record was unusual is often the thing worth chasing. A
// stem that is half silence reports a decay the others do not, and seeing
// that beside its name is what turns it into a question.
func Tracks(
	w io.Writer,
	of []audio.Named,
) error {
	rows := make([][]string, 0, len(of))

	for _, n := range of {
		p := n.Profile

		rows = append(rows, []string{
			paint.Accent(w, n.Name),
			fmt.Sprintf("%.0f%%", p.Low*100),
			fmt.Sprintf("%.0f Hz", p.Centroid),
			readingOf(p.Transient, "%.2f", 1),
			readingOf(p.Decay, "%.2f s", 1),
			fmt.Sprintf("%.1f dB", p.DynamicRange),
			fmt.Sprintf("%.0f%%", p.Harmonics.Mid*100),
		})
	}

	return paint.Section{
		Title:  "Each record",
		Detail: recordsRead(len(of)),
		Headers: []string{
			"record", "low", "centroid", "transient", "decay", "dynamics",
			"harmonics",
		},
		Rows: rows,
		Align: []lipgloss.Position{
			lipgloss.Left, lipgloss.Right, lipgloss.Right, lipgloss.Right,
			lipgloss.Right, lipgloss.Right, lipgloss.Right,
		},
		Empty: "no recordings to measure",
	}.Render(w)
}

// Across prints what several recordings measure as together.
//
// The middle of each measure and the width the records put around it. Neither
// is a verdict: a centroid that sits at 150Hz across four records is a fact
// about those four records, and calling it warm would be an argument this
// does not join.
func Across(
	w io.Writer,
	a audio.Across,
) error {
	rows := [][]string{
		spreadRow(w, "low", "%.0f%%", a.Low, 100),
		spreadRow(w, "mid", "%.0f%%", a.Mid, 100),
		spreadRow(w, "high", "%.0f%%", a.High, 100),
		spreadRow(w, "centroid", "%.0f Hz", a.Centroid, 1),
		rangedRow(w, "transient", "%.2f", a.Transient, 1, a.Tracks),
		rangedRow(w, "decay", "%.2f s", a.Decay, 1, a.Tracks),
		spreadRow(w, "dynamics", "%.1f dB", a.DynamicRange, 1),
		spreadRow(w, "harmonics", "%.0f%%", a.Harmonics, 100),
		{
			paint.Accent(w, "lean"),
			fmt.Sprintf("%+.2f", a.EvenOdd.Mid),
			paint.Mute(w, leanReads(a.EvenOdd.Mid)),
		},
	}

	return paint.Section{
		Title:   "Across the records",
		Detail:  recordsRead(a.Tracks),
		Headers: []string{"measure", "middle", "across the records"},
		Rows:    rows,
		Align: []lipgloss.Position{
			lipgloss.Left, lipgloss.Right, lipgloss.Right,
		},
		Empty:   "no recordings to measure",
		Summary: "the middle is what the records share; the width is how much the record chosen moved it",
	}.Render(w)
}

// spreadRow is one measure's middle and the range the records put around it.
func spreadRow(
	w io.Writer,
	name, format string,
	s audio.Spread,
	scale float64,
) []string {
	return []string{
		paint.Accent(w, name),
		fmt.Sprintf(format, s.Mid*scale),
		paint.Mute(w, fmt.Sprintf(format+"–"+format, s.Low*scale, s.High*scale)),
	}
}

// rangedRow is one measure the records did not all answer.
//
// How many answered is said out loud when it is fewer than all of them. A
// middle taken over two records of four is still a measurement of those two,
// but reading it beside a middle taken over all four without knowing which is
// which is how a corpus quietly becomes a smaller corpus.
func rangedRow(
	w io.Writer,
	name, format string,
	r audio.Ranged,
	scale float64,
	tracks int,
) []string {
	if r.From == 0 {
		return []string{
			paint.Accent(w, name),
			"—",
			paint.Mute(w, "no record answered this"),
		}
	}

	span := fmt.Sprintf(format+"–"+format, r.Low*scale, r.High*scale)
	if r.From < tracks {
		span += fmt.Sprintf(", from %d of %d", r.From, tracks)
	}

	return []string{
		paint.Accent(w, name),
		fmt.Sprintf(format, r.Mid*scale),
		paint.Mute(w, span),
	}
}

// recordsRead says how many recordings the numbers came from.
//
// The count is load-bearing rather than decoration: with four records the
// ends of every width are the extreme records, so a reader needs to know it
// was four and not forty.
func recordsRead(
	n int,
) string {
	if n == 1 {
		return "1 recording"
	}

	return fmt.Sprintf("%d recordings", n)
}
