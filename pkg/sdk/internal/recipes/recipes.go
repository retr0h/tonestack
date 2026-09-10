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
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/retr0h/tonestack/pkg/sdk/internal/gen"
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
func Load(dir string) ([]gen.RigSpec, error) {
	// No directory means the recipes that ship in the binary, which is the
	// case for anyone who has not written their own.
	if dir == "" {
		return loadFS(rigs.FS, ".")
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
func departures(all []gen.RigSpec, spec gen.RigSpec) []result.Variant {
	out := []result.Variant(nil)

	for _, other := range all {
		if other.Extends == nil || *other.Extends != spec.ID {
			continue
		}

		out = append(out, result.Variant{ID: other.ID, Name: other.Subject.Name})
	}

	return out
}

// List reads every rig under dir.
func List(dir string) (result.Recipes, error) {
	all, err := Load(dir)
	if err != nil {
		return result.Recipes{}, err
	}

	return result.Recipes{Dir: dir, Rigs: all}, nil
}

// Show reads one rig, and what the rest of the set says about it.
func Show(dir, id string) (result.Recipe, error) {
	all, err := Load(dir)
	if err != nil {
		return result.Recipe{}, err
	}

	spec, err := find(all, id)
	if err != nil {
		return result.Recipe{}, err
	}

	return result.Recipe{Rig: spec, Variants: departures(all, spec)}, nil
}
