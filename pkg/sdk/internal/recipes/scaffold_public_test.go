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

// nextLine, lineSep and paraSep end a line for a YAML parser, the way a
// newline does. Written as runes so the source stays readable.
var (
	nextLine = string(rune(0x0085))
	lineSep  = string(rune(0x2028))
	paraSep  = string(rune(0x2029))
)

// subjectBlock is the parent's subject as most rigs write it.
const subjectBlock = `subject:
  kind: artist
  name: Parent Player
  band: A Band
`

// requiresAndEvidence is what a rig needs and what it rests on, written as
// rigs marshalled by machine write them: the keys after the first in each
// entry sit at the indent a subject's own keys sit at.
const requiresAndEvidence = `requires:
- name: My IR Pack
  kind: ir

evidence:
- url: https://example.test/b
  kind: cited

`

// meteor is a rig read off a device. Its keys are in marshalled order, so
// `schema` comes late, and five lines beside the subject's name are
// `  name:` at the same indent.
var meteor = filepath.Join(
	"..", "..", "..", "..", "examples", "rigspec", "dir-angl-meteor.yaml")

// parentRig is the rig a copy is made from, in the shapes a rig comes in.
type parentRig struct {
	// subject replaces the parent's subject block.
	subject string
	// extra is written between the instrument and the chain.
	extra string
	// lead is written above the subject.
	lead string
	// breaks is what ends a line, where that is not a newline.
	breaks string
}

// text writes the parent out.
func (p parentRig) text() string {
	subject := subjectBlock
	if p.subject != "" {
		subject = p.subject
	}

	body := `# A header describing the parent, which the copy does not inherit.
#
# More of it.

schema: RigSpec
version: 2
id: parent
aliases: [other-name]
default: true

` + p.lead + subject + `
instrument: bass

` + p.extra + `chain:
  - role: amp
    gear: Ampeg SVT
    evidence:
      - kind: cited
        url: https://example.test/a
    confidence: high

confidence: medium
`

	if p.breaks != "" {
		body = strings.ReplaceAll(body, "\n", p.breaks)
	}

	return body
}

