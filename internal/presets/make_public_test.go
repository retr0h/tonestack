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
package presets_test

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/internal/presets"
	"github.com/retr0h/tonestack/internal/recipes"
	"github.com/retr0h/tonestack/internal/resolve"
	"github.com/retr0h/tonestack/pkg/preset"
)

type MakePublicTestSuite struct {
	suite.Suite
}

func (s *MakePublicTestSuite) opts(id, out string) presets.MakeOptions {
	return presets.MakeOptions{
		RecipeID:    id,
		RecipesDir:  filepath.Join("testdata", "recipes"),
		CatalogPath: filepath.Join("testdata", "catalog.json"),
		OutputPath:  out,
	}
}

func (s *MakePublicTestSuite) TestMakeWritesALoadablePreset() {
	out := filepath.Join(s.T().TempDir(), "test.hlx")

	var log bytes.Buffer
	s.Require().NoError(presets.Make(&log, s.opts("test-player", out)))

	f, err := os.Open(out) //nolint:gosec // a path this test chose
	s.Require().NoError(err)

	defer func() { s.Require().NoError(f.Close()) }()

	doc, err := preset.Read(f)
	s.Require().NoError(err)
	s.Require().Equal(2162694, doc.Data.Device)
	s.Require().Equal("Test Player", doc.Data.Meta.Name)

	spec, err := doc.Spec()
	s.Require().NoError(err)
	s.Require().Len(spec.Blocks, 2)
}

func (s *MakePublicTestSuite) TestMakeReportsWhatItChose() {
	out := filepath.Join(s.T().TempDir(), "test.hlx")

	var log bytes.Buffer
	s.Require().NoError(presets.Make(&log, s.opts("test-player", out)))

	got := log.String()
	s.Require().Contains(got, "Test Player")
	s.Require().Contains(got, "Ampeg SVT",
		"the real gear must be named, not only the model identifier")
	s.Require().Contains(got, "dsp0",
		"the processor budget must be shown, because it is the constraint")
	s.Require().Contains(got, out)
}

func (s *MakePublicTestSuite) TestMakeExplainsWhatItAddedUnasked() {
	// A recipe names an amp; a rig is several blocks. Whatever the corpus
	// contributed has to be visible before anybody plugs in.
	out := filepath.Join(s.T().TempDir(), "test.hlx")

	o := s.opts("test-player", out)
	o.StatsPath = filepath.Join("testdata", "stats.json.gz")

	var log bytes.Buffer
	s.Require().NoError(presets.Make(&log, o))

	got := log.String()
	s.Require().Contains(got, "added")
	s.Require().Contains(got, "Minotaur")
	s.Require().Contains(got, "95% of chains",
		"an unasked-for block must say how common it is")
}

func (s *MakePublicTestSuite) TestMakeReportsAWriterThatFailsOnTheExplanation() {
	o := s.opts("test-player", filepath.Join(s.T().TempDir(), "test.hlx"))
	o.StatsPath = filepath.Join("testdata", "stats.json.gz")

	// The explanation is written after the chain and the budget, so a writer
	// failing there must still surface.
	for _, after := range []int{5, 6, 7, 8} {
		s.Run(fmt.Sprintf("after %d writes", after), func() {
			s.Require().Error(presets.Make(&failingWriter{ok: after}, o))
		})
	}
}

func (s *MakePublicTestSuite) TestMakeCarriesOnWithoutStatistics() {
	// Statistics improve a preset; they are not required to produce one.
	o := s.opts("test-player", filepath.Join(s.T().TempDir(), "test.hlx"))
	o.StatsPath = filepath.Join("testdata", "no-such-stats.gz")

	s.Require().NoError(presets.Make(&bytes.Buffer{}, o))
}

func (s *MakePublicTestSuite) TestMakeReportsAnUnknownRecipe() {
	err := presets.Make(&bytes.Buffer{},
		s.opts("nobody", filepath.Join(s.T().TempDir(), "x.hlx")))

	s.Require().ErrorIs(err, recipes.ErrNotFound)
}

func (s *MakePublicTestSuite) TestMakeReportsAnUnreadableCatalog() {
	o := s.opts("test-player", filepath.Join(s.T().TempDir(), "x.hlx"))
	o.CatalogPath = "testdata/nope.json"

	s.Require().Error(presets.Make(&bytes.Buffer{}, o))
}

func (s *MakePublicTestSuite) TestMakeReportsGearItCannotResolve() {
	err := presets.Make(&bytes.Buffer{},
		s.opts("unbuildable", filepath.Join(s.T().TempDir(), "x.hlx")))

	s.Require().ErrorIs(err, resolve.ErrNoSuchGear)
}

func (s *MakePublicTestSuite) TestMakeRefusesAChainThatWillNotLoad() {
	// Seven heavy pedals plus an amp and a cabinet exceeds what the device can
	// hold, and a preset nobody can load is not a preset.
	err := presets.Make(&bytes.Buffer{},
		s.opts("too-big", filepath.Join(s.T().TempDir(), "x.hlx")))

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "will not load")
}

func (s *MakePublicTestSuite) TestMakeReportsAnUnwritableDestination() {
	err := presets.Make(&bytes.Buffer{},
		s.opts("test-player", filepath.Join(s.T().TempDir(), "no", "such", "dir.hlx")))

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "writing")
}

func (s *MakePublicTestSuite) TestMakeReportsAFailingWriter() {
	// The report is written in three parts, and a writer that fails part way
	// through must still be reported rather than leaving a half-written
	// summary and a success.
	for _, after := range []int{0, 1, 4} {
		s.Run(fmt.Sprintf("after %d writes", after), func() {
			out := filepath.Join(s.T().TempDir(), "test.hlx")

			err := presets.Make(&failingWriter{ok: after}, s.opts("test-player", out))

			s.Require().Error(err)
			s.Require().Contains(err.Error(), "reporting")
		})
	}
}

// failingWriter fails once it has accepted ok writes.
type failingWriter struct {
	ok int
	n  int
}

func (w *failingWriter) Write(p []byte) (int, error) {
	w.n++
	if w.n > w.ok {
		return 0, errors.New("boom")
	}

	return len(p), nil
}

func TestMakePublicTestSuite(t *testing.T) {
	suite.Run(t, new(MakePublicTestSuite))
}
