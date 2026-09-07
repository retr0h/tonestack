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
	"errors"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/retr0h/tonestack/pkg/catalog"
	"github.com/retr0h/tonestack/pkg/chain"
	"github.com/retr0h/tonestack/pkg/corpus"
	"github.com/retr0h/tonestack/pkg/preset"
)

// ErrNoPresets reports that a corpus directory held nothing to measure.
var ErrNoPresets = errors.New("no presets found")

// Measure reads every preset under a directory and reduces them to statistics.
//
// Presets written for other devices are measured too, and deliberately. Line 6
// ships one definition per model with a list of the devices carrying it, so an
// Ampeg SVT on a Helix Floor is the same model with the same controls as on an
// HX Stomp — how people set it is the same fact either way. The same holds for
// chain order, which is a musical habit rather than a property of the
// hardware.
//
// What is device-specific — how many blocks fit, how many signal paths there
// are — comes from the catalog, which already knows. Filtering by device here
// only shrinks the sample: restricting to one device left fourteen bass chains
// to learn from, against a hundred and sixty-nine across all of them.
//
// The filter that does apply is the model: anything the target catalog does
// not carry is skipped, because a statistic about a block this device lacks
// could never be acted on.
func Measure(opts Options, cat *catalog.Catalog) (*corpus.Stats, error) {
	paths, err := find(opts.CorpusDir)
	if err != nil {
		return nil, err
	}

	m := newMeasurer(cat)

	for _, path := range paths {
		doc, err := read(path)
		if err != nil {
			// One malformed preset among thousands is a fact about that file,
			// not a reason to abandon the measurement.
			continue
		}

		m.add(doc)
	}

	if m.presets == 0 {
		return nil, fmt.Errorf("%w in %s", ErrNoPresets, opts.CorpusDir)
	}

	return m.reduce(opts.MinSamples), nil
}

// find lists every preset file under a directory.
func find(dir string) ([]string, error) {
	var paths []string

	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() && strings.EqualFold(filepath.Ext(p), ".hlx") {
			paths = append(paths, p)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("searching %s: %w", dir, err)
	}

	return paths, nil
}

// read decodes one preset.
func read(path string) (*preset.Document, error) {
	f, err := os.Open(path) //nolint:gosec // a path this package walked
	if err != nil {
		return nil, err
	}

	defer func() { _ = f.Close() }()

	return preset.Read(f)
}

// measurer accumulates every observation before anything is averaged.
type measurer struct {
	cat     *catalog.Catalog
	presets int
	uses    map[catalog.ModelID]int
	values  map[catalog.ModelID]map[string]sample
	grammar map[string]map[catalog.Category]*counter
	chains  map[string]int
}

// newMeasurer returns a measurer ready to accumulate.
func newMeasurer(cat *catalog.Catalog) *measurer {
	return &measurer{
		cat:     cat,
		uses:    map[catalog.ModelID]int{},
		values:  map[catalog.ModelID]map[string]sample{},
		grammar: map[string]map[catalog.Category]*counter{},
		chains:  map[string]int{},
	}
}

// add folds one preset into the running totals.
func (m *measurer) add(doc *preset.Document) {
	spec, err := doc.Spec()
	if err != nil || len(spec.Blocks) == 0 {
		return
	}

	m.presets++

	for _, b := range spec.Blocks {
		if _, known := m.cat.Block(b.Model); !known {
			continue
		}

		m.uses[b.Model]++
		m.record(b.Model, b.Params)
	}

	m.grammarOf(spec)
}

// record keeps every numeric value a block was set to.
//
// Booleans and enumerated strings are skipped. A median over an enumeration
// is meaningless, and averaging a switch produces a value the device cannot
// accept.
func (m *measurer) record(id catalog.ModelID, params map[string]catalog.ParamValue) {
	for key, v := range params {
		var f float64

		switch {
		case v.Type() == catalog.ParamFloat:
			f, _ = v.Float()
		case v.Type() == catalog.ParamInt:
			i, _ := v.Int()
			f = float64(i)
		default:
			continue
		}

		if m.values[id] == nil {
			m.values[id] = map[string]sample{}
		}

		m.values[id][key] = append(m.values[id][key], f)
	}
}

