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
	"github.com/retr0h/tonestack/pkg/sdk/corpus"
)

// Measured prints what the corpus says.
//
// Which of the two questions was asked decides what there is to draw, and the
// answer already says which.
func Measured(w io.Writer, m sdk.Measured) error {
	if m.AboutOne() {
		return model(w, m.Stats, m.Catalog, m.Model)
	}

	return grammar(w, m.Stats, m.Instrument)
}

// model prints how one model is set across every preset that used it.
func model(
	w io.Writer,
	stats *corpus.Stats,
	cat *catalog.Catalog,
	id catalog.ModelID,
) error {
	// The operation already refused a model nobody measured, so this is
	// drawing what is there rather than deciding whether there is any.
	ms := stats.Models[id]
	blk, known := cat.Block(id)

	name := string(id)
	if known {
		name = blk.Name
	}

	rows := make([][]string, 0, len(ms.Params))

	for _, key := range sortedParams(ms.Params) {
		p := ms.Params[key]
		def := "—"

		if known {
			if cp, ok := blk.Params[key]; ok {
				def = cp.Default.String()
			}
		}

		rows = append(rows, []string{
			paint.Accent(w, key),
			fmt.Sprintf("%d", p.N),
			paint.Mute(w, def),
			fmt.Sprintf("%.3f", p.Median),
			fmt.Sprintf("%.3f", p.Spread()),
			agreement(w, p.Spread(), span(blk, key, known)),
		})
	}

	return (paint.Section{
		Title:  name,
		Detail: fmt.Sprintf("%d uses across %d presets", ms.Uses, stats.Presets),
		Headers: []string{
			"parameter", "n", "line 6", "median", "spread", "agreement",
		},
		Rows:    rows,
		Align:   []lipgloss.Position{lipgloss.Left, lipgloss.Right},
		Empty:   "no parameter was seen often enough to measure",
		Summary: "a narrow spread means players agree; a wide one means taste",
	}).Render(w)
}

// span is a parameter's range, used to judge a spread against it.
func span(blk catalog.Block, key string, known bool) float64 {
	if !known {
		return 0
	}

	p, ok := blk.Params[key]
	if !ok {
		return 0
	}

	return p.Max - p.Min
}

// agreement renders how tightly players agree, relative to the range the
// parameter can occupy.
func agreement(w io.Writer, spread, span float64) string {
	if span <= 0 {
		return paint.Mute(w, "—")
	}

	switch r := spread / span; {
	case r <= 0.05:
		return paint.OK(w, "unanimous")
	case r <= 0.15:
		return paint.OK(w, "close")
	case r <= 0.35:
		return paint.Info(w, "loose")
	default:
		return paint.Err(w, "none")
	}
}

// grammar prints what chains tend to contain, per instrument.
func grammar(w io.Writer, stats *corpus.Stats, only string) error {
	var rows [][]string

	for _, instrument := range sortedGrammar(stats.Grammar) {
		if only != "" && instrument != only {
			continue
		}

		g := stats.Grammar[instrument]

		for _, c := range sortedCategories(g.Categories) {
			s := g.Categories[c]

			rows = append(rows, []string{
				paint.Accent(w, instrument),
				paint.Category(w, c),
				fmt.Sprintf("%.0f%%", s.Frequency(g.Chains)*100),
				fmt.Sprintf("%.0f%%", s.BeforeAmp()*100),
				paint.Mute(w, fmt.Sprintf("%d chains", g.Chains)),
			})
		}
	}

	return (paint.Section{
		Title:   "Chain grammar",
		Detail:  fmt.Sprintf("%d presets measured", stats.Presets),
		Headers: []string{"instrument", "category", "in chain", "before amp", ""},
		Rows:    rows,
		Empty:   "nothing measured for that instrument",
		Summary: "position is not decoration: drive before an amp overdrives " +
			"its input, drive after it does something else",
	}).Render(w)
}

// sortedParams orders parameter keys so a listing is stable.
func sortedParams(m map[string]corpus.ParamStats) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}

	sort.Strings(out)

	return out
}

// sortedGrammar orders instruments so a listing is stable.
func sortedGrammar(m map[string]corpus.Grammar) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}

	sort.Strings(out)

	return out
}

// sortedCategories orders categories by how often they appear.
func sortedCategories(
	m map[catalog.Category]corpus.CategoryStats,
) []catalog.Category {
	out := make([]catalog.Category, 0, len(m))
	for k := range m {
		out = append(out, k)
	}

	sort.Slice(out, func(i, j int) bool {
		if m[out[i]].Chains != m[out[j]].Chains {
			return m[out[i]].Chains > m[out[j]].Chains
		}

		return out[i] < out[j]
	})

	return out
}
