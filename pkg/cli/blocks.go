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
	"sort"

	"github.com/retr0h/tonestack/pkg/cli/internal/paint"

	"github.com/charmbracelet/lipgloss"

	"github.com/retr0h/tonestack/pkg/sdk"
	"github.com/retr0h/tonestack/pkg/sdk/catalog"
)

// Blocks prints what a device can do, one block to a row.
func Blocks(w io.Writer, b sdk.Blocks) error {
	rows := make([][]string, 0, len(b.Matched))

	for _, blk := range b.Matched {
		rows = append(rows, []string{
			paint.Accent(w, string(blk.ID)),
			blk.Name,
			paint.Category(w, blk.Category),
			paint.Mute(w, blk.Subcategory),
			fmt.Sprintf("%.1f", blk.DSP.Mono),
			paint.Mute(w, blk.BasedOn),
		})
	}

	return reporting(paint.Section{
		Title:   b.Device,
		Detail:  fmt.Sprintf("%d blocks · %s", b.Total, origin(b.Source)),
		Headers: []string{"model", "name", "category", "sub", "dsp", "based on"},
		Rows:    rows,
		Align: []lipgloss.Position{
			lipgloss.Left, lipgloss.Left, lipgloss.Left,
			lipgloss.Left, lipgloss.Right,
		},
		Empty:   "no blocks match",
		Summary: fmt.Sprintf("%d of %d blocks", len(b.Matched), b.Total),
	}.Render(w))
}

// origin says where a catalog came from.
//
// A catalog that recorded nothing is not the same as one from a source nobody
// recognises, and saying so beats a blank where a provenance should be.
func origin(source string) string {
	if source == "" {
		return "source unknown"
	}

	return source
}

// Block prints one block and everything it accepts.
func Block(w io.Writer, b catalog.Block) error {
	d := paint.Detail{Title: b.Name, Subtitle: string(b.ID)}

	if b.BasedOn != "" {
		d.Fields = append(d.Fields, paint.Field{Label: "based on", Value: b.BasedOn})
	}

	category := string(b.Category)
	if b.Subcategory != "" {
		category += " (" + b.Subcategory + ")"
	}

	dsp := fmt.Sprintf("%.2f mono", b.DSP.Mono)
	if b.DSP.Stereo > 0 {
		dsp += fmt.Sprintf(", %.2f stereo", b.DSP.Stereo)
	}

	d.Fields = append(d.Fields,
		paint.Field{Label: "category", Value: category},
		paint.Field{Label: "dsp", Value: dsp},
	)

	if !b.Prov.Trusted() {
		d.Note = "assumed — this figure was inferred, not stated by Line 6"
	}

	if err := d.Render(w); err != nil {
		return reporting(err)
	}

	return params(w, b)
}

// params prints every parameter a block accepts, in name order.
func params(w io.Writer, b catalog.Block) error {
	keys := make([]string, 0, len(b.Params))
	for k := range b.Params {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	rows := make([][]string, 0, len(keys))

	for _, k := range keys {
		p := b.Params[k]
		rng := "—"

		if p.Type == catalog.ParamFloat || p.Type == catalog.ParamInt {
			rng = fmt.Sprintf("%g..%g", p.Min, p.Max)
		}

		// A parameter that came out of the catalog always marshals.
		def, _ := p.Default.MarshalJSON()

		rows = append(rows, []string{
			paint.Accent(w, k),
			paint.Mute(w, string(p.Type)),
			paint.Mute(w, rng),
			string(def),
		})
	}

	return reporting(paint.Section{
		Headers: []string{"parameter", "kind", "range", "default"},
		Rows:    rows,
		Empty:   "no parameters",
	}.Render(w))
}

// reporting gives a reporting failure the same shape everywhere.
func reporting(err error) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf("reporting: %w", err)
}
