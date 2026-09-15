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
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/internal/recipes"
)

// ScaffoldPublicTestSuite covers starting one rig from another.
type ScaffoldPublicTestSuite struct {
	suite.Suite
}

// subjectBlock is the parent's subject as most rigs write it.
const subjectBlock = `subject:
  kind: artist
  name: Parent Player
  band: A Band
`

// meteor is a rig read off a device. Its keys are in marshalled order, so
// `schema` comes late, and five lines beside the subject's name are
// `  name:` at the same indent.
var meteor = filepath.Join(
	"..", "..", "..", "..", "examples", "rigspec", "dir-angl-meteor.yaml")

// parent writes a rig for a copy to be made from, with subject as its
// subject block.
func (s *ScaffoldPublicTestSuite) parent(
	dir string,
	subject string,
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

`+subject+`
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

// example writes the meteor rig beside the parent.
func (s *ScaffoldPublicTestSuite) example(
	dir string,
) {
	raw, err := os.ReadFile(meteor)
	s.Require().NoError(err)
	s.Require().NoError(os.WriteFile(
		filepath.Join(dir, "artists", "dir-angl-meteor.yaml"), raw, 0o600))
}

// copyOf makes the copy a row asks for, in a directory of its own, and
// returns the directory it went to.
func (s *ScaffoldPublicTestSuite) copyOf(
	subject string,
	withExample bool,
	opts recipes.NewOptions,
) (string, error) {
	dir := s.T().TempDir()
	s.parent(dir, subject)

	if withExample {
		s.example(dir)
	}

	opts.Dir = dir
	opts.ID = "copy"

	_, err := recipes.New(context.Background(), opts)

	return dir, err
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
		// cab and pedals are the rest of the gear the report names, and
		// noPedals says it names none.
		cab      string
		pedals   []string
		noPedals bool
		// base is the directory beneath, or empty for the rigs that ship.
		base string
		// subject replaces the parent's subject block.
		subject string
		// withExample writes the meteor rig beside the parent.
		withExample bool
		// compare makes the same copy without the name as well, and
		// requires the two files to differ in the subject's name alone.
		compare bool
		// loads is the name the copy has once read back.
		loads string
	}{
		{
			// Nobody named the copy, so it keeps the parent's name, and the
			// chain comes across whole, so it holds the parent's amp.
			name:       "a copy of a rig in the same directory",
			from:       "parent",
			instrument: "bass",
			named:      "Parent Player",
			amp:        "Ampeg SVT",
			noPedals:   true,
			loads:      "Parent Player",
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
			loads: "One Song",
			want:  []string{"kind: song", "name: One Song", "band: A Band"},
			// The band survives, because the song is still by them.
			absent: []string{"kind: artist", "name: Parent Player"},
		},
		{
			name: "a copy of a rig that ships in the binary",
			from: "mike-dirnt",
			cab:  "Ampeg 8x10",
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
			// Five other lines in this rig are `  name:` at the subject's
			// indent: a device's name, a snapshot's, and so on. Renaming
			// the rig is not renaming those.
			name:        "a renamed copy of a rig read off a device",
			from:        "dir-angl-meteor",
			withExample: true,
			who:         "My Meteor",
			compare:     true,
			loads:       "My Meteor",
			named:       "My Meteor",
			amp:         "ENGL® Fireball 100",
			cab:         `Legacy 4x12" ENGL XXL V30`,
			// Everything else in the chain, in the order the signal meets it.
			pedals: []string{
				"Arbiter Cry Baby",
				"Ibanez® TS808 Tube Screamer®",
				"Mono, Stereo Line 6 Original",
				"Simple Delay",
				"Room",
			},
			// The rig's keys come before `schema`, and all of them are
			// the rig.
			want: []string{"id: copy\nextends: dir-angl-meteor", "HD2_ReverbRoom"},
			absent: []string{
				"id: dir-angl-meteor",
				"# A rig read off a device, not written by hand.",
			},
		},
		{
			// Each of these is YAML syntax written bare after `name: `.
			name:  "a name that is a key and a value",
			from:  "parent",
			who:   "a: b",
			loads: "a: b",
		},
		{
			name:  "a name starting with a hash",
			from:  "parent",
			who:   "#hash",
			loads: "#hash",
		},
		{
			name:  "a name starting with a dash",
			from:  "parent",
			who:   "- dash",
			loads: "- dash",
		},
		{
			name:  "a name in quotes",
			from:  "parent",
			who:   `"quoted"`,
			loads: `"quoted"`,
		},
		{
			name:  "a name with a colon and a quote",
			from:  "parent",
			who:   `Bob's: "live"`,
			loads: `Bob's: "live"`,
		},
		{
			// Bare, a comma and a brace end the value inside a flow.
			name:    "a name with flow syntax, in a subject written as a flow",
			from:    "parent",
			subject: "subject: { kind: artist, name: Parent Player, band: A Band }\n",
			who:     "a, b}",
			loads:   "a, b}",
			want:    []string{"band: A Band }"},
		},
		{
			// The comment is the parent's, about the line and not the name.
			name:    "a name with a comment after it",
			from:    "parent",
			subject: "subject:\n  kind: artist\n  name: Parent Player # who it is\n",
			who:     "Someone Else",
			loads:   "Someone Else",
			want:    []string{"# who it is"},
		},
		{
			name:    "a name the parent quoted",
			from:    "parent",
			subject: "subject:\n  kind: artist\n  name: 'Parent '' Player'\n",
			who:     "it's",
			loads:   "it's",
			absent:  []string{"Parent"},
		},
		{
			name:    "a name the parent double quoted",
			from:    "parent",
			subject: "subject:\n  kind: artist\n  name: \"Parent \\\" Player\"\n",
			who:     "Two\nLines",
			loads:   "Two\nLines",
			absent:  []string{"Parent"},
		},
		{
			// A name spread over lines cannot be replaced where it stands,
			// so the document is written out again.
			name:    "a name the parent wrote over several lines",
			from:    "parent",
			subject: "subject:\n  kind: artist\n  name: >-\n    Parent\n    Player\n",
			who:     "a: b",
			loads:   "a: b",
			absent:  []string{"Parent"},
			want:    []string{"url: https://example.test/a"},
		},
		{
			name:    "a quoted name the parent closed on a later line",
			from:    "parent",
			subject: "subject:\n  kind: artist\n  name: \"Parent\n    Player\"\n",
			who:     "a: b",
			loads:   "a: b",
			absent:  []string{"Parent"},
		},
		{
			// Replaced where it starts, the rest of the old name would be
			// left on the next line, so it is not replaced there.
			name:    "a bare name the parent carried onto a second line",
			from:    "parent",
			subject: "subject:\n  kind: artist\n  name: Parent\n    Player\n",
			who:     "a: b",
			loads:   "a: b",
			absent:  []string{"Parent"},
		},
		{
			// Replacing the text after `name: ` would drop the anchor with
			// the name, so the document is written out again instead.
			name:    "a name the parent anchored",
			from:    "parent",
			subject: "subject:\n  kind: artist\n  name: &who Parent Player\n",
			who:     "Someone Else",
			loads:   "Someone Else",
			absent:  []string{"&who", "Parent Player"},
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
			subject := subjectBlock
			if tt.subject != "" {
				subject = tt.subject
			}

			dir := s.T().TempDir()
			s.parent(dir, subject)

			if tt.withExample {
				s.example(dir)
			}

			if tt.broken {
				s.Require().NoError(os.WriteFile(
					filepath.Join(dir, "artists", "broken.yaml"),
					[]byte("schema: RigSpec\nid: broken\n"), 0o600))
			}

			if tt.unreadable {
				s.Require().NoError(os.MkdirAll(
					filepath.Join(dir, "artists", "adir.yaml"), 0o750))
			}

			opts := recipes.NewOptions{
				Base: tt.base,
				From: tt.from,
				Kind: tt.kind,
				Name: tt.who,
			}

			o := opts
			o.Dir = dir
			o.ID = "copy"

			got, err := recipes.New(context.Background(), o)

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

			if tt.cab != "" {
				s.Require().Equal(tt.cab, got.Cab)
			}

			if tt.pedals != nil {
				s.Require().Equal(tt.pedals, got.Pedals)
			}

			if tt.noPedals {
				s.Require().Empty(got.Pedals)
			}

			body, err := os.ReadFile(filepath.Join(dir, "artists", "copy.yaml"))
			s.Require().NoError(err)

			for _, want := range tt.want {
				s.Require().Contains(string(body), want)
			}

			for _, absent := range tt.absent {
				s.Require().NotContains(string(body), absent)
			}

			if tt.compare {
				unnamed := opts
				unnamed.Name = ""

				other, err := s.copyOf(subject, tt.withExample, unnamed)
				s.Require().NoError(err)

				plain, err := os.ReadFile(filepath.Join(other, "artists", "copy.yaml"))
				s.Require().NoError(err)

				s.requireOnlyNameDiffers(string(plain), string(body))
			}

			if tt.broken || tt.unreadable {
				return
			}

			// A copy is a whole rig, so it loads on its own.
			all, err := recipes.Load(dir)
			s.Require().NoError(err)

			found := false

			for _, spec := range all {
				if spec.ID != "copy" {
					continue
				}

				found = true

				if tt.loads != "" {
					s.Require().Equal(tt.loads, spec.Subject.Name)
				}
			}

			s.Require().True(found, "the copy loads")
		})
	}
}

// requireOnlyNameDiffers requires two copies to be the same file but for the
// subject's name.
func (s *ScaffoldPublicTestSuite) requireOnlyNameDiffers(
	plain, named string,
) {
	was := strings.Split(plain, "\n")
	now := strings.Split(named, "\n")
	s.Require().Len(now, len(was))

	var differ []int

	for i := range was {
		if was[i] != now[i] {
			differ = append(differ, i)
		}
	}

	s.Require().Len(differ, 1, "only the subject's name changes")
	s.Require().Equal("  name: DIR:ANGL Meteor", was[differ[0]])
	s.Require().Equal("  name: My Meteor", now[differ[0]])
}

func TestScaffoldPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ScaffoldPublicTestSuite))
}
