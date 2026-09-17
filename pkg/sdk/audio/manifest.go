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
	"io"
	"regexp"
	"strings"

	"go.yaml.in/yaml/v3"
)

// A corpus is audio nobody may redistribute and a record of what it was.
//
// The audio cannot be committed: it is somebody else's, and a repository is
// not a way around that. The record of which songs were measured can be, and
// it is the half that matters to anybody reading a rig afterwards. Mirrors
// how the preset corpus works, where the fetch script and the attribution are
// committed and the payload is not.
//
// So the manifest names tracks and links them. It never names a file.

// Record names one recording and where somebody else can find it.
type Record struct {
	// Track is the recording's name, matching the stem directory measured.
	Track string `yaml:"track"`
	// URL is where to hear it. Not where the audio lives on disk: a link is
	// checkable by somebody who does not have the file.
	URL string `yaml:"url"`
	// At is the part measured, as "1:20" or "1:20-1:45".
	At string `yaml:"at"`
	// Note is anything worth saying about this recording in particular.
	Note string `yaml:"note"`
}

// Manifest is a corpus of recordings, without the recordings.
type Manifest struct {
	// Artist is whose playing this corpus is of.
	Artist string `yaml:"artist"`
	// Tracks is what was measured, one entry per recording.
	Tracks []Record `yaml:"tracks"`
}

// Where a timestamp is allowed to point, and what a link has to look like.
//
// The same shapes the rig contract accepts, checked here so a bad one is
// caught while somebody is looking at the manifest. Left until build time it
// surfaces as a validation failure against `chain[0].evidence[1].at`, which
// says nothing about which song was wrong.
var (
	atPattern  = regexp.MustCompile(`^\d{1,2}:\d{2}(:\d{2})?(-\d{1,2}:\d{2}(:\d{2})?)?$`)
	urlPattern = regexp.MustCompile(`^https?://\S+$`)
)

// ReadManifest reads a corpus manifest.
//
// Unknown fields are refused rather than ignored. A manifest is written by
// hand, `track:` and `tracks:` are one letter apart, and a typo that silently
// measures nothing is worse than one that stops.
func ReadManifest(
	r io.Reader,
) (Manifest, error) {
	var m Manifest

	dec := yaml.NewDecoder(r)
	dec.KnownFields(true)

	if err := dec.Decode(&m); err != nil {
		return Manifest{}, fmt.Errorf("reading manifest: %w", err)
	}

	for _, rec := range m.Tracks {
		if err := rec.check(); err != nil {
			return Manifest{}, err
		}
	}

	return m, nil
}

// check holds one entry to what a rig will accept from it.
func (r Record) check() error {
	if r.Track == "" {
		return fmt.Errorf("reading manifest: an entry has no track name")
	}

	if r.URL != "" && !urlPattern.MatchString(r.URL) {
		return fmt.Errorf(
			"reading manifest: %s has a url that is not a link: %q", r.Track, r.URL)
	}

	if r.At != "" && !atPattern.MatchString(r.At) {
		return fmt.Errorf(
			`reading manifest: %s has an at that is not a timestamp: %q, wanted "1:20" or "1:20-1:45"`,
			r.Track,
			r.At,
		)
	}

	return nil
}

// Join attaches what the manifest knows to what was measured.
//
// Matched on the recording's name, which for a separated stem is the
// directory holding it and so is the source file's own name. Anything the
// manifest does not mention keeps an empty source rather than being dropped:
// a measurement without a link is still a measurement.
func (m Manifest) Join(
	of []Named,
) []Named {
	out := make([]Named, 0, len(of))

	for _, n := range of {
		for _, rec := range m.Tracks {
			if strings.EqualFold(rec.Track, n.Name) {
				n.Source = rec

				break
			}
		}

		out = append(out, n)
	}

	return out
}

// Unmatched is what the manifest and the recordings disagree about.
//
// Both directions, because both are mistakes somebody wants told. A track
// named in the manifest with nothing measured usually means the separation
// did not run on it; a recording nothing names is one whose evidence will go
// out with no link on it.
func (m Manifest) Unmatched(
	of []Named,
) (missing, unnamed []string) {
	measured := make(map[string]bool, len(of))
	for _, n := range of {
		measured[strings.ToLower(n.Name)] = true
	}

	named := make(map[string]bool, len(m.Tracks))

	for _, rec := range m.Tracks {
		named[strings.ToLower(rec.Track)] = true

		if !measured[strings.ToLower(rec.Track)] {
			missing = append(missing, rec.Track)
		}
	}

	for _, n := range of {
		if !named[strings.ToLower(n.Name)] {
			unnamed = append(unnamed, n.Name)
		}
	}

	return missing, unnamed
}
