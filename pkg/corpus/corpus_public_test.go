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

package corpus_test

import (
	"bytes"
	"compress/gzip"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/catalog"
	"github.com/retr0h/tonestack/pkg/corpus"
)

type CorpusPublicTestSuite struct {
	suite.Suite
}

func (s *CorpusPublicTestSuite) TestBuiltInIsUsable() {
	st, err := corpus.BuiltIn()

	s.Require().NoError(err)
	s.Require().Greater(st.Presets, 1000, "the shipped measurement is a real one")
	s.Require().NotEmpty(st.Models)
	s.Require().Contains(st.Grammar, "bass")
	s.Require().Contains(st.Grammar, "guitar")
}

func (s *CorpusPublicTestSuite) TestBuiltInMeasuresAKnownAmp() {
	st, err := corpus.BuiltIn()
	s.Require().NoError(err)

	p, ok := st.Param("HD2_AmpSVBeastNrm", "Treble")

	s.Require().True(ok)
	s.Require().Positive(p.N)
	s.Require().Positive(p.Median)
}

func (s *CorpusPublicTestSuite) TestParamReportsWhatWasNeverMeasured() {
	st, err := corpus.BuiltIn()
	s.Require().NoError(err)

	tests := []struct {
		name  string
		model string
		key   string
	}{
		{"a model nobody used", "HD2_NoSuchModel", "Drive"},
		{"a parameter that model does not have", "HD2_AmpSVBeastNrm", "Nonsense"},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			_, ok := st.Param(catalog.ModelID(tc.model), tc.key)

			s.Require().False(ok)
		})
	}
}

func (s *CorpusPublicTestSuite) TestSpreadIsTheInterquartileRange() {
	s.Require().InDelta(0.3,
		corpus.ParamStats{P25: 0.2, P75: 0.5}.Spread(), 1e-9)
}

func (s *CorpusPublicTestSuite) TestFrequency() {
	c := corpus.CategoryStats{Chains: 89}

	s.Require().InDelta(0.89, c.Frequency(100), 1e-9)
	s.Require().Zero(c.Frequency(0), "nothing measured is not a frequency of one")
}

func (s *CorpusPublicTestSuite) TestBeforeAmp() {
	s.Require().InDelta(0.8,
		corpus.CategoryStats{Before: 8, After: 2}.BeforeAmp(), 1e-9)
	s.Require().Zero(corpus.CategoryStats{}.BeforeAmp(),
		"a category nobody used sits nowhere")
}

func (s *CorpusPublicTestSuite) TestLoadRejectsWhatIsNotStatistics() {
	_, err := corpus.Load(strings.NewReader("{ not json"))

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "decoding corpus statistics")
}

func (s *CorpusPublicTestSuite) TestLoadAcceptsPlainJSONAndGzip() {
	// The generator writes gzip because the file is embedded, but somebody
	// inspecting a copy will have plain JSON. Both are statistics.
	plain := `{"device":"HX Stomp","presets":3,"models":{},"grammar":{}}`

	got, err := corpus.Load(strings.NewReader(plain))
	s.Require().NoError(err)
	s.Require().Equal(3, got.Presets)

	var packed bytes.Buffer

	zw := gzip.NewWriter(&packed)
	_, err = zw.Write([]byte(plain))
	s.Require().NoError(err)
	s.Require().NoError(zw.Close())

	got, err = corpus.Load(&packed)
	s.Require().NoError(err)
	s.Require().Equal(3, got.Presets)
}

func (s *CorpusPublicTestSuite) TestDecodeRejectsWhatIsNotStatistics() {
	tests := []struct {
		name   string
		packed []byte
		want   string
	}{
		{"bytes that are neither", []byte("not gzip at all"), "decoding"},
		{
			"a gzip header with nothing after it",
			[]byte{0x1f, 0x8b, 0x08, 0, 0, 0, 0, 0, 0, 0xff},
			"decoding",
		},
		{"a truncated gzip header", []byte{0x1f, 0x8b}, "opening"},
		{"nothing at all", nil, "decoding"},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			_, err := corpus.Decode(tc.packed)

			s.Require().Error(err)
			s.Require().Contains(err.Error(), tc.want)
		})
	}
}

func (s *CorpusPublicTestSuite) TestLoadReportsAReaderThatFails() {
	_, err := corpus.Load(&failingReader{})

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "reading corpus statistics")
}

// failingReader fails outright rather than reaching the end of its input, so
// the difference between "nothing there" and "could not read" is exercised.
type failingReader struct{}

func (*failingReader) Read([]byte) (int, error) { return 0, errors.New("boom") }

func TestCorpusPublicTestSuite(t *testing.T) {
	suite.Run(t, new(CorpusPublicTestSuite))
}
