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

package corpusgen

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/retr0h/tonestack/internal/catalogview"
	"github.com/retr0h/tonestack/internal/cli"
	"github.com/retr0h/tonestack/pkg/catalog"
	"github.com/retr0h/tonestack/pkg/corpus"
)

// defaultMinSamples is how many values a parameter needs before its
// distribution is worth recording.
const defaultMinSamples = 5

// Run measures a corpus and writes the statistics, reporting what it found.
func Run(w io.Writer, opts Options) error {
	if opts.MinSamples == 0 {
		opts.MinSamples = defaultMinSamples
	}

	cat, err := catalogview.Open(opts.CatalogPath)
	if err != nil {
		return err
	}

	stats, err := Measure(opts, cat)
	if err != nil {
		return err
	}

	// Statistics hold only numbers and strings, so encoding cannot fail.
	raw, _ := json.Marshal(stats)

	if err := os.WriteFile(opts.OutputPath, compress(raw), 0o600); err != nil {
		return fmt.Errorf("writing %s: %w", opts.OutputPath, err)
	}

	return report(w, stats, cat, opts.OutputPath)
}

// compress gzips the statistics, which are repetitive JSON and embedded in
// the binary.
func compress(raw []byte) []byte {
	var buf bytes.Buffer

	// Compressing into a buffer cannot fail.
	zw := gzip.NewWriter(&buf)
	_, _ = zw.Write(raw)
	_ = zw.Close()

	return buf.Bytes()
}

// report says what the corpus turned out to contain.
func report(
	w io.Writer,
	stats *corpus.Stats,
	cat *catalog.Catalog,
	path string,
) error {
	var measured int

	for _, m := range stats.Models {
		measured += len(m.Params)
	}

	if err := (cli.Section{
		Title:  stats.Device,
		Detail: fmt.Sprintf("%d presets measured", stats.Presets),
		Headers: []string{
			"instrument", "chains", "category", "in chain", "before amp",
		},
		Rows:  grammarRows(w, stats, cat),
		Empty: "no chains held an amp, so nothing could be ordered",
		Summary: fmt.Sprintf("%d models, %d parameter distributions, wrote %s",
			len(stats.Models), measured, path),
	}).Render(w); err != nil {
		return fmt.Errorf("reporting: %w", err)
	}

	return nil
}

// grammarRows renders what each instrument's chains tend to hold.
func grammarRows(
	w io.Writer,
	stats *corpus.Stats,
	_ *catalog.Catalog,
) [][]string {
	var rows [][]string

	for _, instrument := range sortedKeys(stats.Grammar) {
		g := stats.Grammar[instrument]

		for _, category := range sortedCategories(g.Categories) {
			c := g.Categories[category]

			rows = append(rows, []string{
				cli.Accent(w, instrument),
				fmt.Sprintf("%d", g.Chains),
				cli.Category(w, category),
				fmt.Sprintf("%.0f%%", c.Frequency(g.Chains)*100),
				fmt.Sprintf("%.0f%%", c.BeforeAmp()*100),
			})
		}
	}

	return rows
}

// sortedKeys returns map keys in a stable order, so a report does not shuffle
// between runs.
func sortedKeys(m map[string]corpus.Grammar) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}

	sort.Strings(out)

	return out
}

// sortedCategories orders categories by how often they appear, so the most
// telling rows come first.
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
