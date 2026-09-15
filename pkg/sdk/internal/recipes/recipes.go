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
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"

	"github.com/retr0h/tonestack/pkg/sdk/result"
	"github.com/retr0h/tonestack/pkg/sdk/rig"
	"github.com/retr0h/tonestack/pkg/sdk/rigs"
)

// DefaultDir is where recipes live.
const DefaultDir = "pkg/sdk/rigs"

// Load reads every rig under dir, in identifier order.
//
// A file that does not satisfy the contract stops the walk: a half-read
// knowledge base is worse than a clear complaint about the file to fix.
func Load(
	dir string,
) ([]rig.Spec, error) {
	all, err := readBase(dir)
	if err != nil {
		return nil, err
	}

	return specs(all), nil
}

// readBase reads the rigs a layer sits on, refusing any file that is not a
// rig.
func readBase(
	dir string,
) ([]stored, error) {
	// No directory means the recipes that ship in the binary, which is the
	// case for anyone who has not written their own.
	fsys, name := fs.FS(rigs.FS), "the built-in recipes"
	if dir != "" {
		fsys, name = os.DirFS(dir), dir
	}

	all, broken, err := readFS(fsys, name)
	if err != nil {
		return nil, err
	}

	if len(broken) > 0 {
		return nil, broken[0].err
	}

	return all, nil
}

// readFS reads every rig under the root of fsys, wherever that filesystem
// comes from. name says where that is, for a directory that cannot be read.
//
// A file that will not open or will not decode is handed back rather than
// stopping the walk, so the caller decides what one costs.
func readFS(
	fsys fs.FS,
	name string,
) ([]stored, []brokenFile, error) {
	// Glob drops a directory it cannot read, which would make one nobody may
	// open look like one holding no recipes. A directory that is not there is
	// different: nobody has written a recipe into it yet.
	if _, err := fs.ReadDir(fsys, "."); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, nil, fmt.Errorf("reading %s: %w", name, err)
	}

	// The pattern is a constant, so it cannot be malformed.
	paths, _ := fs.Glob(fsys, path.Join("*", "*.yaml"))

	out := make([]stored, 0, len(paths))
	broken := []brokenFile(nil)

	for _, p := range paths {
		raw, err := fs.ReadFile(fsys, p)
		if err != nil {
			broken = append(broken, brokenFile{
				names: claimed(p, nil),
				err:   fmt.Errorf("opening %s: %w", path.Base(p), err),
			})

			continue
		}

		spec, err := decode(raw, p)
		if err != nil {
			broken = append(broken, brokenFile{names: claimed(p, raw), err: err})

			continue
		}

		out = append(out, stored{spec: spec, raw: raw})
	}

	sortEntries(out)

	return out, broken, nil
}

// decode parses one rig, naming the file it came from when it will not parse.
// A recipe is hand-written, so the name is the useful half of the message.
func decode(
	raw []byte,
	name string,
) (rig.Spec, error) {
	spec, err := rig.Load(bytes.NewReader(raw))
	if err != nil {
		return rig.Spec{}, fmt.Errorf("%s: %w", filepath.Base(name), err)
	}

	return spec, nil
}

// read reads every rig a Source holds.
func read(
	src Source,
) (set, error) {
	base, err := readBase(src.Dir)
	if err != nil {
		return set{}, err
	}

	if src.User == "" {
		return set{base: base}, nil
	}

	user, broken, err := readFS(os.DirFS(src.User), src.User)
	if err != nil {
		return set{}, err
	}

	return set{user: user, base: base, broken: broken}, nil
}

// List reads every rig a Source holds.
//
// A file of somebody's own that is not a rig is reported here, every one of
// them, because a listing is where somebody looks for what they wrote.
func List(
	src Source,
) (result.Recipes, error) {
	all, err := read(src)
	if err != nil {
		return result.Recipes{}, err
	}

	if len(all.broken) > 0 {
		errs := make([]error, 0, len(all.broken))
		for _, b := range all.broken {
			errs = append(errs, b.err)
		}

		return result.Recipes{}, errors.Join(errs...)
	}

	dir := src.Dir
	if src.User != "" {
		dir = src.User
	}

	return result.Recipes{Dir: dir, Rigs: specs(all.merged())}, nil
}

// Find returns the rig with the given identifier, or one of its aliases.
func Find(
	src Source,
	id string,
) (rig.Spec, error) {
	all, err := read(src)
	if err != nil {
		return rig.Spec{}, err
	}

	found, err := all.find(id)
	if err != nil {
		return rig.Spec{}, err
	}

	return found.spec, nil
}

// Show reads one rig, and what the rest of the set says about it.
func Show(
	src Source,
	id string,
) (result.Recipe, error) {
	all, err := read(src)
	if err != nil {
		return result.Recipe{}, err
	}

	found, err := all.find(id)
	if err != nil {
		return result.Recipe{}, err
	}

	return result.Recipe{
		Rig:      found.spec,
		Variants: departures(specs(all.merged()), found.spec),
	}, nil
}

// departures names the rigs that are a small change on this one.
//
// A player owns several rigs — by era, by song — and they are siblings rather
// than deltas, because two of them can differ at the amp and a delta assumes
// a spine they may not share. A rig that genuinely is a small change says so
// with `extends`, and this is the other end of that link: reading the
// characteristic rig should show what departs from it.
func departures(
	all []rig.Spec,
	spec rig.Spec,
) []result.Variant {
	out := []result.Variant(nil)

	for _, other := range all {
		if other.Extends == nil || *other.Extends != spec.ID {
			continue
		}

		out = append(out, result.Variant{ID: other.ID, Name: other.Subject.Name})
	}

	return out
}

// specs are the rigs of a set of entries, in the same order.
func specs(
	all []stored,
) []rig.Spec {
	out := make([]rig.Spec, 0, len(all))
	for _, e := range all {
		out = append(out, e.spec)
	}

	return out
}

// sortEntries puts entries in identifier order.
func sortEntries(
	all []stored,
) {
	sort.SliceStable(all, func(i, j int) bool { return all[i].spec.ID < all[j].spec.ID })
}
