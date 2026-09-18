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
package recipes

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/retr0h/tonestack/pkg/sdk/audio"
	"github.com/retr0h/tonestack/pkg/sdk/result"
	"github.com/retr0h/tonestack/pkg/sdk/rig"
)

// manifestName is what a player's corpus directory calls its manifest.
const manifestName = "corpus.yaml"

// Backing reads which records back each rig, and holds them to its era.
//
// The two halves of a measured claim live in different files: a rig says
// which years its gear describes, and a manifest says when each measured
// record was made. Nothing joined them until this, and four of the nine rigs
// here turned out to be deriving words from records made on other gear. Flea's
// rig is his 2012 touring rig and his records are from 1989 to 1995; Mike
// Dirnt's is the American Idiot rig and his records are Dookie and Insomniac.
//
// Reported rather than refused. Which half is wrong is a judgement: the rig
// may describe the wrong period, or the records may be the wrong records, and
// only somebody who knows the player can say which.
func Backing(
	src Source,
	corpus string,
) ([]result.Backing, error) {
	all, err := read(src)
	if err != nil {
		return nil, err
	}

	out := []result.Backing(nil)
	claimed := map[string]bool{}

	for _, spec := range specs(all.merged()) {
		claimed[spec.ID] = true
		one := result.Backing{ID: spec.ID}

		if spec.Subject.Era != nil {
			one.Era = *spec.Subject.Era
		}

		if spec.Subject.Years != nil {
			one.From, one.To = spec.Subject.Years.From, spec.Subject.Years.To
		}

		one.Direct, one.Both, one.Captured, one.Stage = rooms(spec.Chain)

		records, err := recordsFor(corpus, spec.ID)
		if err != nil {
			return nil, err
		}

		one.Misnamed = misnamed(spec.Played, records)

		for _, r := range records {
			one.Records = append(one.Records, result.Record{
				Track:   r.Track,
				Year:    r.Year,
				Outside: one.Stated() && (r.Year < one.From || r.Year > one.To),
			})
		}

		out = append(out, one)
	}

	orphans, err := unclaimed(corpus, claimed)
	if err != nil {
		return nil, err
	}

	out = append(out, orphans...)

	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })

	return out, nil
}

// unclaimed finds records sitting in a directory no rig is named for.
//
// A rig reaches its records by the directory carrying its identifier, and
// nothing else joins the two. A directory called anything else is measured by
// nobody, and it reads exactly like a rig nobody has measured yet, which is
// how a typo survives. Records fetched before the rig that will use them look
// the same and are fine, so this reports rather than refuses.
func unclaimed(
	corpus string,
	claimed map[string]bool,
) ([]result.Backing, error) {
	entries, err := os.ReadDir(corpus)
	if os.IsNotExist(err) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", corpus, err)
	}

	out := []result.Backing(nil)

	for _, e := range entries {
		if !e.IsDir() || claimed[e.Name()] {
			continue
		}

		records, err := recordsFor(corpus, e.Name())
		if err != nil {
			return nil, err
		}

		if len(records) == 0 {
			continue
		}

		one := result.Backing{ID: e.Name(), NoRig: true}
		for _, r := range records {
			one.Records = append(one.Records, result.Record{Track: r.Track, Year: r.Year})
		}

		out = append(out, one)
	}

	return out, nil
}

// recordsFor reads one player's manifest, or none where nobody has measured
// them.
//
// A rig with no corpus is the ordinary case rather than a fault: most players
// have gear evidence long before anybody owns their records.
func recordsFor(
	corpus, id string,
) ([]audio.Record, error) {
	at := filepath.Join(corpus, id, manifestName)

	f, err := os.Open(at) //nolint:gosec // a path built from the corpus given
	if os.IsNotExist(err) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", at, err)
	}

	// Opened read-only, so Close has nothing to report the read did not.
	defer func() { _ = f.Close() }()

	m, err := audio.ReadManifest(f)
	if err != nil {
		return nil, err
	}

	return m.Tracks, nil
}

// rooms counts the chain entries that never met a microphone, and those whose
// only evidence is a tour.
//
// Both are the same mistake wearing different clothes: a figure measured off a
// record attributed to gear that was not in the room. The era check catches
// the wrong decade; this catches the right decade and the wrong room.
func rooms(
	chain []rig.ChainEntry,
) (direct, both, captured, stage int) {
	for _, e := range chain {
		if e.Capture != nil {
			captured++

			switch *e.Capture {
			case rig.CaptureDirect:
				direct++
			case rig.CaptureBoth:
				both++
			case rig.CaptureMiked:
			}
		}

		if e.Stage != nil && *e.Stage {
			stage++
		}
	}

	return direct, both, captured, stage
}

// misnamed finds track names an instrument claims that no manifest carries.
//
// `played[].records` joins an instrument to the records it made by their track
// names, which is the same join the corpus directory uses and fails the same
// way: a name matching nothing is silently attributed to nothing, and reads
// like an instrument nobody has got to yet.
func misnamed(
	played *[]rig.Played,
	have []audio.Record,
) []string {
	known := map[string]bool{}
	for _, r := range have {
		known[strings.ToLower(r.Track)] = true
	}

	var out []string

	if played == nil {
		return nil
	}

	for _, p := range *played {
		if p.Records == nil {
			continue
		}

		for _, name := range *p.Records {
			if !known[strings.ToLower(name)] {
				out = append(out, name)
			}
		}
	}

	return out
}
