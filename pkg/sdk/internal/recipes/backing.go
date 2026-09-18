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

	"github.com/retr0h/tonestack/pkg/sdk/audio"
	"github.com/retr0h/tonestack/pkg/sdk/result"
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

	for _, spec := range specs(all.merged()) {
		one := result.Backing{ID: spec.ID}

		if spec.Subject.Era != nil {
			one.Era = *spec.Subject.Era
		}

		if spec.Subject.Years != nil {
			one.From, one.To = spec.Subject.Years.From, spec.Subject.Years.To
		}

		records, err := recordsFor(corpus, spec.ID)
		if err != nil {
			return nil, err
		}

		for _, r := range records {
			one.Records = append(one.Records, result.Record{
				Track:   r.Track,
				Year:    r.Year,
				Outside: one.Stated() && (r.Year < one.From || r.Year > one.To),
			})
		}

		out = append(out, one)
	}

	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })

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
