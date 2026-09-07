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
// Package catalogview reports what a device can do.
package catalogview

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/retr0h/tonestack/internal/cli"
	"github.com/retr0h/tonestack/pkg/catalog"
)

// DefaultPath is where the generated catalog lives.
const DefaultPath = "schemas/hx-stomp.catalog.json"

// Filter narrows what List reports.
type Filter struct {
	// Category keeps only blocks of one kind — amp, cab, drive.
	Category string
	// Subcategory keeps only blocks Line 6 tags this way — Guitar, Bass.
	Subcategory string
	// Search keeps only blocks whose name or real-world gear mentions this.
	Search string
}

// Open reads the catalog at path.
func Open(path string) (*catalog.Catalog, error) {
	// No path means the catalog that ships in the binary, which is the case
	// for anyone who has not generated their own.
	if path == "" {
		return catalog.BuiltIn()
	}

	f, err := os.Open(path) //nolint:gosec // a path the caller named
	if err != nil {
		return nil, fmt.Errorf("opening catalog: %w", err)
	}

	defer func() { _ = f.Close() }()

	c, err := catalog.Load(f)
	if err != nil {
		return nil, err
	}

	return c, nil
}

// List writes the blocks matching f to w.
func List(w io.Writer, path string, f Filter) error {
	c, err := Open(path)
	if err != nil {
		return err
	}

	matched := match(c, f)
	rows := make([][]string, 0, len(matched))

	for _, b := range matched {
		rows = append(rows, []string{
			cli.Accent(w, string(b.ID)),
			b.Name,
			cli.Category(w, b.Category),
			cli.Mute(w, b.Subcategory),
			fmt.Sprintf("%.1f", b.DSP.Mono),
			cli.Mute(w, b.BasedOn),
		})
	}

	return report(cli.Section{
		Title:   c.Device,
		Detail:  fmt.Sprintf("%d blocks · %s", len(c.Blocks), origin(c)),
		Headers: []string{"model", "name", "category", "sub", "dsp", "based on"},
		Rows:    rows,
		Align: []lipgloss.Position{
			lipgloss.Left, lipgloss.Left, lipgloss.Left,
			lipgloss.Left, lipgloss.Right,
		},
		Empty:   "no blocks match",
		Summary: fmt.Sprintf("%d of %d blocks", len(matched), len(c.Blocks)),
	}.Render(w))
}

// match returns the blocks satisfying f, in identifier order.
// origin says which release a catalog was generated from.
//
// A catalog is only true of the models that release knew about, so one that
// cannot name its source is worth flagging rather than presenting as fact.
func origin(c *catalog.Catalog) string {
	if c.Source == "" {
		return "source unknown"
	}

	return c.Source
}

func match(c *catalog.Catalog, f Filter) []catalog.Block {
	out := make([]catalog.Block, 0, len(c.Blocks))

	for _, b := range c.Blocks {
		if f.Category != "" && !strings.EqualFold(string(b.Category), f.Category) {
			continue
		}

		if f.Subcategory != "" && !strings.EqualFold(b.Subcategory, f.Subcategory) {
			continue
		}

		if f.Search != "" && !mentions(b, f.Search) {
			continue
		}

		out = append(out, b)
	}

	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })

	return out
}

// mentions reports whether a block's name or real-world gear contains term.
func mentions(b catalog.Block, term string) bool {
	term = strings.ToLower(term)

	return strings.Contains(strings.ToLower(b.Name), term) ||
		strings.Contains(strings.ToLower(b.BasedOn), term) ||
		strings.Contains(strings.ToLower(string(b.ID)), term)
}

// Show writes one block's parameters to w.
func Show(w io.Writer, path, id string) error {
	c, err := Open(path)
	if err != nil {
		return err
	}

	b, ok := c.Block(catalog.ModelID(id))
	if !ok {
		return &NotFoundError{ID: id, Known: len(c.Blocks)}
	}

	d := cli.Detail{Title: b.Name, Subtitle: string(b.ID)}

	if b.BasedOn != "" {
		d.Fields = append(d.Fields, cli.Field{Label: "based on", Value: b.BasedOn})
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
		cli.Field{Label: "category", Value: category},
		cli.Field{Label: "dsp", Value: dsp},
	)

	if !b.Prov.Trusted() {
		d.Note = "assumed — this figure was inferred, not stated by Line 6"
	}

	if err := d.Render(w); err != nil {
		return report(err)
	}

	return writeParams(w, b)
}

// writeParams renders a block's knobs, in name order.
func writeParams(w io.Writer, b catalog.Block) error {
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
			cli.Accent(w, k),
			cli.Mute(w, string(p.Type)),
			cli.Mute(w, rng),
			string(def),
		})
	}

	return report(cli.Section{
		Headers: []string{"parameter", "kind", "range", "default"},
		Rows:    rows,
		Empty:   "no parameters",
	}.Render(w))
}

// report gives a reporting failure the same shape everywhere.
func report(err error) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf("reporting: %w", err)
}
