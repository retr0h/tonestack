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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/internal/recipes"
)

// LayerPublicTestSuite covers somebody's own rigs read over the ones that
// ship.
type LayerPublicTestSuite struct {
	suite.Suite
}

// userRig is a rig of somebody's own, about "Their Player". extra is a line
// such as aliases or extends, or empty.
func userRig(
	id string,
	extra string,
) string {
	return "schema: RigSpec\nversion: 2\nid: " + id + "\n" + extra + `

subject:
  kind: artist
  name: Their Player

instrument: bass

chain:
  - role: amp
    gear: Aguilar DB51
    evidence:
      - { kind: cited, note: "a test says so" }
    confidence: high

confidence: high
`
}

// TestLayered covers which rig a lookup finds and what a listing shows.
func (s *LayerPublicTestSuite) TestLayered() {
	tests := []struct {
		name string
		// files are written under artists/ in their directory.
		files map[string]string
		// folders are directories under artists/ named like a rig.
		folders []string
		missing bool
		locked  bool
		// lockedArtists leaves artists/ in their directory unreadable.
		lockedArtists bool
		id            string
		// want is the subject the lookup finds.
		want string
		// variants must be among what the found rig reports.
		variants []string
		findErr  string
		listErr  string
		is       error
	}{
		{
			name: "nothing of theirs",
			id:   "mike-dirnt",
			want: "Mike Dirnt",
		},
		{
			name:  "a rig of theirs with a shipped rig's identifier",
			files: map[string]string{"mine.yaml": userRig("mike-dirnt", "")},
			id:    "mike-dirnt",
			want:  "Their Player",
		},
		{
			name:  "an alias of theirs that is a shipped rig's identifier, in any case",
			files: map[string]string{"mine.yaml": userRig("their-player", "aliases: [MIKE-DIRNT]")},
			id:    "mike-dirnt",
			want:  "Their Player",
		},
		{
			// Asked for by the shipped rig's own identifier, which no rig of
			// theirs answers to, and still theirs, as a listing shows.
			name:  "an identifier of theirs that is a shipped rig's alias",
			files: map[string]string{"mine.yaml": userRig("dirnt", "")},
			id:    "mike-dirnt",
			want:  "Their Player",
		},
		{
			name:  "a rig of theirs sharing no name with a shipped one",
			files: map[string]string{"mine.yaml": userRig("their-player", "")},
			id:    "their-player",
			want:  "Their Player",
		},
		{
			name: "a variant of theirs on a shipped rig",
			files: map[string]string{
				"mine.yaml": userRig("mike-dirnt-live", "extends: mike-dirnt"),
			},
			id:       "mike-dirnt",
			want:     "Mike Dirnt",
			variants: []string{"mike-dirnt-live"},
		},
		{
			name:    "a directory that is not there",
			missing: true,
			id:      "mike-dirnt",
			want:    "Mike Dirnt",
		},
		{
			name:    "a directory that cannot be read",
			files:   map[string]string{"mine.yaml": userRig("mike-dirnt", "")},
			locked:  true,
			id:      "mike-dirnt",
			findErr: "reading",
			listErr: "reading",
		},
		{
			// The glob would drop it, and every rig in it with no word said.
			name:          "an artists directory of theirs that cannot be read",
			files:         map[string]string{"mine.yaml": userRig("mike-dirnt", "")},
			lockedArtists: true,
			id:            "mike-dirnt",
			findErr:       "artists",
			listErr:       "artists",
		},
		{
			// Copied from the shipped rig under its own identifier, so it
			// extends the rig it replaces and is not a variant of itself.
			name:  "a rig of theirs extending the shipped rig it replaces",
			files: map[string]string{"mine.yaml": userRig("mike-dirnt", "extends: mike-dirnt")},
			id:    "mike-dirnt",
			want:  "Their Player",
		},
		{
			// One mistake in their directory does not stop every shipped rig
			// building. The listing is where it is reported.
			name:    "a file of theirs that is not a rig, beside a shipped one",
			files:   map[string]string{"broken.yaml": "schema: RigSpec\nid: broken\n"},
			id:      "mike-dirnt",
			want:    "Mike Dirnt",
			listErr: "broken.yaml",
		},
		{
			name:    "a file of theirs that is not a rig, asked for by its filename",
			files:   map[string]string{"broken.yaml": "schema: RigSpec\nid: broken\n"},
			id:      "broken",
			findErr: "broken.yaml",
			listErr: "broken.yaml",
		},
		{
			// Building the shipped rig instead would quietly pass over the
			// one they wrote to replace it.
			name:    "a file of theirs that is not a rig, named for a shipped one",
			files:   map[string]string{"mike-dirnt.yaml": "schema: RigSpec\n"},
			id:      "mike-dirnt",
			findErr: "mike-dirnt.yaml",
			listErr: "mike-dirnt.yaml",
		},
		{
			name: "a file of theirs that is not a rig, stating a shipped rig's alias",
			files: map[string]string{
				"other.yaml": "schema: RigSpec\nid: other\naliases: [DIRNT, 7]\n",
			},
			id:      "mike-dirnt",
			findErr: "other.yaml",
			listErr: "other.yaml",
		},
		{
			name:    "a file of theirs that is not a rig, stating the identifier asked for",
			files:   map[string]string{"stem.yaml": "id: wanted\n"},
			id:      "wanted",
			findErr: "stem.yaml",
			listErr: "stem.yaml",
		},
		{
			// Nothing in it can be read, so only its filename names it.
			name:    "a file of theirs that is not even YAML",
			files:   map[string]string{"garbled.yaml": "id: [unclosed\n"},
			id:      "garbled",
			findErr: "garbled.yaml",
			listErr: "garbled.yaml",
		},
		{
			name:    "a directory of theirs wearing a rig's name",
			folders: []string{"adir.yaml"},
			id:      "mike-dirnt",
			want:    "Mike Dirnt",
			listErr: "opening adir.yaml",
		},
		{
			name:    "a rig nobody wrote",
			id:      "nobody-at-all",
			findErr: "no such recipe",
			is:      recipes.ErrNotFound,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			if (tt.locked || tt.lockedArtists) && os.Geteuid() == 0 {
				s.T().Skip("root reads a directory whatever its mode")
			}

			dir := filepath.Join(s.T().TempDir(), "recipes")

			if !tt.missing {
				s.Require().NoError(os.MkdirAll(filepath.Join(dir, "artists"), 0o750))
			}

			for name, body := range tt.files {
				s.Require().NoError(os.WriteFile(
					filepath.Join(dir, "artists", name), []byte(body), 0o600))
			}

			for _, name := range tt.folders {
				s.Require().NoError(os.MkdirAll(filepath.Join(dir, "artists", name), 0o750))
			}

			if tt.locked {
				s.Require().NoError(os.Chmod(dir, 0o000))
				// Put back so the directory can be removed; a failure shows
				// up in TempDir's own cleanup.
				s.T().Cleanup(func() { _ = os.Chmod(dir, 0o750) })
			}

			if tt.lockedArtists {
				artists := filepath.Join(dir, "artists")
				s.Require().NoError(os.Chmod(artists, 0o000))
				s.T().Cleanup(func() { _ = os.Chmod(artists, 0o750) })
			}

			src := recipes.Source{User: dir}

			listed, listErr := recipes.List(src)
			shown, showErr := recipes.Show(src, tt.id)
			found, findErr := recipes.Find(src, tt.id)

			if tt.listErr != "" {
				s.Require().ErrorContains(listErr, tt.listErr)
			} else {
				s.Require().NoError(listErr)
				s.Require().Equal(dir, listed.Dir)
				s.Require().NotEmpty(listed.Rigs, "the shipped rigs are still there")

				if tt.want != "" {
					subjects := make([]string, 0, len(listed.Rigs))
					for _, r := range listed.Rigs {
						subjects = append(subjects, r.Subject.Name)
					}

					s.Require().Contains(subjects, tt.want)

					if tt.want != "Mike Dirnt" && tt.id == "mike-dirnt" {
						s.Require().NotContains(subjects, "Mike Dirnt",
							"a listing shows the rig a lookup finds, not both")
					}
				}
			}

			if tt.findErr != "" {
				s.Require().ErrorContains(showErr, tt.findErr)
				s.Require().ErrorContains(findErr, tt.findErr)

				if tt.is != nil {
					s.Require().ErrorIs(findErr, tt.is)
				}

				return
			}

			s.Require().NoError(showErr)
			s.Require().NoError(findErr)
			s.Require().Equal(tt.want, found.Subject.Name)
			s.Require().Equal(tt.want, shown.Rig.Subject.Name)

			got := make([]string, 0, len(shown.Variants))
			for _, v := range shown.Variants {
				got = append(got, v.ID)
			}

			for _, want := range tt.variants {
				s.Require().Contains(got, want)
			}

			s.Require().NotContains(got, found.ID, "a rig is not its own variant")
		})
	}
}

