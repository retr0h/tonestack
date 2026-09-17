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
	"bytes"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"
)

// Named is one recording's measurement and what to call it.
type Named struct {
	// Name is what to call the recording in a report.
	Name string
	// Profile is what it measured as.
	Profile Profile
	// Source is where the recording came from, when a manifest said. Empty
	// otherwise: a measurement without a link is still a measurement, and the
	// audio on disk is not a source anybody else can check.
	Source Record
}

// MeasureAll measures every .wav in a tree, in name order.
//
// A corpus rather than a take: several records by one player, so the answer
// describes a habit rather than one session. The order is fixed so two runs
// over the same directory read the same way.
//
// Separation tools write one directory per track holding stems named for the
// instrument, so a file called bass.wav is named for the directory holding it
// rather than for itself. Four rows reading "bass" would name nothing.
func MeasureAll(
	fsys fs.FS,
	root string,
) ([]Named, error) {
	var out []Named

	err := fs.WalkDir(fsys, root, func(at string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.EqualFold(path.Ext(at), ".wav") {
			return nil
		}

		if isAccompaniment(at) {
			return nil
		}

		p, err := measureOne(fsys, at)
		if err != nil {
			return err
		}

		out = append(out, Named{Name: nameFor(at), Profile: p})

		return nil
	})
	if err != nil {
		return nil, err
	}

	slices.SortFunc(out, func(a, b Named) int { return strings.Compare(a.Name, b.Name) })

	return out, nil
}

// measureOne reads and measures a single file.
//
// Read in full rather than streamed. A decoder seeks around a WAV's headers,
// an fs.File is not required to seek, and a recording is small enough that
// holding one is cheaper than the alternatives.
func measureOne(
	fsys fs.FS,
	at string,
) (Profile, error) {
	data, err := fs.ReadFile(fsys, at)
	if err != nil {
		return Profile{}, fmt.Errorf("reading %s: %w", at, err)
	}

	samples, rate, err := Read(bytes.NewReader(data))
	if err != nil {
		return Profile{}, fmt.Errorf("reading %s: %w", at, err)
	}

	return Measure(samples, rate), nil
}

// nameFor is what to call a recording in a report.
//
// A stem carries the instrument's name rather than the track's, so the
// directory holding it is the name worth printing.
func nameFor(
	at string,
) string {
	base := strings.TrimSuffix(path.Base(at), path.Ext(at))

	if dir := path.Base(path.Dir(at)); isStem(base) && dir != "." && dir != "/" {
		return dir
	}

	return base
}

// isAccompaniment says whether a file is everything except the instrument.
//
// A two-stem separation writes both halves side by side: bass.wav and
// no_bass.wav. Measuring both would put two rows under one track name and
// build a corpus half made of the parts the bass was taken out of, which is
// the opposite of what separating was for.
func isAccompaniment(
	at string,
) bool {
	base := strings.TrimSuffix(path.Base(at), path.Ext(at))

	return strings.HasPrefix(strings.ToLower(base), "no_")
}

// isStem says whether a filename names an instrument rather than a recording.
//
// The four a two-stem or four-stem separation writes. Anything else is taken
// to be the track's own name.
func isStem(
	base string,
) bool {
	switch strings.ToLower(base) {
	case "bass", "drums", "vocals", "other", "no_bass", "guitar", "piano":
		return true
	default:
		return false
	}
}
