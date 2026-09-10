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

	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/chain"
)

// Category colours.
//
// A chain is read by shape before it is read by word — where the amp sits,
// whether anything comes after the cab. Colouring by what a block does makes
// that shape visible without adding a column.
//
// These are domain colours rather than theme roles: they mean the same thing
// whichever theme is active, the way a status colour does.
var categoryColor = map[catalog.Category]lipgloss.Color{
	catalog.CategoryAmp:    lipgloss.Color("#ffa032"),
	catalog.CategoryCab:    lipgloss.Color("#b06a1e"),
	catalog.CategoryDrive:  lipgloss.Color("#e0864a"),
	catalog.CategoryComp:   lipgloss.Color("#7fa8d0"),
	catalog.CategoryDelay:  lipgloss.Color("#9a86c8"),
	catalog.CategoryReverb: lipgloss.Color("#7f9fc8"),
	catalog.CategoryEQ:     lipgloss.Color("#8fc0a8"),
	catalog.CategoryMod:    lipgloss.Color("#c88fb4"),
	catalog.CategoryGate:   lipgloss.Color("#6f9ab8"),
	catalog.CategoryWah:    lipgloss.Color("#d0a0c8"),
	catalog.CategoryPitch:  lipgloss.Color("#a8b0d8"),
	catalog.CategoryFilter: lipgloss.Color("#9fc8b8"),
}

// Category renders a category name in the colour that block type carries
// everywhere else, so a one-line chain summary and a full chain agree.
func Category(w io.Writer, c catalog.Category) string {
	col, ok := categoryColor[c]
	if !ok {
		return Mute(w, string(c))
	}

	return render(w, lipgloss.NewStyle().Foreground(col), string(c))
}

// Meter draws a proportion as a bar.
//
// DSP is the constraint a chain lives inside, and a number alone does not say
// how close to the edge it is. A bar does, at a glance.
func Meter(w io.Writer, pct float64, width int) string {
	filled := int(pct / 100 * float64(width))
	if filled > width {
		filled = width
	}

	if filled < 0 {
		filled = 0
	}

	style := active.OK
	switch {
	case pct >= 90:
		style = active.Err
	case pct >= 75:
		style = active.Info
	}

	return render(w, style, strings.Repeat("█", filled)) +
		render(w, active.Mute, strings.Repeat("░", width-filled))
}

// Chain renders a signal chain.
//
// Blocks arrive in the order the device runs them, so they are printed in
// that order and the processor totals follow.
func Chain(w io.Writer, spec chain.Chain, cat *catalog.Catalog) error {
	rows := make([][]string, 0, len(spec.Blocks))
	used := map[int]float64{}

	for _, b := range spec.Blocks {
		blk, known := cat.Block(b.Model)
		used[b.DSP] += blk.DSP.Mono

		rows = append(rows, []string{
			state(w, b.Enabled),
			fmt.Sprintf("%d.%d", b.DSP, b.Pos),
			blockName(w, blk, b.Model, known),
			Mute(w, basedOn(blk, b)),
			cost(w, blk.DSP.Mono, known),
		})
	}

	if len(rows) == 0 {
		_, err := fmt.Fprintln(w, Indent+Mute(w, "empty"))

		return err
	}

	if err := Table(w, rows, []lipgloss.Position{
		lipgloss.Left, lipgloss.Left, lipgloss.Left, lipgloss.Left, lipgloss.Right,
	}); err != nil {
		return err
	}

	if err := budget(w, used); err != nil {
		return err
	}

	return userIRs(w, spec)
}

// basedOn describes what a block imitates.
//
// A user impulse response imitates nothing the catalog knows; it plays
// whatever is in a slot of the owner's IR library, so the slot is the useful
// thing to show.
func basedOn(blk catalog.Block, b chain.Block) string {
	if !catalog.NeedsUserIR(b.Model) {
		return blk.BasedOn
	}

	if idx, ok := b.Params["Index"]; ok {
		return "IR slot " + idx.String()
	}

	return "a user IR"
}

// userIRs warns about blocks that depend on the device owner's IR library.
//
// This is the one thing in a preset that the file does not carry. Somebody
// opening a preset from elsewhere needs to know that part of its sound lives
// on the machine it came from.
func userIRs(w io.Writer, spec chain.Chain) error {
	var slots []string

	for _, b := range spec.Blocks {
		if !catalog.NeedsUserIR(b.Model) {
			continue
		}

		if idx, ok := b.Params["Index"]; ok {
			slots = append(slots, idx.String())
		}
	}

	if len(slots) == 0 {
		return nil
	}

	_, err := fmt.Fprintf(w, "\n%s%s\n", Indent, Info(w, fmt.Sprintf(
		"plays your own impulse responses from slot %s — this preset sounds "+
			"like whoever made it only if the same IRs are loaded there",
		strings.Join(slots, ", "))))

	return err
}

// state marks whether a block is doing anything.
//
// A bypassed block still occupies its position and still costs DSP, so it is
// shown rather than hidden, but dimmed so a chain reads as what it sounds
// like rather than what it contains.
func state(w io.Writer, enabled bool) string {
	if enabled {
		return OK(w, "●")
	}

	return Mute(w, "○")
}

// blockName colours a block by category, and says so when the catalog has
// never heard of it.
func blockName(
	w io.Writer,
	blk catalog.Block,
	id catalog.ModelID,
	known bool,
) string {
	if !known {
		return Info(w, string(id)) + Mute(w, " (not in catalog)")
	}

	if col, ok := categoryColor[blk.Category]; ok {
		return render(w, lipgloss.NewStyle().Foreground(col), blk.Name)
	}

	return blk.Name
}

// cost renders a DSP figure, or nothing when it is not known.
func cost(w io.Writer, mono float64, known bool) string {
	if !known {
		return Mute(w, "?")
	}

	return fmt.Sprintf("%.1f", mono)
}

// budget prints how much of each processor the chain uses.
func budget(w io.Writer, used map[int]float64) error {
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}

	for dsp := 0; dsp < 2; dsp++ {
		pct, ok := used[dsp]
		if !ok {
			continue
		}

		_, err := fmt.Fprintf(w, "%s%s  %s  %s\n",
			Indent,
			Mute(w, fmt.Sprintf("dsp%d", dsp)),
			Meter(w, pct, 24),
			Accent(w, fmt.Sprintf("%.1f%%", pct)),
		)
		if err != nil {
			return err
		}
	}

	return nil
}