// grammarOf records what a chain contained and where, for its instrument.
func (m *measurer) grammarOf(spec chain.Chain) {
	instrument, ampAt, ok := m.instrumentOf(spec)
	if !ok {
		return
	}

	m.chains[instrument]++

	if m.grammar[instrument] == nil {
		m.grammar[instrument] = map[catalog.Category]*counter{}
	}

	seen := map[catalog.Category]bool{}

	for i, b := range spec.Blocks {
		blk, known := m.cat.Block(b.Model)
		if !known || !tonal(blk.Category) {
			continue
		}

		c := m.grammar[instrument][blk.Category]
		if c == nil {
			c = &counter{}
			m.grammar[instrument][blk.Category] = c
		}

		if !seen[blk.Category] {
			c.chains++
			seen[blk.Category] = true
		}

		if i < ampAt {
			c.before++
		} else {
			c.after++
		}
	}
}

// tonal reports whether a category is something a player chooses for how it
// sounds.
//
// The amp is the thing everything else is positioned around, so it is not
// counted as a neighbour of itself. Plumbing is excluded because a volume
// block in 93% of chains says nothing about how anybody builds a tone.
func tonal(c catalog.Category) bool {
	return c != catalog.CategoryAmp && c != catalog.CategoryUtility
}

// instrumentOf finds the amp in a chain and reports which instrument it is
// for.
//
// A chain with no amp says nothing about ordering, because there is nothing to
// order around.
func (m *measurer) instrumentOf(spec chain.Chain) (string, int, bool) {
	for i, b := range spec.Blocks {
		blk, known := m.cat.Block(b.Model)
		if !known || blk.Category != catalog.CategoryAmp {
			continue
		}

		instrument := strings.ToLower(blk.Subcategory)
		if instrument != "guitar" && instrument != "bass" {
			return "", 0, false
		}

		return instrument, i, true
	}

	return "", 0, false
}

// reduce turns the accumulated observations into quartiles.
func (m *measurer) reduce(minSamples int) *corpus.Stats {
	out := &corpus.Stats{
		Device:   m.cat.Device,
		DeviceID: m.cat.DeviceID,
		Presets:  m.presets,
		Models:   map[catalog.ModelID]corpus.ModelStats{},
		Grammar:  map[string]corpus.Grammar{},
	}

	for id, uses := range m.uses {
		ms := corpus.ModelStats{Uses: uses, Params: map[string]corpus.ParamStats{}}

		for key, vals := range m.values[id] {
			if len(vals) < minSamples {
				continue
			}

			ms.Params[key] = quartiles(vals)
		}

		out.Models[id] = ms
	}

	for instrument, cats := range m.grammar {
		g := corpus.Grammar{
			Chains:     m.chains[instrument],
			Categories: map[catalog.Category]corpus.CategoryStats{},
		}

		for category, c := range cats {
			g.Categories[category] = corpus.CategoryStats{
				Chains: c.chains, Before: c.before, After: c.after,
			}
		}

		out.Grammar[instrument] = g
	}

	return out
}

// quartiles reduces a sample to the three figures worth keeping.
func quartiles(vals sample) corpus.ParamStats {
	sort.Float64s(vals)

	return corpus.ParamStats{
		N:      len(vals),
		Median: round(at(vals, 0.50)),
		P25:    round(at(vals, 0.25)),
		P75:    round(at(vals, 0.75)),
	}
}

// round trims a value to the precision a person would actually dial.
//
// Presets in the wild carry float32 artifacts written out in full — 0.74
// appears as 0.7400000095367432 — and a median lands on whichever literal it
// picked. Keeping that noise would put a value in a generated preset that
// differs from Line 6's own by a hundred-millionth and reads as though it
// were measured to that precision.
func round(v float64) float64 {
	return math.Round(v*precision) / precision
}

// precision is three decimal places, which is finer than any control Line 6
// states and far coarser than float32 noise.
const precision = 1000

// at returns the value at a quantile of a sorted sample.
//
// No bounds check: the quantiles asked for here top out at 0.75, so the index
// is always inside the sample. A guard against a case that cannot arise would
// be a branch no test can reach.
func at(sorted sample, q float64) float64 {
	return sorted[int(float64(len(sorted))*q)]
}
