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
//
// A recipe is a RigSpec that ships with the project. There is no separate
// recipe format: what a person writes by hand, what a preset lifts to, and
// what compiles back down are all the same document.
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
	"github.com/retr0h/tonestack/pkg/rig"
	"github.com/retr0h/tonestack/pkg/rig/gen"
	recipedata "github.com/retr0h/tonestack/recipes"
)

// DefaultDir is where recipes live.
const DefaultDir = "recipes"

// Load reads every rig under dir, in identifier order.
//
// A file that does not satisfy the contract stops the walk: a half-read
// knowledge base is worse than a clear complaint about the file to fix.
func Load(dir string) ([]gen.RigSpec, error) {
	// No directory means the recipes that ship in the binary, which is the
	// case for anyone who has not written their own.
	if dir == "" {
		return loadFS(recipedata.FS, ".")
	}

	return loadFS(os.DirFS(dir), ".")
}

// loadFS reads every rig under root, wherever that filesystem comes from.
func loadFS(fsys fs.FS, root string) ([]gen.RigSpec, error) {
	// The pattern is a constant, so it cannot be malformed.
	paths, _ := fs.Glob(fsys, path.Join(root, "*", "*.yaml"))

	out := make([]gen.RigSpec, 0, len(paths))

	for _, p := range paths {
		raw, err := fs.ReadFile(fsys, p)
		if err != nil {
			return nil, fmt.Errorf("opening %s: %w", path.Base(p), err)
		}

		spec, err := decode(raw, p)
		if err != nil {
			return nil, err
		}

		out = append(out, spec)
	}

	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })

	return out, nil
}

// decode parses one rig, naming the file it came from when it will not parse.
// A recipe is hand-written, so the name is the useful half of the message.
func decode(raw []byte, name string) (gen.RigSpec, error) {
	spec, err := rig.Load(bytes.NewReader(raw))
	if err != nil {
		return gen.RigSpec{}, fmt.Errorf("%s: %w", filepath.Base(name), err)
	}

	return spec, nil
}

// Find returns the rig with the given identifier, or one of its aliases.
func Find(dir, id string) (gen.RigSpec, error) {
	all, err := Load(dir)
	if err != nil {
		return gen.RigSpec{}, err
	}

	return find(all, id)
}

// find picks one rig out of a set already read.
func find(all []gen.RigSpec, id string) (gen.RigSpec, error) {
	for _, spec := range all {
		if strings.EqualFold(spec.ID, id) || matchesAlias(spec, id) {
			return spec, nil
		}
	}

	return gen.RigSpec{}, &NotFoundError{ID: id, Known: len(all)}
}

// matchesAlias reports whether id is one of the rig's other names.
func matchesAlias(spec gen.RigSpec, id string) bool {
	if spec.Aliases == nil {
		return false
	}

	for _, a := range *spec.Aliases {
		if strings.EqualFold(a, id) {
			return true
		}
	}

	return false
}

// departures names the rigs that are a small change on this one.
//
// A player owns several rigs — by era, by song — and they are siblings rather
// than deltas, because two of them can differ at the amp and a delta assumes
// a spine they may not share. A rig that genuinely is a small change says so
// with `extends`, and this is the other end of that link: reading the
// characteristic rig should show what departs from it.
func departures(all []gen.RigSpec, spec gen.RigSpec) []cli.Field {
	out := []cli.Field(nil)

	for _, other := range all {
		if other.Extends == nil || *other.Extends != spec.ID {
			continue
		}

		label := ""
		if len(out) == 0 {
			label = "variants"
		}

		out = append(out, cli.Field{
			Label: label,
			Value: fmt.Sprintf("%s (%s)", other.Subject.Name, other.ID),
		})
	}

	return out
}

// List writes every rig under dir to w.
func List(w io.Writer, dir string) error {
	all, err := Load(dir)
	if err != nil {
		return err
	}

	rows := make([][]string, 0, len(all))

	for _, spec := range all {
		rows = append(rows, []string{
			cli.Accent(w, spec.ID),
			spec.Subject.Name,
			cli.Mute(w, string(spec.Instrument)),
			rig.GearName(spec, gen.RoleAmp),
			source(w, spec),
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

// source names where a rig's knowledge came from, and marks it when nobody
// has confirmed it.
//
// This is the column that decides whether to trust the row, so it is the one
// that carries colour. A rig is only shown as confirmed when every claim in
// it rests on something checkable — the weakest link is what the reader needs
// to know about.
func source(w io.Writer, spec gen.RigSpec) string {
	if rig.Trusted(spec) {
		return cli.OK(w, string(rig.Sourced(spec)))
	}

	return cli.Info(w, string(rig.Sourced(spec)))
}

// Show writes one rig to w in full.
func Show(w io.Writer, dir, id string) error {
	all, err := Load(dir)
	if err != nil {
		return err
	}

	spec, err := find(all, id)
	if err != nil {
		return err
	}

	d := cli.Detail{Title: spec.Subject.Name, Subtitle: spec.ID}

	if spec.Subject.Band != nil && *spec.Subject.Band != "" {
		d.Fields = append(d.Fields,
			cli.Field{Label: "band", Value: *spec.Subject.Band})
	}

	if spec.Subject.Era != nil && *spec.Subject.Era != "" {
		d.Fields = append(d.Fields,
			cli.Field{Label: "era", Value: *spec.Subject.Era})
	}

	d.Fields = append(d.Fields,
		cli.Field{Label: "instrument", Value: string(spec.Instrument)})
	d.Fields = append(d.Fields, chain(spec)...)

	if spec.Technique != nil && *spec.Technique != "" {
		d.Fields = append(d.Fields,
			cli.Field{Label: "technique", Value: *spec.Technique})
	}

	d.Fields = append(d.Fields, character(spec)...)
	d.Fields = append(d.Fields, departures(all, spec)...)
	d.Fields = append(d.Fields, cli.Field{
		Label: "source",
		Value: fmt.Sprintf("%s, %s confidence",
			rig.Sourced(spec), confidence(spec)),
	})

	if !rig.Trusted(spec) {
		d.Note = "unverified — nobody has confirmed this gear"
	}

	return wrapReport(d.Render(w))
}

// chain renders the signal path, in order, one row per piece of gear.
//
// Labelled by role rather than by position, because "amp" is what a person
// reading this wants to find and "3" is not.
func chain(spec gen.RigSpec) []cli.Field {
	out := make([]cli.Field, 0, len(spec.Chain))

	for _, e := range spec.Chain {
		out = append(out, cli.Field{Label: string(e.Role), Value: e.Gear})
	}

	return out
}

// confidence reports how far a rig says it should be trusted.
func confidence(spec gen.RigSpec) gen.Confidence {
	if spec.Confidence == nil {
		return gen.ConfidenceLow
	}

	return *spec.Confidence
}

// character renders the intent lines, one per row, labelled only once.
//
// The label repeats as blank so the values line up in the same column as
// every other field rather than starting a block of their own.
func character(spec gen.RigSpec) []cli.Field {
	if spec.Character == nil || len(*spec.Character) == 0 {
		return nil
	}

	out := make([]cli.Field, 0, len(*spec.Character))

	for i, c := range *spec.Character {
		label := ""
		if i == 0 {
			label = "character"
		}

		out = append(out, cli.Field{Label: label, Value: c})
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
