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
package catalogview_test

import (
	"bytes"
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/internal/catalogview"
)

type CatalogViewPublicTestSuite struct {
	suite.Suite
}

func (s *CatalogViewPublicTestSuite) path() string {
	return filepath.Join("testdata", "catalog.json")
}

func (s *CatalogViewPublicTestSuite) TestOpenReadsACatalog() {
	c, err := catalogview.Open(s.path())

	s.Require().NoError(err)
	s.Require().Equal("HX Stomp", c.Device)
	s.Require().Len(c.Blocks, 2)
}

func (s *CatalogViewPublicTestSuite) TestOpenReportsAMissingFile() {
	_, err := catalogview.Open("testdata/nope.json")

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "opening catalog")
}

func (s *CatalogViewPublicTestSuite) TestOpenReportsAMalformedFile() {
	_, err := catalogview.Open(filepath.Join("testdata", "bad.json"))

	s.Require().Error(err)
}

func (s *CatalogViewPublicTestSuite) TestListShowsEveryBlock() {
	var out bytes.Buffer

	s.Require().NoError(catalogview.List(&out, s.path(), catalogview.Filter{}))
	s.Require().Contains(out.String(), "HD2_AmpTestBass")
	s.Require().Contains(out.String(), "HD2_DriveTest")
	s.Require().Contains(out.String(), "2 of 2 blocks")
}

func (s *CatalogViewPublicTestSuite) TestListFiltersByCategory() {
	var out bytes.Buffer

	s.Require().NoError(catalogview.List(&out, s.path(),
		catalogview.Filter{Category: "amp"}))
	s.Require().Contains(out.String(), "HD2_AmpTestBass")
	s.Require().NotContains(out.String(), "HD2_DriveTest")
	s.Require().Contains(out.String(), "1 of 2 blocks")
}

func (s *CatalogViewPublicTestSuite) TestListFiltersBySubcategory() {
	var out bytes.Buffer

	// This is the filter that makes a bass request draw from bass amps.
	s.Require().NoError(catalogview.List(&out, s.path(),
		catalogview.Filter{Subcategory: "bass"}))
	s.Require().Contains(out.String(), "HD2_AmpTestBass")
	s.Require().NotContains(out.String(), "HD2_DriveTest")
}

func (s *CatalogViewPublicTestSuite) TestListSearchesNameGearAndIdentifier() {
	for _, term := range []string{"Test Bass", "ampeg", "SVBeast", "hd2_amptestbass"} {
		var out bytes.Buffer

		s.Require().NoError(catalogview.List(&out, s.path(),
			catalogview.Filter{Search: term}))

		if term == "SVBeast" {
			s.Require().Contains(out.String(), "no blocks match")

			continue
		}

		s.Require().Contains(out.String(), "HD2_AmpTestBass", "searching %q", term)
	}
}

func (s *CatalogViewPublicTestSuite) TestListSaysSoWhenNothingMatches() {
	var out bytes.Buffer

	s.Require().NoError(catalogview.List(&out, s.path(),
		catalogview.Filter{Category: "looper"}))
	s.Require().Contains(out.String(), "no blocks match")
}

func (s *CatalogViewPublicTestSuite) TestListPropagatesAnUnreadableCatalog() {
	s.Require().Error(catalogview.List(&bytes.Buffer{}, "nope.json", catalogview.Filter{}))
}

func (s *CatalogViewPublicTestSuite) TestListReportsAFailingWriter() {
	s.Require().Error(catalogview.List(&failingWriter{}, s.path(), catalogview.Filter{}))
	s.Require().Error(catalogview.List(&failingWriter{}, s.path(),
		catalogview.Filter{Category: "looper"}))
}

func (s *CatalogViewPublicTestSuite) TestShowRendersParametersWithRanges() {
	var out bytes.Buffer

	s.Require().NoError(catalogview.Show(&out, s.path(), "HD2_AmpTestBass"))

	got := out.String()
	s.Require().Contains(got, "Ampeg SVT (normal channel)")
	s.Require().Contains(got, "amp (Bass)")
	s.Require().Contains(got, "26.67 mono")
	s.Require().Contains(got, "40.10 stereo")
	s.Require().Contains(got, "Drive")
	s.Require().Contains(got, "0..1")
	s.Require().Contains(got, "0.53")
	s.Require().Contains(got, "Bright")
	s.Require().Contains(got, "—", "a bool has no range")
}

