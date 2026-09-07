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
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/internal/recipes"
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

func (s *NewPublicTestSuite) TestWritesARecipeThatLoads() {
	dir := s.T().TempDir()

	var log bytes.Buffer
	s.Require().NoError(recipes.New(&log, s.opts(dir)))

	// The point of scaffolding is a file the loader accepts, not a template.
	all, err := recipes.Load(dir)
	s.Require().NoError(err)
	s.Require().Len(all, 1)
	s.Require().Equal("test-player", all[0].ID)
	s.Require().Equal("Ampeg SVT", all[0].Rig.Amp)

	s.Require().Contains(log.String(), "Test Player")
	s.Require().Contains(log.String(), "presets make")
}

func (s *NewPublicTestSuite) TestWritesEverythingItWasGiven() {
	dir := s.T().TempDir()

	o := s.opts(dir)
	o.Band = "A Band"
	o.Cab = "Ampeg SVT 410HLF"
	o.Pedals = []string{"Klon Centaur"}

	var log bytes.Buffer
	s.Require().NoError(recipes.New(&log, o))

	all, err := recipes.Load(dir)
	s.Require().NoError(err)
	s.Require().Equal("Ampeg SVT 410HLF", *all[0].Rig.Cab)
	s.Require().Equal([]string{"Klon Centaur"}, *all[0].Rig.Pedals)
	s.Require().Contains(log.String(), "Klon Centaur")
}

func (s *NewPublicTestSuite) TestRefusesGearNoDeviceModels() {
	// Checking before writing is the point. A recipe naming an amplifier
	// nothing models is otherwise only discovered at build time, by which
	// point the name has usually been copied somewhere else too.
	o := s.opts(s.T().TempDir())
	o.Amp = "Ampeg SVQ"

	err := recipes.New(&bytes.Buffer{}, o)

	s.Require().ErrorIs(err, recipes.ErrNoSuchGear)
	s.Require().Contains(err.Error(), "did you mean")
	s.Require().Contains(err.Error(), "Ampeg SVT")
}

func (s *NewPublicTestSuite) TestRefusesGearInAnyPosition() {
	tests := []struct {
		name   string
		mutate func(*recipes.NewOptions)
	}{
		{"a cabinet", func(o *recipes.NewOptions) { o.Cab = "Nonesuch 9x9" }},
		{"a pedal", func(o *recipes.NewOptions) { o.Pedals = []string{"Nonesuch Fuzz"} }},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			o := s.opts(s.T().TempDir())
			tc.mutate(&o)

			s.Require().ErrorIs(recipes.New(&bytes.Buffer{}, o), recipes.ErrNoSuchGear)
		})
	}
}

func (s *NewPublicTestSuite) TestSuggestsNothingWhenNothingIsClose() {
	o := s.opts(s.T().TempDir())
	o.Amp = "Zzz"

	err := recipes.New(&bytes.Buffer{}, o)

	s.Require().ErrorIs(err, recipes.ErrNoSuchGear)
	s.Require().NotContains(err.Error(), "did you mean",
		"a weak suggestion is worse than none when somebody is deciding "+
			"whether they got a name wrong")
}

func (s *NewPublicTestSuite) TestRefusesAnIdentifierThatIsNotOne() {
	tests := []string{"Test Player", "test_player", "", "-leading", "trailing-"}

	for _, id := range tests {
		s.Run(id, func() {
			o := s.opts(s.T().TempDir())
			o.ID = id

			err := recipes.New(&bytes.Buffer{}, o)

			s.Require().ErrorIs(err, recipes.ErrBadID)
			s.Require().Contains(err.Error(), "hyphens")
		})
	}
}

func (s *NewPublicTestSuite) TestRefusesToOverwrite() {
	dir := s.T().TempDir()
	s.Require().NoError(recipes.New(&bytes.Buffer{}, s.opts(dir)))

	err := recipes.New(&bytes.Buffer{}, s.opts(dir))

	s.Require().ErrorIs(err, recipes.ErrExists)
	s.Require().Contains(err.Error(), "test-player.yaml")
}

func (s *NewPublicTestSuite) TestSuggestsAtMostAHandful() {
	// Against the real catalog "ampeg" matches far more than anybody wants
	// listed in an error message.
	o := s.opts(s.T().TempDir())
	o.CatalogPath = ""
	o.Amp = "Ampeg Nonesuch"

	err := recipes.New(&bytes.Buffer{}, o)

	s.Require().ErrorIs(err, recipes.ErrNoSuchGear)
	s.Require().LessOrEqual(strings.Count(err.Error(), ","), 5)
}

func (s *NewPublicTestSuite) TestReportsADirectoryItCannotWriteInto() {
	dir := s.T().TempDir()
	artists := filepath.Join(dir, "artists")
	s.Require().NoError(os.MkdirAll(artists, 0o750))
	s.Require().NoError(os.Chmod(artists, 0o500))

	defer func() { s.Require().NoError(os.Chmod(artists, 0o750)) }()

	err := recipes.New(&bytes.Buffer{}, s.opts(dir))

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "writing")
}

func (s *NewPublicTestSuite) TestReportsProblems() {
	tests := []struct {
		name    string
		mutate  func(*recipes.NewOptions)
		message string
	}{
		{
			"a catalog that is not there",
			func(o *recipes.NewOptions) {
				o.CatalogPath = filepath.Join("testdata", "nope.json")
			},
			"catalog",
		},
		{
			"a directory that cannot be made",
			func(o *recipes.NewOptions) { o.Dir = s.file() },
			"making room",
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			o := s.opts(s.T().TempDir())
			tc.mutate(&o)

			err := recipes.New(&bytes.Buffer{}, o)

			s.Require().Error(err)
			s.Require().Contains(err.Error(), tc.message)
		})
	}
}

func (s *NewPublicTestSuite) TestReportsAWriterThatFails() {
	s.Require().Error(recipes.New(&failingWriter{}, s.opts(s.T().TempDir())))
}

// file returns a path that is a file, so making a directory under it fails.
func (s *NewPublicTestSuite) file() string {
	path := filepath.Join(s.T().TempDir(), "not-a-directory")
	s.Require().NoError(os.WriteFile(path, []byte("x"), 0o600))

	return path
}

func TestNewPublicTestSuite(t *testing.T) {
	suite.Run(t, new(NewPublicTestSuite))
}
