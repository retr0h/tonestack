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
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/internal/recipes"
)

type RecipesPublicTestSuite struct {
	suite.Suite
}

func (s *RecipesPublicTestSuite) good() string  { return "testdata-good" }
func (s *RecipesPublicTestSuite) mixed() string { return "testdata" }

func (s *RecipesPublicTestSuite) TestLoadReadsEveryRecipe() {
	all, err := recipes.Load(s.good())

	s.Require().NoError(err)
	s.Require().Len(all, 2)
	s.Require().Equal("mike-dirnt", all[0].ID, "sorted by identifier")
	s.Require().Equal("minimal", all[1].ID)
}

func (s *RecipesPublicTestSuite) TestLoadStopsOnAnInvalidRecipe() {
	// A half-read knowledge base is worse than a complaint about the file to
	// fix, so one bad recipe fails the whole load.
	_, err := recipes.Load(s.mixed())

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "broken.yaml")
}

func (s *RecipesPublicTestSuite) TestLoadOnAnEmptyDirectory() {
	all, err := recipes.Load(s.T().TempDir())

	s.Require().NoError(err)
	s.Require().Empty(all)
}

func (s *RecipesPublicTestSuite) TestFindByIdentifier() {
	r, err := recipes.Find(s.good(), "mike-dirnt")

	s.Require().NoError(err)
	s.Require().Equal("Mike Dirnt", r.Name)
}

func (s *RecipesPublicTestSuite) TestFindIsCaseInsensitive() {
	_, err := recipes.Find(s.good(), "MIKE-DIRNT")

	s.Require().NoError(err)
}

func (s *RecipesPublicTestSuite) TestFindByAlias() {
	r, err := recipes.Find(s.good(), "spare")

	s.Require().NoError(err)
	s.Require().Equal("minimal", r.ID)
}

func (s *RecipesPublicTestSuite) TestShowOmitsWhatIsNotThere() {
	var out bytes.Buffer

	s.Require().NoError(recipes.Show(&out, s.good(), "minimal"))

	got := out.String()
	s.Require().NotContains(got, "character", "nothing to say about how it sounds")
	s.Require().NotContains(got, "variants")
	s.Require().NotContains(got, "band")
	s.Require().NotContains(got, "unverified", "a curated recipe is confirmed")
}

func (s *RecipesPublicTestSuite) TestLoadReportsAnUnreadableFile() {
	dir := s.T().TempDir()
	sub := filepath.Join(dir, "artists")
	s.Require().NoError(os.MkdirAll(sub, 0o750))

	path := filepath.Join(sub, "locked.yaml")
	s.Require().NoError(os.WriteFile(path, []byte("id: locked"), 0o600))
	s.Require().NoError(os.Chmod(path, 0o000))

	_, err := recipes.Load(dir)

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "opening")
}

func (s *RecipesPublicTestSuite) TestFindReportsAnUnknownIdentifier() {
	_, err := recipes.Find(s.good(), "nobody")

	s.Require().ErrorIs(err, recipes.ErrNotFound)
	s.Require().Contains(err.Error(), "recipes list")
}

func (s *RecipesPublicTestSuite) TestFindPropagatesALoadFailure() {
	_, err := recipes.Find(s.mixed(), "mike-dirnt")

	s.Require().Error(err)
}

func (s *RecipesPublicTestSuite) TestListNamesEachRecipe() {
	var out bytes.Buffer

	s.Require().NoError(recipes.List(&out, s.good()))
	s.Require().Contains(out.String(), "mike-dirnt")
	s.Require().Contains(out.String(), "Ampeg SVT")
	s.Require().Contains(out.String(), "llm")
}

func (s *RecipesPublicTestSuite) TestListSaysSoWhenThereAreNone() {
	var out bytes.Buffer
	dir := s.T().TempDir()

	s.Require().NoError(recipes.List(&out, dir))
	s.Require().Contains(out.String(), "no recipes here")
}

func (s *RecipesPublicTestSuite) TestListPropagatesALoadFailure() {
	s.Require().Error(recipes.List(&bytes.Buffer{}, s.mixed()))
}

func (s *RecipesPublicTestSuite) TestListReportsAFailingWriter() {
	s.Require().Error(recipes.List(&failingWriter{}, s.good()))
	s.Require().Error(recipes.List(&failingWriter{}, s.T().TempDir()))
}

func (s *RecipesPublicTestSuite) TestShowRendersEverythingAPersonWrote() {
	var out bytes.Buffer

	s.Require().NoError(recipes.Show(&out, s.good(), "mike-dirnt"))

	got := out.String()
	s.Require().Contains(got, "Mike Dirnt")
	s.Require().Contains(got, "Green Day")
	s.Require().Contains(got, "pick, near the bridge")
	s.Require().Contains(got, "mid-forward")
	s.Require().Contains(got, "Longview")
	s.Require().Contains(got, "unverified",
		"an llm-sourced recipe must say nobody confirmed it")
}

func (s *RecipesPublicTestSuite) TestShowReportsAnUnknownRecipe() {
	s.Require().ErrorIs(
		recipes.Show(&bytes.Buffer{}, s.good(), "nobody"), recipes.ErrNotFound)
}

func (s *RecipesPublicTestSuite) TestShowReportsAFailingWriter() {
	s.Require().Error(recipes.Show(&failingWriter{}, s.good(), "mike-dirnt"))
}

func (s *RecipesPublicTestSuite) TestNotFoundErrorNamesWhatWasAsked() {
	err := &recipes.NotFoundError{ID: "flea", Known: 3}

	s.Require().Contains(err.Error(), "flea")
	s.Require().Contains(err.Error(), "3 known")
	s.Require().ErrorIs(err, recipes.ErrNotFound)
}

func (s *RecipesPublicTestSuite) TestDefaultDirIsWhereRecipesLive() {
	s.Require().Equal("recipes", recipes.DefaultDir)
	s.Require().DirExists(filepath.Join("..", "..", recipes.DefaultDir))
}

type failingWriter struct{}

func (*failingWriter) Write([]byte) (int, error) { return 0, errors.New("boom") }

func (s *RecipesPublicTestSuite) TestLoadFallsBackToTheBuiltInRecipes() {
	// No directory is the case for anyone running an installed binary rather
	// than working in a checkout.
	all, err := recipes.Load("")

	s.Require().NoError(err)
	s.Require().NotEmpty(all, "recipes ship in the binary")
}

func TestRecipesPublicTestSuite(t *testing.T) {
	suite.Run(t, new(RecipesPublicTestSuite))
}