func (s *CatalogViewPublicTestSuite) TestShowOmitsAbsentDetail() {
	var out bytes.Buffer

	s.Require().NoError(catalogview.Show(&out, s.path(), "HD2_DriveTest"))
	s.Require().NotContains(out.String(), "stereo",
		"a block with no stereo cost should not claim one")
}

func (s *CatalogViewPublicTestSuite) TestShowReportsAnUnknownBlock() {
	err := catalogview.Show(&bytes.Buffer{}, s.path(), "HD2_Nope")

	s.Require().ErrorIs(err, catalogview.ErrNotFound)
	s.Require().Contains(err.Error(), "catalog list")
}

func (s *CatalogViewPublicTestSuite) TestShowPropagatesAnUnreadableCatalog() {
	s.Require().Error(catalogview.Show(&bytes.Buffer{}, "nope.json", "x"))
}

func (s *CatalogViewPublicTestSuite) TestShowReportsAFailingWriter() {
	s.Require().Error(catalogview.Show(&failingWriter{}, s.path(), "HD2_AmpTestBass"))
	s.Require().Error(catalogview.Show(&failAfter{n: 1}, s.path(), "HD2_AmpTestBass"))
}

func (s *CatalogViewPublicTestSuite) TestNotFoundErrorNamesWhatWasAsked() {
	err := &catalogview.NotFoundError{ID: "HD2_Nope", Known: 665}

	s.Require().Contains(err.Error(), "HD2_Nope")
	s.Require().Contains(err.Error(), "665")
	s.Require().ErrorIs(err, catalogview.ErrNotFound)
}

func (s *CatalogViewPublicTestSuite) TestDefaultPathIsWhereTheCatalogLives() {
	s.Require().Equal("schemas/hx-stomp.catalog.json", catalogview.DefaultPath)
}

type failingWriter struct{}

func (*failingWriter) Write([]byte) (int, error) { return 0, errors.New("boom") }

// failAfter fails on the nth write, so a later reporting step can be reached.
type failAfter struct{ n int }

func (f *failAfter) Write(p []byte) (int, error) {
	if f.n == 0 {
		return 0, errors.New("boom")
	}

	f.n--

	return len(p), nil
}

func (s *CatalogViewPublicTestSuite) TestOpenFallsBackToTheBuiltInCatalog() {
	// No path is the case for anyone who has not generated their own, which
	// is everyone who installed a binary.
	c, err := catalogview.Open("")

	s.Require().NoError(err)
	s.Require().NotEmpty(c.Blocks)
	s.Require().NotEmpty(c.Source)
}

func (s *CatalogViewPublicTestSuite) TestListNamesWhereTheCatalogCameFrom() {
	var out bytes.Buffer

	s.Require().NoError(catalogview.List(&out, "", catalogview.Filter{
		Search: "klon",
	}))

	s.Require().Contains(out.String(), "HX Edit",
		"a catalog is only true of the release it came from, so it says which")
}

func (s *CatalogViewPublicTestSuite) TestListFlagsACatalogThatCannotNameItsSource() {
	var out bytes.Buffer

	s.Require().NoError(catalogview.List(&out, s.path(), catalogview.Filter{}))

	s.Require().Contains(out.String(), "source unknown")
}

func (s *CatalogViewPublicTestSuite) TestShowMarksAFigureNobodyStated() {
	var out bytes.Buffer

	s.Require().NoError(catalogview.Show(
		&out, filepath.Join("testdata", "assumed.json"), "HD2_Guessed"))

	s.Require().Contains(out.String(), "assumed",
		"a DSP cost that was inferred must not read as Line 6's own figure")
}

func TestCatalogViewPublicTestSuite(t *testing.T) {
	suite.Run(t, new(CatalogViewPublicTestSuite))
}
