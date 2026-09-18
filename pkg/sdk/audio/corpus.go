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
package audio

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
)

// Player is one artist in a music corpus: what their records measure as, and
// what that says about them against everybody else.
type Player struct {
	// ID is the directory the records sit in, which is the rig's identifier.
	ID string
	// Records is how many of their recordings were measured.
	Records int
	// Across is what those records measure as together.
	Across Across
	// Terms are the words their figures earn against the other players.
	// Empty is the ordinary answer with few players, and it means the
	// evidence is mixed rather than that something went wrong.
	Terms []Derived
}

// Corpus measures every player under a tree and derives what each one's
// figures say against the others.
//
// One directory per player, holding their records somewhere beneath it:
//
//	resources/music/bass/mike-dirnt/stems/htdemucs/longview/bass.wav
//	resources/music/bass/flea/stems/htdemucs/aeroplane/bass.wav
//
// Deriving needs the comparison, which is why this exists at all: a term is
// earned by sitting clear of the other players, so one player's records can
// be measured alone but can never earn a word.
//
// A directory holding no recordings is not a player, and is skipped rather
// than reported as one measuring nothing.
func Corpus(
	fsys fs.FS,
	root string,
) ([]Player, error) {
	entries, err := fs.ReadDir(fsys, root)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", root, err)
	}

	out := []Player(nil)
	together := map[string]Across{}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}

		got, err := MeasureAll(fsys, path.Join(root, e.Name()))
		if err != nil {
			return nil, err
		}

		got, err = named(fsys, path.Join(root, e.Name()), got)
		if err != nil {
			return nil, err
		}

		if len(got) == 0 {
			continue
		}

		a := Together(profilesOf(got))
		together[e.Name()] = a

		out = append(out, Player{ID: e.Name(), Records: len(got), Across: a})
	}

	for i := range out {
		out[i].Terms = Derive(out[i].Across, without(together, out[i].ID))
	}

	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })

	return out, nil
}

// named keeps the recordings the manifest names, where there is one.
//
// The manifest is the statement of what was measured and the disk is
// incidental: a record dropped from a manifest is a record somebody decided
// not to measure, and its stems are still sitting there because separating
// takes minutes and nothing here deletes audio. Reading the tree alone would
// measure it anyway.
//
// That is not hypothetical. Six records were taken out of four manifests for
// being from the wrong era, their stems stayed on disk as the workflow says
// they should, and without this every one of them would still be in the
// figures.
//
// A player with no manifest is measured as found, which is what a directory
// somebody is still assembling looks like.
func named(
	fsys fs.FS,
	dir string,
	got []Named,
) ([]Named, error) {
	f, err := fsys.Open(path.Join(dir, "corpus.yaml"))
	if errors.Is(err, fs.ErrNotExist) {
		return got, nil
	}

	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", dir, err)
	}

	// Opened read-only, so Close has nothing to report the read did not.
	defer func() { _ = f.Close() }()

	m, err := ReadManifest(f)
	if err != nil {
		return nil, err
	}

	wanted := make(map[string]bool, len(m.Tracks))
	for _, rec := range m.Tracks {
		wanted[strings.ToLower(rec.Track)] = true
	}

	out := make([]Named, 0, len(got))

	for _, n := range got {
		if wanted[strings.ToLower(n.Name)] {
			out = append(out, n)
		}
	}

	return out, nil
}

// profilesOf is what each of a player's records measured as.
func profilesOf(
	of []Named,
) []Profile {
	out := make([]Profile, 0, len(of))
	for _, n := range of {
		out = append(out, n.Profile)
	}

	return out
}

// without is everybody else, which is what a player is compared against.
func without(
	all map[string]Across,
	id string,
) map[string]Across {
	out := make(map[string]Across, len(all))

	for name, a := range all {
		if name != id {
			out[name] = a
		}
	}

	return out
}
