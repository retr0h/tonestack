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

package recipes_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/internal/recipes"
)

// ScaffoldPublicTestSuite covers starting one rig from another.
type ScaffoldPublicTestSuite struct {
	suite.Suite
}

// parent writes a rig for a copy to be made from.
func (s *ScaffoldPublicTestSuite) parent(
	dir string,
) {
	artists := filepath.Join(dir, "artists")
	s.Require().NoError(os.MkdirAll(artists, 0o750))
	s.Require().NoError(os.WriteFile(
		filepath.Join(
			artists,
			"parent.yaml",
		),
		[]byte(`# A header describing the parent, which the copy does not inherit.
#
# More of it.

schema: RigSpec
version: 2
id: parent
aliases: [other-name]
default: true

subject:
  kind: artist
  name: Parent Player
  band: A Band

instrument: bass

chain:
  - role: amp
    gear: Ampeg SVT
    evidence:
      - kind: cited
        url: https://example.test/a
    confidence: high

confidence: medium
`),
		0o600,
	))
}

// TestNewFrom covers copying a rig as the start of another.
func (s *ScaffoldPublicTestSuite) TestNewFrom() {
	tests := []struct {
		name   string
		from   string
		kind   string
		who    string
		want   []string
		absent []string
		broken bool
		// unreadable puts a directory where the walk expects a file.
		unreadable bool
		errText    string
		// instrument is what the report says the copy is played on.
		instrument string
		// named and amp are what the report says the copy is called and
		// which amplifier it holds.
		named string
		amp   string
		// base is the directory beneath, or empty for the rigs that ship.
		base string
	}{
		{
			// Nobody named the copy, so it keeps the parent's name, and the
			// chain comes across whole, so it holds the parent's amp.
			name:       "a copy of a rig in the same directory",
			from:       "parent",
			instrument: "bass",
			named:      "Parent Player",
			amp:        "Ampeg SVT",
			want: []string{
				"id: copy\nextends: parent",
				// The citation comes across, which is the point and the
				// hazard, so the header says to re-check it.
				"url: https://example.test/a",
				"gear: Ampeg SVT",
				"Every citation below came across with the copy",
			},
			absent: []string{
				// The parent's identity, which is not the copy's.
				"aliases:", "default: true",
				"# A header describing the parent",
			},
		},
		{
			name:  "a copy that is one song rather than a player",
			from:  "parent",
			kind:  "song",
			who:   "One Song",
			named: "One Song",
			amp:   "Ampeg SVT",
			want:  []string{"kind: song", "name: One Song"},
			// The band survives, because the song is still by them.
			absent: []string{"kind: artist", "name: Parent Player"},
		},
		{
			name: "a copy of a rig that ships in the binary",
			from: "mike-dirnt",
			want: []string{"extends: mike-dirnt", "gear: Ampeg SVT"},
		},
		{
			// An alias belongs to the parent, and `extends` is matched
			// against an id, so a copy made by alias has to record what the
			// alias resolved to or the link never fires.
			name:   "a copy made by one of the parent's aliases",
			from:   "dirnt",
			want:   []string{"extends: mike-dirnt"},
			absent: []string{"extends: dirnt"},
		},
		{
			// The rigs beneath are read whole, the way a lookup reads them.
			name:    "a directory beneath that will not load",
			from:    "parent",
			base:    "testdata",
			errText: "broken.yaml",
		},
		{
			name:    "a copy of a rig nobody has",
			from:    "nobody-at-all",
			errText: "no such recipe",
		},
		{
			// A broken rig beside the parent is not the parent, so it does
			// not stop the copy.
			name:   "a rig that is not one, beside the parent",
			from:   "parent",
			broken: true,
			want:   []string{"id: copy\nextends: parent"},
		},
		{
			name:    "a rig that is not one, asked for",
			from:    "broken",
			broken:  true,
			errText: "not a valid rig",
		},
		{
			// A directory named like a rig: the glob matches it and reading
			// it cannot work.
			name:       "a directory wearing a rig's name, asked for",
			from:       "adir",
			unreadable: true,
			errText:    "opening",
		},
		{
			name:       "a directory wearing a rig's name, beside the parent",
			from:       "parent",
			unreadable: true,
			want:       []string{"extends: parent"},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			dir := s.T().TempDir()
			s.parent(dir)

			if tt.broken {
				s.Require().NoError(os.WriteFile(
					filepath.Join(dir, "artists", "broken.yaml"),
					[]byte("schema: RigSpec\nid: broken\n"), 0o600))
			}

			if tt.unreadable {
				s.Require().NoError(os.MkdirAll(
					filepath.Join(dir, "artists", "adir.yaml"), 0o750))
			}

			got, err := recipes.New(context.Background(), recipes.NewOptions{
				Dir:  dir,
				Base: tt.base,
				ID:   "copy",
				From: tt.from,
				Kind: tt.kind,
				Name: tt.who,
			})

			if tt.errText != "" {
				s.Require().ErrorContains(err, tt.errText)

				return
			}

			s.Require().NoError(err)

			if tt.instrument != "" {
				s.Require().Equal(tt.instrument, got.Instrument)
			}

			if tt.named != "" {
				s.Require().Equal(tt.named, got.Name)
			}

			if tt.amp != "" {
				s.Require().Equal(tt.amp, got.Amp)
			}

			body, err := os.ReadFile(filepath.Join(dir, "artists", "copy.yaml"))
			s.Require().NoError(err)

			for _, want := range tt.want {
				s.Require().Contains(string(body), want)
			}

			for _, absent := range tt.absent {
				s.Require().NotContains(string(body), absent)
			}

			if tt.broken || tt.unreadable {
				return
			}

			// A copy is a whole rig, so it loads on its own.
			all, err := recipes.Load(dir)
			s.Require().NoError(err)
			s.Require().Len(all, 2)
		})
	}
}

func TestScaffoldPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ScaffoldPublicTestSuite))
}
