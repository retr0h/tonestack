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
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/internal/recipes"
	"github.com/retr0h/tonestack/pkg/sdk/rig"
)

type NewPublicTestSuite struct {
	suite.Suite
}

func (s *NewPublicTestSuite) opts(dir string) recipes.NewOptions {
	return recipes.NewOptions{
		Dir:         dir,
		ID:          "test-player",
		Name:        "Test Player",
		Instrument:  "bass",
		Amp:         "Ampeg SVT",
		CatalogPath: filepath.Join("testdata", "catalog.json"),
	}
}

// file returns a path that is a file, so making a directory under it fails.
func (s *NewPublicTestSuite) file() string {
	path := filepath.Join(s.T().TempDir(), "not-a-directory")
	s.Require().NoError(os.WriteFile(path, []byte("x"), 0o600))

	return path
}

// readOnly returns a directory a recipe cannot be written into.
func (s *NewPublicTestSuite) readOnly() string {
	dir := s.T().TempDir()

	artists := filepath.Join(dir, "artists")
	s.Require().NoError(os.MkdirAll(artists, 0o750))
	s.Require().NoError(os.Chmod(artists, 0o500))

	s.T().Cleanup(func() { s.Require().NoError(os.Chmod(artists, 0o750)) })

	return dir
}

// TestNew scaffolds a recipe.
func (s *NewPublicTestSuite) TestNew() {
	tests := []struct {
		name string
		id   string
		// an identifier nobody gave.
		noID   bool
		amp    string
		cab    string
		band   string
		pedals []string
		// the catalog to check gear against: this suite's fixture unless a
		// case says otherwise.
		catalog string
		// the real catalog this binary ships.
		builtIn bool
		// which directory to write into.
		dir string
		// write the recipe once first, so the call under test finds it there.
		twice bool

		// the gear the written recipe must name, by role and by position.
		wantAmp   string
		wantCab   string
		wantFirst string
		wantRoles []rig.Role
		says      []string

		err     error
		errText string
		absent  string
		atMost  int
	}{
		{
			// The point of scaffolding is a file the loader accepts, not a
			// template.
			name:    "a recipe naming an amplifier",
			wantAmp: "Ampeg SVT",
			says:    []string{"Test Player"},
		},
		{
			name:   "everything somebody named",
			band:   "A Band",
			cab:    "Ampeg SVT 410HLF",
			pedals: []string{"Klon Centaur"},
			// The pedal is written ahead of the amp, because that is where a
			// pedal goes and a chain is ordered by what the signal does.
			wantFirst: "Klon Centaur",
			wantRoles: []rig.Role{rig.RoleDrive, rig.RoleAmp},
			wantAmp:   "Ampeg SVT",
			wantCab:   "Ampeg SVT 410HLF",
			says:      []string{"Klon Centaur"},
		},
		{
			// Checking before writing is the point. A recipe naming an
			// amplifier nothing models is otherwise only discovered at build
			// time, by which point the name has usually been copied somewhere
			// else too.
			name:    "an amplifier nothing models",
			amp:     "Ampeg SVQ",
			err:     recipes.ErrNoSuchGear,
			errText: "did you mean",
		},
		{
			name: "a cabinet nothing models",
			cab:  "Nonesuch 9x9",
			err:  recipes.ErrNoSuchGear,
		},
		{
			name:   "a pedal nothing models",
			pedals: []string{"Nonesuch Fuzz"},
			err:    recipes.ErrNoSuchGear,
		},
		{
			// A weak suggestion is worse than none when somebody is deciding
			// whether they got a name wrong.
			name:   "a name nothing is close to",
			amp:    "Zzz",
			err:    recipes.ErrNoSuchGear,
			absent: "did you mean",
		},
		{
			// Against the real catalog "ampeg" matches far more than anybody
			// wants listed in an error message.
			name:    "a name matching far too much",
			builtIn: true,
			amp:     "Ampeg Nonesuch",
			err:     recipes.ErrNoSuchGear,
			atMost:  5,
		},
		{
			name:    "an identifier with a space in it",
			id:      "Test Player",
			err:     recipes.ErrBadID,
			errText: "hyphens",
		},
		{
			name:    "an identifier with an underscore",
			id:      "test_player",
			err:     recipes.ErrBadID,
			errText: "hyphens",
		},
		{
			name:    "no identifier at all",
			noID:    true,
			err:     recipes.ErrBadID,
			errText: "hyphens",
		},
		{
			name:    "an identifier starting with a hyphen",
			id:      "-leading",
			err:     recipes.ErrBadID,
			errText: "hyphens",
		},
		{
			name:    "an identifier ending with one",
			id:      "trailing-",
			err:     recipes.ErrBadID,
			errText: "hyphens",
		},
		{
			name:    "a recipe already written",
			twice:   true,
			err:     recipes.ErrExists,
			errText: "test-player.yaml",
		},
		{
			name:    "a directory it cannot write into",
			dir:     "read-only",
			errText: "writing",
		},
		{
			name:    "a catalog that is not there",
			catalog: filepath.Join("testdata", "nope.json"),
			errText: "catalog",
		},
		{
			name:    "a directory that cannot be made",
			dir:     "a file",
			errText: "making room",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			dir := s.T().TempDir()

			switch tt.dir {
			case "read-only":
				dir = s.readOnly()
			case "a file":
				dir = s.file()
			}

			o := s.opts(dir)
			o.Band = tt.band
			o.Cab = tt.cab
			o.Pedals = tt.pedals

			if tt.id != "" {
				o.ID = tt.id
			}

			if tt.noID {
				o.ID = ""
			}

			if tt.amp != "" {
				o.Amp = tt.amp
			}

			if tt.catalog != "" {
				o.CatalogPath = tt.catalog
			}

			if tt.builtIn {
				o.CatalogPath = ""
			}

			if tt.twice {
				_, err := recipes.New(s.opts(dir))
				s.Require().NoError(err)
			}

			made, err := recipes.New(o)

			if tt.err != nil || tt.errText != "" {
				s.Require().Error(err)

				if tt.err != nil {
					s.Require().ErrorIs(err, tt.err)
				}

				if tt.errText != "" {
					s.Require().Contains(err.Error(), tt.errText)
				}

				if tt.absent != "" {
					s.Require().NotContains(err.Error(), tt.absent)
				}

				if tt.atMost > 0 {
					s.Require().LessOrEqual(strings.Count(err.Error(), ","), tt.atMost)
				}

				return
			}

			s.Require().NoError(err)

			all, err := recipes.Load(dir)
			s.Require().NoError(err)
			s.Require().Len(all, 1)
			s.Require().Equal("test-player", all[0].ID)

			if tt.wantAmp != "" {
				s.Require().Equal(tt.wantAmp, rig.GearName(all[0], rig.RoleAmp))
			}

			if tt.wantCab != "" {
				s.Require().Equal(tt.wantCab, rig.GearName(all[0], rig.RoleCab))
			}

			if tt.wantFirst != "" {
				s.Require().Equal(tt.wantFirst, all[0].Chain[0].Gear)
			}

			for i, want := range tt.wantRoles {
				s.Require().Equal(want, all[0].Chain[i].Role)
			}

			// What was written, from the answer rather than from the
			// options it was asked for.
			said := strings.Join(append(
				[]string{made.ID, made.Name, made.Amp, made.Cab, made.Path},
				made.Pedals...), " ")

			for _, want := range tt.says {
				s.Require().Contains(said, want)
			}
		})
	}
}

func TestNewPublicTestSuite(t *testing.T) {
	suite.Run(t, new(NewPublicTestSuite))
}
