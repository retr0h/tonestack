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

	"github.com/retr0h/tonestack/pkg/cli/internal/paint"

	"github.com/retr0h/tonestack/pkg/sdk"
	"github.com/retr0h/tonestack/pkg/sdk/corpus"
)

// Catalogued says what a catalog generation run produced.
//
// The gear count is the number worth reading: a catalog names every block a
// device has, and how many of those anybody can ask for by the name of the
// thing they emulate is the gap between a model list and knowledge.
func Catalogued(w io.Writer, r sdk.Catalogued) error {
	_, err := fmt.Fprintf(w,
		"wrote %s: %d blocks for %s from %s, %d mapped to real gear\n",
		r.Path, r.Blocks, r.Device, r.Source, r.Named)
	if err != nil {
		return fmt.Errorf("reporting: %w", err)
	}

	return nil
}

// Counted says what a corpus measuring run produced.
func Counted(w io.Writer, r sdk.Counted) error {
	var measured int

	for _, m := range r.Stats.Models {
		measured += len(m.Params)
	}

	if err := (paint.Section{
		Title:  r.Stats.Device,
		Detail: fmt.Sprintf("%d presets measured", r.Stats.Presets),
		Headers: []string{
			"instrument", "chains", "category", "in chain", "before amp",
		},
		Rows:  grammarRows(w, r.Stats),
		Empty: "no chains held an amp, so nothing could be ordered",
		Summary: fmt.Sprintf("%d models, %d parameter distributions, wrote %s",
			len(r.Stats.Models), measured, r.Path),
	}).Render(w); err != nil {
		return fmt.Errorf("reporting: %w", err)
	}

	return nil
}

// grammarRows draws what the measurements say about chain shape.
func grammarRows(w io.Writer, stats *corpus.Stats) [][]string {
	var rows [][]string

	for _, instrument := range sortedGrammar(stats.Grammar) {
		g := stats.Grammar[instrument]

		for _, category := range sortedCategories(g.Categories) {
			c := g.Categories[category]

			rows = append(rows, []string{
				paint.Accent(w, instrument),
				fmt.Sprintf("%d", g.Chains),
				paint.Category(w, category),
				fmt.Sprintf("%.0f%%", c.Frequency(g.Chains)*100),
				fmt.Sprintf("%.0f%%", c.BeforeAmp()*100),
			})
		}
	}

	return rows
}
