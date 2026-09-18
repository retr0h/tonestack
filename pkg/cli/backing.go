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
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/retr0h/tonestack/pkg/cli/internal/paint"
	sdk "github.com/retr0h/tonestack/pkg/sdk"
)

// Backing prints which records back each rig, and which of them were made
// outside the era the rig describes.
//
// The figures measured from a record are honest and the rig is honest, and
// the join between them can still be wrong: a record made before the
// amplifier existed measures a different rig. Nothing in either file says so
// on its own, which is why this exists.
func Backing(
	w io.Writer,
	all []sdk.Backing,
) error {
	rows := make([][]string, 0, len(all))

	for _, b := range all {
		rows = append(rows, []string{
			paint.Accent(w, b.ID),
			eraOf(w, b),
			recordsOf(w, b),
			roomOf(w, b),
			verdictOf(w, b),
		})
	}

	return paint.Section{
		Title:   "What backs each rig",
		Detail:  ridingOn(all),
		Headers: []string{"rig", "era", "records", "room", "reads"},
		Rows:    rows,
		Align: []lipgloss.Position{
			lipgloss.Left, lipgloss.Left, lipgloss.Left, lipgloss.Left,
			lipgloss.Left,
		},
		Empty: "no rigs to read",
		Summary: "a record made outside a rig's era measures gear the rig " +
			"does not describe, and one made in another room measures gear " +
			"that was never in the signal",
	}.Render(w)
}

// eraOf is the years a rig claims, as it states them.
func eraOf(
	w io.Writer,
	b sdk.Backing,
) string {
	if b.NoRig {
		return paint.Mute(w, "—")
	}

	if !b.Stated() {
		return paint.Info(w, "says none")
	}

	if b.From == b.To {
		return strconv.Itoa(b.From)
	}

	return fmt.Sprintf("%d–%d", b.From, b.To)
}

// recordsOf lists the years measured for a rig, marking the ones outside it.
func recordsOf(
	w io.Writer,
	b sdk.Backing,
) string {
	if len(b.Records) == 0 {
		return paint.Mute(w, "none measured")
	}

	out := make([]string, 0, len(b.Records))

	for _, r := range b.Records {
		year := strconv.Itoa(r.Year)
		if r.Outside {
			year = paint.Err(w, year+"*")
		}

		out = append(out, year)
	}

	return strings.Join(out, " ")
}

// roomOf says whether the gear was in the room the record was made in.
//
// The era check asks whether the records were made when the gear was. This
// asks whether they were made through it. A rig can pass the first and fail
// the second: gear from the right years, documented on a stage, in front of a
// signal that went to the desk.
func roomOf(
	w io.Writer,
	b sdk.Backing,
) string {
	room := ""

	switch {
	case b.Captured == 0:
		room = ""
	case b.Direct > 0:
		room = "direct"
	case b.Both > 0:
		room = "direct and miked"
	default:
		room = "miked"
	}

	switch {
	case b.NoRig:
		return paint.Mute(w, "—")
	case room == "" && b.Stage == 0:
		return paint.Info(w, "not established")
	case room == "":
		return paint.Err(w, "gear from a stage")
	case b.Stage > 0:
		return paint.Err(w, room+", gear from a stage")
	case room == "miked":
		return paint.OK(w, room)
	default:
		return paint.Err(w, room)
	}
}

// verdictOf says what the years amount to.
func verdictOf(
	w io.Writer,
	b sdk.Backing,
) string {
	switch {
	case b.NoRig:
		return paint.Err(w, "no rig is named for this directory")
	case len(b.Misnamed) > 0:
		return paint.Err(w, fmt.Sprintf("no record called %s",
			strings.Join(b.Misnamed, ", ")))
	case len(b.Records) == 0:
		return paint.Mute(w, "nothing measured for it")
	case !b.Stated():
		return paint.Info(w, "no era to hold them to")
	case b.Outside() == 0:
		return paint.OK(w, "records match the era")
	case b.Outside() == len(b.Records):
		return paint.Err(w, "every record is from another era")
	default:
		return paint.Err(w, fmt.Sprintf("%d of %d from another era",
			b.Outside(), len(b.Records)))
	}
}

// ridingOn counts the rigs whose records disagree with their era.
func ridingOn(
	all []sdk.Backing,
) string {
	bad := 0

	for _, b := range all {
		if b.Outside() > 0 || b.NoRig {
			bad++
		}
	}

	if bad == 0 {
		return fmt.Sprintf("%d rigs, every record in era", len(all))
	}

	return fmt.Sprintf("%d rigs, %d with something to answer for", len(all), bad)
}
