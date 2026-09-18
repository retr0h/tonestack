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
	"github.com/retr0h/tonestack/pkg/sdk/result"
)

// BackingPublicTestSuite covers holding a rig's records to the era it claims.
type BackingPublicTestSuite struct {
	suite.Suite
}

// read joins the fixture rigs to the fixture corpus.
func (s *BackingPublicTestSuite) read() map[string]result.Backing {
	got, err := recipes.Backing(
		recipes.Source{Dir: filepath.Join("testdata", "backing", "rigs")},
		filepath.Join("testdata", "backing", "music"),
	)
	s.Require().NoError(err)

	out := map[string]result.Backing{}
	for _, b := range got {
		out[b.ID] = b
	}

	return out
}

// TestRecordsInsideTheEra covers the case nobody has to act on.
func (s *BackingPublicTestSuite) TestRecordsInsideTheEra() {
	got := s.read()["in-era"]

	s.Require().True(got.Stated())
	s.Require().Len(got.Records, 2)
	s.Require().Zero(got.Outside())

	for _, r := range got.Records {
		s.Require().False(r.Outside)
	}
}

// TestARecordOutsideTheEra covers the finding this exists for.
//
// The rig describes 2004 and one record is from 1994, so the figures measured
// from it describe gear the rig does not name.
func (s *BackingPublicTestSuite) TestARecordOutsideTheEra() {
	got := s.read()["out-of-era"]

	s.Require().Equal(1, got.Outside())
	s.Require().True(got.Records[0].Outside, "1994 against a 2004 rig")
	s.Require().False(got.Records[1].Outside, "2004 against a 2004 rig")
}

// TestRecordsNoRigIsNamedFor covers the join failing quietly.
//
// A rig reaches its records by the directory carrying its identifier. A
// directory called anything else reads exactly like a rig nobody has measured
// yet, so the typo survives until something says which it is.
func (s *BackingPublicTestSuite) TestRecordsNoRigIsNamedFor() {
	got := s.read()["mccartney"]

	s.Require().True(got.NoRig)
	s.Require().NotEmpty(got.Records, "the records are there, and nobody claims them")
	s.Require().Zero(got.Outside(), "there is no era to be outside of")
}

// TestARigIsNotItsOwnOrphan covers the ordinary directories staying ordinary.
func (s *BackingPublicTestSuite) TestARigIsNotItsOwnOrphan() {
	for _, id := range []string{"in-era", "out-of-era", "no-years"} {
		s.Require().False(s.read()[id].NoRig, id)
	}
}

// TestADirectoryWithNoManifest covers a directory somebody made and has not
// filled, which claims nothing and is nobody's problem.
//
// Built here rather than kept in testdata, because git does not track an
// empty directory: as a fixture this passed locally and never ran anywhere
// else, which is the kind of test that reports coverage it does not have.
func (s *BackingPublicTestSuite) TestADirectoryWithNoManifest() {
	corpus := s.T().TempDir()

	s.Require().NoError(os.MkdirAll(filepath.Join(corpus, "empty-dir"), 0o750))
	s.Require().NoError(os.MkdirAll(filepath.Join(corpus, "mccartney"), 0o750))

	named, err := os.ReadFile(
		filepath.Join("testdata", "backing", "music", "mccartney", "corpus.yaml"))
	s.Require().NoError(err)
	s.Require().NoError(os.WriteFile(
		filepath.Join(corpus, "mccartney", "corpus.yaml"), named, 0o600))

	got, err := recipes.Backing(
		recipes.Source{Dir: filepath.Join("testdata", "backing", "rigs")}, corpus)
	s.Require().NoError(err)

	seen := map[string]bool{}
	for _, b := range got {
		seen[b.ID] = true
	}

	s.Require().True(seen["mccartney"], "records nobody's rig is named for")
	s.Require().False(seen["empty-dir"], "nothing in it to report")
}

// TestAnOrphanManifestThatWillNotRead covers a directory no rig claims whose
// manifest is broken, which is reported rather than passed over.
func (s *BackingPublicTestSuite) TestAnOrphanManifestThatWillNotRead() {
	_, err := recipes.Backing(
		recipes.Source{Dir: filepath.Join("testdata", "backing", "rigs")},
		filepath.Join("testdata", "backing", "brokenorphan"),
	)

	s.Require().Error(err)
}

