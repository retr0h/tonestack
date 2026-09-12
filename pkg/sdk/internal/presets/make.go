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
// Package presets turns curated knowledge into a preset file.
package presets

import (
	"bytes"
	"fmt"
	"os"

	"github.com/retr0h/tonestack/pkg/sdk/chain"
	"github.com/retr0h/tonestack/pkg/sdk/corpus"
	"github.com/retr0h/tonestack/pkg/sdk/internal/compile"
	"github.com/retr0h/tonestack/pkg/sdk/preset"
	"github.com/retr0h/tonestack/pkg/sdk/result"
	"github.com/retr0h/tonestack/pkg/sdk/rig"
)

// MakeOptions says what to build and where to put it.
type MakeOptions struct {
	// Deps are the collaborators this command works through.
	Deps

	// RecipeID names the curated knowledge to build from.
	RecipeID string
	// RecipesDir is where recipes live.
	RecipesDir string
	// CatalogPath is the generated catalog for the target device. Empty
	// means the one built into this binary.
	CatalogPath string
	// StatsPath is measured corpus statistics. Empty means the ones built
	// into this binary.
	StatsPath string
	// OutputPath is where the preset is written.
	OutputPath string
}

// Make builds a preset from a recipe and writes it, reporting what it chose.
//
// Reporting the chain matters as much as writing the file. A generated preset
// is a set of decisions, and a wrong amp should be visible before anyone plugs
// in rather than after.
func Make(opts MakeOptions) (result.Made, error) {
	rec, err := opts.recipes().Find(opts.RecipesDir, opts.RecipeID)
	if err != nil {
		return result.Made{}, err
	}

	cat, err := opts.catalogs().Open(opts.CatalogPath)
	if err != nil {
		return result.Made{}, err
	}

	// Statistics are an improvement on the catalog's defaults, not a
	// requirement. Without them a preset still loads, it is just more
	// generic, so a failure to read them is not a failure to build.
	stats, _ := openStats(opts.StatsPath)

	spec, added, moved, err := opts.compiler().Resolve(rec, cat, stats)
	if err != nil {
		return result.Made{}, err
	}

	limits := chain.HXStompLimits()
	spec = opts.compiler().Fit(spec, cat, limits)

	if err := chain.Validate(cat, spec, limits); err != nil {
		return result.Made{}, fmt.Errorf(
			"the chain this recipe describes will not load: %w", err)
	}

	doc := build(cat.DeviceID, spec)

	if err := write(opts.OutputPath, doc); err != nil {
		return result.Made{}, err
	}

	return result.Made{
		Chain:      spec,
		Added:      addedFrom(added),
		Moved:      movedFrom(moved),
		Unfamiliar: unfamiliar(rec),
		Path:       opts.OutputPath,
	}, nil
}

// openStats reads measured statistics, falling back to the ones in this
// binary.
func openStats(path string) (*corpus.Stats, error) {
	if path == "" {
		return corpus.BuiltIn()
	}

	f, err := os.Open(path) //nolint:gosec // the path is the user's own file
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", path, err)
	}

	defer func() { _ = f.Close() }()

	return corpus.Load(f)
}

// build puts a chain into a preset the device would recognise.
//
// Written into an untouched preset rather than assembled from nothing: a
// device expects inputs, outputs, a split and a join around a chain, and
// 98.6% of real presets carry them. One built without them is unlike anything
// the hardware has ever written.
func build(deviceID int, spec chain.Chain) *preset.Document {
	// The blank is embedded and covered by its own test, so reading it cannot
	// fail here. SetSpec refuses a parameter named like a block attribute,
	// and a resolved chain cannot hold one because the catalog excludes them.
	doc, _ := preset.Blank()
	doc.Data.Device = deviceID
	_ = doc.SetSpec(spec)

	return doc
}

// write puts the preset on disk.
//
// The document is rendered to memory first so there is one failure to report —
// the write — rather than three, two of which no test can reach.
func write(path string, doc *preset.Document) error {
	var buf bytes.Buffer

	// Writing to a buffer cannot fail, and a document this package built
	// always encodes.
	_ = preset.Write(&buf, doc)

	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}

	return nil
}

// unfamiliar names the character terms nothing defines.
//
// Said rather than refused. A term moves no knob, so an unfamiliar one costs
// the preset nothing, and a build that stopped over a word would be refusing
// somebody the right to describe a sound in their own words. The rigs this
// project ships are held to the list by a test instead.
func unfamiliar(rec rig.Spec) []result.Unfamiliar {
	unknown := compile.CheckCharacter(rec)

	out := make([]result.Unfamiliar, 0, len(unknown))
	for _, u := range unknown {
		out = append(out, result.Unfamiliar{Term: u.Term, Near: u.Near})
	}

	return out
}

// addedFrom says what went into the chain that the recipe did not name.
func addedFrom(added []compile.Added) []result.Added {
	out := make([]result.Added, 0, len(added))
	for _, a := range added {
		out = append(out, result.Added{
			Name:   a.Block.Name,
			Reason: a.Reason,
			Share:  a.Share,
		})
	}

	return out
}

// movedFrom says which words turned which knobs.
func movedFrom(moved []compile.Moved) []result.Moved {
	out := make([]result.Moved, 0, len(moved))
	for _, m := range moved {
		out = append(out, result.Moved{
			Term:    m.Term,
			Param:   m.Param,
			From:    m.From,
			To:      m.To,
			Against: m.Against,
			Because: m.Because,
			Already: m.Already,
		})
	}

	return out
}