// TestLayeredOverADirectory covers their rigs over a directory named in place
// of the shipped ones, which are then not read at all.
func (s *LayerPublicTestSuite) TestLayeredOverADirectory() {
	user := s.T().TempDir()
	s.Require().NoError(os.MkdirAll(filepath.Join(user, "artists"), 0o750))
	s.Require().NoError(os.WriteFile(filepath.Join(user, "artists", "mine.yaml"),
		[]byte(userRig("minimal-live", "extends: minimal")), 0o600))

	src := recipes.Source{Dir: s.good(), User: user}

	shown, err := recipes.Show(src, "minimal")
	s.Require().NoError(err)
	s.Require().Len(shown.Variants, 1)
	s.Require().Equal("minimal-live", shown.Variants[0].ID)

	_, err = recipes.Find(src, "flea")
	s.Require().ErrorIs(err, recipes.ErrNotFound, "the shipped rigs are not beneath")

	_, err = recipes.List(recipes.Source{Dir: s.mixed(), User: user})
	s.Require().ErrorContains(err, "broken.yaml", "the directory beneath is read whole")
}

// good and mixed are the fixture directories RecipesPublicTestSuite reads.
func (s *LayerPublicTestSuite) good() string  { return "testdata-good" }
func (s *LayerPublicTestSuite) mixed() string { return "testdata" }

func TestLayerPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(LayerPublicTestSuite))
}