// parent writes a rig for a copy to be made from.
func (s *ScaffoldPublicTestSuite) parent(
	dir string,
	rig parentRig,
) {
	artists := filepath.Join(dir, "artists")
	s.Require().NoError(os.MkdirAll(artists, 0o750))
	s.Require().NoError(os.WriteFile(
		filepath.Join(artists, "parent.yaml"), []byte(rig.text()), 0o600))
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
	rig parentRig,
	withExample bool,
	opts recipes.NewOptions,
) (string, error) {
	dir := s.T().TempDir()
	s.parent(dir, rig)

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
		// rig is the shape of the rig copied from.
		rig parentRig
		// withExample writes the meteor rig beside the parent.
		withExample bool
		// compare makes the same copy without the name as well, and
		// requires the two files to differ in the subject's name alone.
		compare bool
		// loads and loadsKind are what the copy says it is once read back.
		loads     string
		loadsKind string
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
			name:      "a copy that is one song rather than a player",
			from:      "parent",
			kind:      "song",
			who:       "One Song",
			named:     "One Song",
			amp:       "Ampeg SVT",
			loads:     "One Song",
			loadsKind: "song",
			want:      []string{"kind: song", "name: One Song", "band: A Band"},
			// The band survives, because the song is still by them.
			absent: []string{"kind: artist", "name: Parent Player"},
		},
		{
			// `kind` is the subject's, and a rig says what it requires and
			// what it rests on in the same word at the same indent.
			name:      "a copy of a rig that says what it requires",
			from:      "parent",
			kind:      "song",
			rig:       parentRig{extra: requiresAndEvidence},
			loadsKind: "song",
			want:      []string{"kind: ir", "kind: cited", "  kind: song"},
		},
		{
			// Nothing matched a flow subject's kind, so asking for one was
			// taken and then ignored.
			name:      "a copy of a rig whose subject is a flow, made one song",
			from:      "parent",
			kind:      "song",
			rig:       parentRig{subject: "subject: { kind: artist, name: Parent Player }\n"},
			loadsKind: "song",
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
			// The loader reads YAML 1.1, where a bare Yes is a boolean and a
			// rig's name has to be a string. Yes and No are band names.
			name:  "a name that is a word for yes",
			from:  "parent",
			who:   "Yes",
			loads: "Yes",
		},
		{
			name:  "a name that is a word for no",
			from:  "parent",
			who:   "No",
			loads: "No",
		},
		{
			name:  "a name that is a switch position",
			from:  "parent",
			who:   "on",
			loads: "on",
		},
		{
			name:  "a name that is a letter for no",
			from:  "parent",
			who:   "n",
			loads: "n",
		},
		{
			name:  "a name that is a letter for yes",
			from:  "parent",
			who:   "y",
			loads: "y",
		},
		{
			name:  "a name that is a word for true",
			from:  "parent",
			who:   "true",
			loads: "true",
		},
		{
			name:  "a name that is a number",
			from:  "parent",
			who:   "123",
			loads: "123",
		},
		{
			name:  "a name that is a word for nothing",
			from:  "parent",
			who:   "~",
			loads: "~",
		},
		{
			name: "a name with flow syntax, in a subject written as a flow",
			from: "parent",
			rig: parentRig{
				subject: "subject: { kind: artist, name: Parent Player, band: A Band }\n",
			},
			who:   "a, b}",
			loads: "a, b}",
			want:  []string{"band: A Band }"},
		},
		{
			// The comment is the parent's, about the line and not the name.
			name: "a name with a comment after it",
			from: "parent",
			rig: parentRig{
				subject: "subject:\n  kind: artist\n  name: Parent Player # who it is\n",
			},
			who:   "Someone Else",
			loads: "Someone Else",
			want:  []string{"# who it is"},
		},
		{
			name:   "a name the parent quoted",
			from:   "parent",
			rig:    parentRig{subject: "subject:\n  kind: artist\n  name: 'Parent '' Player'\n"},
			who:    `it's: "live"`,
			loads:  `it's: "live"`,
			absent: []string{"Parent"},
		},
		{
			name: "a name the parent double quoted",
			from: "parent",
			rig: parentRig{
				subject: "subject:\n  kind: artist\n  name: \"Parent \\\" Player\"\n",
			},
			who:    "Two\nLines",
			loads:  "Two\nLines",
			absent: []string{"Parent"},
		},
		{
			// A name spread over lines cannot be replaced where it stands,
			// so the document is written out again.
			name: "a name the parent wrote over several lines",
			from: "parent",
			rig: parentRig{
				subject: "subject:\n  kind: artist\n  name: >-\n    Parent\n    Player\n",
			},
			who:    "a: b",
			loads:  "a: b",
			absent: []string{"Parent"},
			want:   []string{"url: https://example.test/a"},
		},
		{
			name: "a quoted name the parent closed on a later line",
			from: "parent",
			rig: parentRig{
				subject: "subject:\n  kind: artist\n  name: \"Parent\n    Player\"\n",
			},
			who:    "a: b",
			loads:  "a: b",
			absent: []string{"Parent"},
		},
		{
			// Replaced where it starts, the rest of the old name would be
			// left on the next line, so it is not replaced there.
			name:   "a bare name the parent carried onto a second line",
			from:   "parent",
			rig:    parentRig{subject: "subject:\n  kind: artist\n  name: Parent\n    Player\n"},
			who:    "a: b",
			loads:  "a: b",
			absent: []string{"Parent"},
		},
		{
			// The anchor is the name's, and the band is written as whatever
			// the name is, so dropping the anchor leaves a rig that has lost
			// a value nothing can supply.
			name: "a name the parent anchored and pointed at",
			from: "parent",
			rig: parentRig{
				subject: "subject:\n  kind: artist\n  name: &who Parent Player\n  band: *who\n",
			},
			who:    "Someone Else",
			loads:  "Someone Else",
			absent: []string{"Parent Player"},
		},
		{
			// A parser ends a line at any of these, so a file written with
			// them holds more lines than counting newlines finds.
			name:  "a rig whose lines end in carriage returns",
			from:  "parent",
			rig:   parentRig{breaks: "\r"},
			who:   "Someone Else",
			loads: "Someone Else",
			want:  []string{"gear: Ampeg SVT"},
		},
		{
			name:  "a rig whose lines end in next-line characters",
			from:  "parent",
			rig:   parentRig{breaks: nextLine},
			who:   "Someone Else",
			loads: "Someone Else",
		},
		{
			name:  "a rig whose lines end in line separators",
			from:  "parent",
			rig:   parentRig{breaks: lineSep},
			who:   "Someone Else",
			loads: "Someone Else",
		},
		{
			name:  "a rig whose lines end in paragraph separators",
			from:  "parent",
			rig:   parentRig{breaks: paraSep},
			who:   "Someone Else",
			loads: "Someone Else",
		},
		{
			// The separator ends the comment, so the parser counts two lines
			// where counting newlines finds one, and everything below it is
			// a line further down than it looks. Renaming to the name that
			// is already there is the case a comparison cannot catch.
			name:  "a comment above the name carrying a line separator",
			from:  "parent",
			rig:   parentRig{lead: "# a note" + lineSep + "# and more of it\n"},
			who:   "Parent Player",
			loads: "Parent Player",
			want: []string{
				"# a note" + lineSep + "# and more of it",
				"  name: Parent Player",
			},
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
			s.parent(dir, tt.rig)

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

				other, err := s.copyOf(tt.rig, tt.withExample, unnamed)
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

				if tt.loadsKind != "" {
					s.Require().Equal(tt.loadsKind, string(spec.Subject.Kind))
				}
			}

			s.Require().True(found, "the copy loads")
		})
	}
}

