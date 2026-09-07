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
	"io"
	"os"

	"github.com/retr0h/tonestack/internal/catalogview"
	"github.com/retr0h/tonestack/internal/cli"
	"github.com/retr0h/tonestack/internal/recipes"
	"github.com/retr0h/tonestack/internal/resolve"
	"github.com/retr0h/tonestack/pkg/catalog"
	"github.com/retr0h/tonestack/pkg/chain"
	"github.com/retr0h/tonestack/pkg/corpus"
	"github.com/retr0h/tonestack/pkg/preset"
)

// MakeOptions says what to build and where to put it.
type MakeOptions struct {
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
func Make(w io.Writer, opts MakeOptions) error {
	rec, err := recipes.Find(opts.RecipesDir, opts.RecipeID)
	if err != nil {
		return err
	}

	cat, err := catalogview.Open(opts.CatalogPath)
	if err != nil {
		return err
	}

	// Statistics are an improvement on the catalog's defaults, not a
	// requirement. Without them a preset still loads, it is just more
	// generic, so a failure to read them is not a failure to build.
	stats, _ := openStats(opts.StatsPath)

	spec, added, err := resolve.Resolve(rec, cat, stats)
	if err != nil {
		return err
	}

	limits := chain.HXStompLimits()
	spec = resolve.Fit(spec, cat, limits)

	if err := chain.Validate(cat, spec, limits); err != nil {
		return fmt.Errorf("the chain this recipe describes will not load: %w", err)
	}

	doc := build(cat.DeviceID, spec)

	if err := write(opts.OutputPath, doc); err != nil {
		return err
	}

	return report(w, spec, cat, added, opts.OutputPath)
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

// report writes the chain that was chosen, and what it cost.
//
// Make passes through several stages, so a failure here says which one it
// was rather than surfacing a bare write error.
func report(
	w io.Writer,
	spec chain.Chain,
	cat *catalog.Catalog,
	added []resolve.Added,
	path string,
) error {
	if err := render(w, spec, cat, added, path); err != nil {
		return fmt.Errorf("reporting: %w", err)
	}

	return nil
}

// render prints the chain, using the same renderer that prints a slot read
// off the device, so a generated preset and one somebody made by hand are
// read the same way.
func render(
	w io.Writer,
	spec chain.Chain,
	cat *catalog.Catalog,
	added []resolve.Added,
	path string,
) error {
	if _, err := fmt.Fprintf(
		w, "\n%s%s\n\n", cli.Indent, cli.Title(w, spec.Name),
	); err != nil {
		return err
	}

	if err := cli.Chain(w, spec, cat); err != nil {
		return err
	}

	if err := explain(w, added); err != nil {
		return err
	}

	_, err := fmt.Fprintf(w, "\n%s%s\n\n", cli.Indent, cli.Success(w, "wrote "+path))

	return err
}

// explain names the blocks nobody asked for, and why they are there.
//
// A recipe names an amp; a rig is four or five blocks. The rest come from what
// the corpus shows chains of this kind almost always hold, and a choice made
// on the player's behalf has to be visible before they plug in.
func explain(w io.Writer, added []resolve.Added) error {
	if len(added) == 0 {
		return nil
	}

	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}

	for _, a := range added {
		// A block drawn from the corpus can say how common it is. One
		// substituted for gear no model emulates cannot, and appending "0% of
		// chains" to it would read as a measurement.
		line := fmt.Sprintf("%s — %s", a.Block.Name, a.Reason)
		if a.Share > 0 {
			line += fmt.Sprintf(" (%.0f%% of chains)", a.Share*100)
		}

		_, err := fmt.Fprintf(w, "%s%s %s\n", cli.Indent, cli.Mute(w, "added"), line)
		if err != nil {
			return err
		}
	}

	return nil
}