// TestACorpusDirectoryThatIsAFile covers the corpus argument naming a file,
// which is caught reading the rigs' own records.
func (s *BackingPublicTestSuite) TestACorpusDirectoryThatIsAFile() {
	_, err := recipes.Backing(
		recipes.Source{Dir: filepath.Join("testdata", "backing", "rigs")},
		filepath.Join("testdata", "backing", "notadir", "in-era"),
	)

	s.Require().Error(err)
}

// TestACorpusThatIsAFileWithNoRigsToRead covers the same argument reaching
// the scan for directories nobody claims, which is the other way in.
func (s *BackingPublicTestSuite) TestACorpusThatIsAFileWithNoRigsToRead() {
	_, err := recipes.Backing(
		recipes.Source{Dir: filepath.Join("testdata", "backing", "notadir")},
		filepath.Join("testdata", "backing", "notadir", "in-era"),
	)

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "in-era")
}

// TestARigThatStatesNoEra covers what cannot be checked.
//
// Reported rather than passed over: a rig with no years is a rig nothing can
// hold its records to, which is worth seeing beside the ones that can.
func (s *BackingPublicTestSuite) TestARigThatStatesNoEra() {
	got := s.read()["no-years"]

	s.Require().False(got.Stated())
	s.Require().Zero(got.Outside(), "nothing to be outside of")
}

// TestARigNobodyHasMeasured covers the ordinary case.
//
// Most players have gear evidence long before anybody owns their records, so
// a rig with no corpus is not a fault.
func (s *BackingPublicTestSuite) TestARigNobodyHasMeasured() {
	got := s.read()["no-years"]

	s.Require().Empty(got.Records)
}

// TestACorpusThatIsNotThere covers the path being wrong.
//
// A directory nobody has is the same as a player nobody has measured, so it
// reports rather than fails: the rigs still read.
func (s *BackingPublicTestSuite) TestACorpusThatIsNotThere() {
	got, err := recipes.Backing(
		recipes.Source{Dir: filepath.Join("testdata", "backing", "rigs")},
		filepath.Join("testdata", "backing", "nowhere"),
	)

	s.Require().NoError(err)
	s.Require().NotEmpty(got)

	for _, b := range got {
		s.Require().Empty(b.Records)
	}
}

// TestAManifestThatWillNotRead covers a corpus somebody broke.
func (s *BackingPublicTestSuite) TestAManifestThatWillNotRead() {
	_, err := recipes.Backing(
		recipes.Source{Dir: filepath.Join("testdata", "backing", "rigs")},
		filepath.Join("testdata", "backing", "broken"),
	)

	s.Require().Error(err)
}

// TestARecipeDirectoryNobodyHas covers reading a corpus with no rigs to read
// it against.
//
// Not an empty answer: the records are there and nothing claims them, which
// is the same thing as a misspelt directory and reads the same way.
func (s *BackingPublicTestSuite) TestARecipeDirectoryNobodyHas() {
	got, err := recipes.Backing(
		recipes.Source{Dir: filepath.Join("testdata", "nowhere")},
		filepath.Join("testdata", "backing", "music"),
	)

	s.Require().NoError(err)
	s.Require().NotEmpty(got)

	for _, b := range got {
		s.Require().True(b.NoRig, b.ID)
	}
}

// TestACorpusPathThatIsNotADirectory covers a corpus argument naming a file.
func (s *BackingPublicTestSuite) TestACorpusPathThatIsNotADirectory() {
	_, err := recipes.Backing(
		recipes.Source{Dir: filepath.Join("testdata", "backing", "rigs")},
		filepath.Join("testdata", "backing", "notadir"),
	)

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "in-era")
}

// TestARigThatWillNotRead covers a recipe directory holding a broken file.
//
// A half-read knowledge base is worse than a clear complaint about the file
// to fix, which is what Load does, and this reports it rather than answering
// about the rigs that happened to parse.
func (s *BackingPublicTestSuite) TestARigThatWillNotRead() {
	_, err := recipes.Backing(
		recipes.Source{Dir: "testdata"},
		filepath.Join("testdata", "backing", "music"),
	)

	s.Require().Error(err)
}

func TestBackingPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(BackingPublicTestSuite))
}
