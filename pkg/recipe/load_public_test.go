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
package recipe_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/recipe"
)

type LoadPublicTestSuite struct {
	suite.Suite
}

const valid = `
id: mike-dirnt
kind: artist
name: Mike Dirnt
instrument_type: bass
rig:
  amp: Ampeg SVT
provenance:
  source: llm
  confidence: medium
`

func (s *LoadPublicTestSuite) TestLoadReadsAValidRecipe() {
	r, err := recipe.Load(strings.NewReader(valid))

	s.Require().NoError(err)
	s.Require().Equal("mike-dirnt", r.ID)
	s.Require().Equal(recipe.KindArtist, r.Kind)
	s.Require().Equal(recipe.InstrumentBass, r.InstrumentType)
	s.Require().Equal("Ampeg SVT", r.Rig.Amp)
}

func (s *LoadPublicTestSuite) TestLoadRejectsMalformedYAML() {
	_, err := recipe.Load(strings.NewReader("id: [unclosed"))

	s.Require().Error(err)
}

func (s *LoadPublicTestSuite) TestLoadRejectsWhatTheTypeSystemCannot() {
	tests := map[string]string{
		"bad id":       strings.Replace(valid, "id: mike-dirnt", "id: Mike_Dirnt", 1),
		"empty name":   strings.Replace(valid, "name: Mike Dirnt", `name: ""`, 1),
		"unknown kind": strings.Replace(valid, "kind: artist", "kind: banjo", 1),
		"bad instrument": strings.Replace(
			valid,
			"instrument_type: bass",
			"instrument_type: kazoo",
			1,
		),
		"no amp":         strings.Replace(valid, "  amp: Ampeg SVT", "  cab: 8x10", 1),
		"bad source":     strings.Replace(valid, "source: llm", "source: vibes", 1),
		"bad confidence": strings.Replace(valid, "confidence: medium", "confidence: total", 1),
	}

	for name, doc := range tests {
		s.Run(name, func() {
			_, err := recipe.Load(strings.NewReader(doc))
			s.Require().ErrorIs(err, recipe.ErrInvalid)
		})
	}
}

func (s *LoadPublicTestSuite) TestLoadRejectsABadVariant() {
	for _, bad := range []string{
		valid + "variants:\n  - id: Long_View\n    name: Longview\n",
		valid + "variants:\n  - id: longview\n    name: \"\"\n",
	} {
		_, err := recipe.Load(strings.NewReader(bad))
		s.Require().ErrorIs(err, recipe.ErrInvalid)
	}
}

func (s *LoadPublicTestSuite) TestLoadAcceptsAGoodVariant() {
	r, err := recipe.Load(strings.NewReader(
		valid + "variants:\n  - id: longview\n    name: Longview\n"))

	s.Require().NoError(err)
	s.Require().Len(r.Variants, 1)
}

func (s *LoadPublicTestSuite) TestLoadReportsAFailingReader() {
	_, err := recipe.Load(&failingReader{})

	s.Require().Error(err)
}

func (s *LoadPublicTestSuite) TestTrustedDistinguishesConfirmedKnowledge() {
	s.Require().False(recipe.Provenance{Source: recipe.SourceLLM}.Trusted())
	s.Require().True(recipe.Provenance{Source: recipe.SourceCurated}.Trusted())
	s.Require().True(recipe.Provenance{Source: recipe.SourceCited}.Trusted())
}

type failingReader struct{}

func (*failingReader) Read([]byte) (int, error) { return 0, errBoom }

func TestLoadPublicTestSuite(t *testing.T) {
	suite.Run(t, new(LoadPublicTestSuite))
}
