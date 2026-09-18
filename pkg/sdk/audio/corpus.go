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
	"fmt"
	"io/fs"
	"path"
	"sort"
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

		if len(got) == 0 {
			continue
		}

		all := make([]Profile, 0, len(got))
		for _, n := range got {
			all = append(all, n.Profile)
		}

		a := Together(all)
		together[e.Name()] = a

		out = append(out, Player{ID: e.Name(), Records: len(got), Across: a})
	}

	for i := range out {
		out[i].Terms = Derive(out[i].Across, without(together, out[i].ID))
	}

	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })

	return out, nil
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