// TestReplaceSubject covers replacing a subject's field in a rig's own text.
//
// New cannot reach these: it rewrites a rig that loaded, and a rig that
// loaded has a subject carrying both fields, because the contract requires
// them. This is its own helper with its own contract, and what it does with
// a document that has no such field is part of that contract.
func (s *ScaffoldPublicTestSuite) TestReplaceSubject() {
	tests := []struct {
		name  string
		body  string
		key   string
		value string
		want  string
		// errText is what rewriting fails with, and nothing is returned.
		errText string
	}{
		{
			name:  "a rig with a name to replace",
			body:  "schema: RigSpec\nsubject:\n  kind: artist\n  name: Parent Player\n",
			key:   "name",
			value: "Someone Else",
			want:  "  name: Someone Else\n",
		},
		{
			name:  "a rig with a kind to replace",
			body:  "schema: RigSpec\nsubject:\n  kind: artist\n  name: Parent Player\n",
			key:   "kind",
			value: "song",
			want:  "  kind: song\n",
		},
		{
			name:    "a document that is not YAML",
			body:    "subject: [\n  unclosed\n",
			key:     "name",
			errText: "rewriting the copy's subject",
		},
		{
			name:    "a rig with no subject",
			body:    "schema: RigSpec\nid: parent\n",
			key:     "name",
			errText: "no such field to replace",
		},
		{
			name:    "a subject that holds no name",
			body:    "schema: RigSpec\nsubject:\n  kind: artist\n",
			key:     "name",
			errText: "no such field to replace",
		},
		{
			name:    "a subject that is not a mapping",
			body:    "schema: RigSpec\nsubject: Parent Player\n",
			key:     "name",
			errText: "no such field to replace",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := recipes.ReplaceSubject(tt.body, tt.key, tt.value)

			if tt.errText != "" {
				s.Require().ErrorContains(err, tt.errText)
				s.Require().Empty(got)

				return
			}

			s.Require().NoError(err)
			s.Require().Contains(got, tt.want)
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
