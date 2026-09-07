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
// Package recipes finds and reports the curated knowledge on disk.
package recipes

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/retr0h/tonestack/internal/cli"
	"github.com/retr0h/tonestack/pkg/recipe"
	"github.com/retr0h/tonestack/pkg/recipe/gen"
	recipedata "github.com/retr0h/tonestack/recipes"
)

// DefaultDir is where recipes live.
const DefaultDir = "recipes"

// Load reads every recipe under dir, in identifier order.
//
// A file that does not satisfy the contract stops the walk: a half-read
// knowledge base is worse than a clear complaint about the file to fix.
func Load(dir string) ([]*gen.Recipe, error) {
	// No directory means the recipes that ship in the binary, which is the
	// case for anyone who has not written their own.
	if dir == "" {
		return loadFS(recipedata.FS, ".")
	}

	return loadFS(os.DirFS(dir), ".")
}

// loadFS reads every recipe under root, wherever that filesystem comes from.
func loadFS(fsys fs.FS, root string) ([]*gen.Recipe, error) {
	// The pattern is a constant, so it cannot be malformed.
	paths, _ := fs.Glob(fsys, path.Join(root, "*", "*.yaml"))

	out := make([]*gen.Recipe, 0, len(paths))

	for _, p := range paths {
		raw, err := fs.ReadFile(fsys, p)
		if err != nil {
			return nil, fmt.Errorf("opening %s: %w", path.Base(p), err)
		}

		r, err := decode(raw, p)
		if err != nil {
			return nil, err
		}

		out = append(out, r)
	}

	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })

	return out, nil
}

// loadOne reads a single recipe file.
// decode parses one recipe, naming the file it came from when it will not
// parse. A recipe is hand-written, so the name is the useful half of the
// message.
func decode(raw []byte, name string) (*gen.Recipe, error) {
	r, err := recipe.Load(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", filepath.Base(name), err)
	}

	return r, nil
}

// Find returns the recipe with the given identifier, or one of its aliases.
func Find(dir, id string) (*gen.Recipe, error) {
	all, err := Load(dir)
	if err != nil {
		return nil, err
	}

	for _, r := range all {
		if strings.EqualFold(r.ID, id) || matchesAlias(r, id) {
			return r, nil
		}
	}

	return nil, &NotFoundError{ID: id, Known: len(all)}
}

// matchesAlias reports whether id is one of the recipe's other names.
func matchesAlias(r *gen.Recipe, id string) bool {
	if r.Aliases == nil {
		return false
	}

	for _, a := range *r.Aliases {
		if strings.EqualFold(a, id) {
			return true
		}
	}

	return false
}

// List writes every recipe under dir to w.
func List(w io.Writer, dir string) error {
	all, err := Load(dir)
	if err != nil {
		return err
	}

	rows := make([][]string, 0, len(all))

	for _, r := range all {
		rows = append(rows, []string{
			cli.Accent(w, r.ID),
			r.Name,
			cli.Mute(w, string(r.InstrumentType)),
			r.Rig.Amp,
			source(w, r),
		})
	}

	return wrapReport(cli.Section{
		Title:   "Recipes",
		Detail:  dir,
		Headers: []string{"id", "name", "instrument", "amp", "source"},
		Rows:    rows,
		Empty:   "no recipes here",
	}.Render(w))
}

// source names where a recipe's knowledge came from, and marks it when
// nobody has confirmed it.
//
// This is the column that decides whether to trust the row, so it is the one
// that carries colour.
func source(w io.Writer, r *gen.Recipe) string {
	if recipe.Trusted(r.Provenance) {
		return cli.OK(w, string(r.Provenance.Source))
	}

	return cli.Info(w, string(r.Provenance.Source))
}

// Show writes one recipe to w in full.
func Show(w io.Writer, dir, id string) error {
	r, err := Find(dir, id)
	if err != nil {
		return err
	}

	d := cli.Detail{Title: r.Name, Subtitle: r.ID}

	if r.Band != nil && *r.Band != "" {
		d.Fields = append(d.Fields, cli.Field{Label: "band", Value: *r.Band})
	}

	d.Fields = append(d.Fields,
		cli.Field{Label: "instrument", Value: string(r.InstrumentType)},
		cli.Field{Label: "amp", Value: r.Rig.Amp},
	)

	if r.Rig.Cab != nil && *r.Rig.Cab != "" {
		d.Fields = append(d.Fields, cli.Field{Label: "cab", Value: *r.Rig.Cab})
	}

	if r.Rig.Technique != nil && *r.Rig.Technique != "" {
		d.Fields = append(d.Fields,
			cli.Field{Label: "technique", Value: *r.Rig.Technique})
	}

	d.Fields = append(d.Fields, character(r)...)
	d.Fields = append(d.Fields, variants(r)...)
	d.Fields = append(d.Fields, cli.Field{
		Label: "source",
		Value: fmt.Sprintf("%s, %s confidence",
			r.Provenance.Source, r.Provenance.Confidence),
	})

	if !recipe.Trusted(r.Provenance) {
		d.Note = "unverified — nobody has confirmed this gear"
	}

	return wrapReport(d.Render(w))
}

// writeCharacter renders how a rig should sound.
// character renders the intent lines, one per row, labelled only once.
//
// The label repeats as blank so the values line up in the same column as
// every other field rather than starting a block of their own.
func character(r *gen.Recipe) []cli.Field {
	if r.Character == nil || len(*r.Character) == 0 {
		return nil
	}

	out := make([]cli.Field, 0, len(*r.Character))

	for i, c := range *r.Character {
		label := ""
		if i == 0 {
			label = "character"
		}

		out = append(out, cli.Field{Label: label, Value: c})
	}

	return out
}

// writeVariants renders the per-song departures.
// variants renders the named departures from the characteristic rig.
func variants(r *gen.Recipe) []cli.Field {
	if r.Variants == nil || len(*r.Variants) == 0 {
		return nil
	}

	out := make([]cli.Field, 0, len(*r.Variants))

	for i, v := range *r.Variants {
		label := ""
		if i == 0 {
			label = "variants"
		}

		out = append(out, cli.Field{
			Label: label,
			Value: fmt.Sprintf("%s (%s)", v.Name, v.ID),
		})
	}

	return out
}

// wrapReport gives a reporting failure the same shape everywhere.
func wrapReport(err error) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf("reporting: %w", err)
}
