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
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/catalog"
	"github.com/retr0h/tonestack/pkg/corpus"
)

type CorpusPublicTestSuite struct {
	suite.Suite
}

// TestBuiltIn covers the measurement this binary ships.
func (s *CorpusPublicTestSuite) TestBuiltIn() {
	got, err := corpus.BuiltIn()

	s.Require().NoError(err)
	s.Require().Greater(got.Presets, 1000, "the shipped measurement is a real one")
	s.Require().NotEmpty(got.Models)
	s.Require().Contains(got.Grammar, "bass")
	s.Require().Contains(got.Grammar, "guitar")
}

// TestParam reads what the corpus measured about one knob.
func (s *CorpusPublicTestSuite) TestParam() {
	tests := []struct {
		name  string
		model catalog.ModelID
		key   string
		ok    bool
	}{
		{
			name:  "a parameter of an amp people use",
			model: "HD2_AmpSVBeastNrm",
			key:   "Treble",
			ok:    true,
		},
		{name: "a model nobody used", model: "HD2_NoSuchModel", key: "Drive"},
		{
			name:  "a parameter that model does not have",
			model: "HD2_AmpSVBeastNrm",
			key:   "Nonsense",
		},
	}

	st, err := corpus.BuiltIn()
	s.Require().NoError(err)

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, ok := st.Param(tt.model, tt.key)

			s.Require().Equal(tt.ok, ok)

			if !tt.ok {
				return
			}

			s.Require().Positive(got.N)
			s.Require().Positive(got.Median)
		})
	}
}

// TestSpread is the interquartile range: how far apart people set a knob.
func (s *CorpusPublicTestSuite) TestSpread() {
	tests := []struct {
		name string
		in   corpus.ParamStats
		want float64
	}{
		{
			name: "a quarter either side of the middle",
			in:   corpus.ParamStats{P25: 0.2, P75: 0.5},
			want: 0.3,
		},
		{name: "a knob nobody moved", in: corpus.ParamStats{P25: 0.5, P75: 0.5}},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().InDelta(tt.want, tt.in.Spread(), 1e-9)
		})
	}
}

// TestFrequency is the share of chains a category appears in.
func (s *CorpusPublicTestSuite) TestFrequency() {
	tests := []struct {
		name   string
		in     corpus.CategoryStats
		chains int
		want   float64
	}{
		{
			name:   "a category most chains have",
			in:     corpus.CategoryStats{Chains: 89},
			chains: 100,
			want:   0.89,
		},
		{
			// Nothing measured is not a frequency of one.
			name: "no chains measured at all",
			in:   corpus.CategoryStats{Chains: 89},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().InDelta(tt.want, tt.in.Frequency(tt.chains), 1e-9)
		})
	}
}

// TestBeforeAmp is the share of uses that sit ahead of the amplifier.
func (s *CorpusPublicTestSuite) TestBeforeAmp() {
	tests := []struct {
		name string
		in   corpus.CategoryStats
		want float64
	}{
		{
			name: "a category usually ahead of it",
			in:   corpus.CategoryStats{Before: 8, After: 2},
			want: 0.8,
		},
		{name: "a category nobody used, which sits nowhere"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().InDelta(tt.want, tt.in.BeforeAmp(), 1e-9)
		})
	}
}

// TestLoad reads statistics off a reader.
func (s *CorpusPublicTestSuite) TestLoad() {
	// The generator writes gzip because the file is embedded, but somebody
	// inspecting a copy will have plain JSON. Both are statistics.
	const plain = `{"device":"HX Stomp","presets":3,"models":{},"grammar":{}}`

	tests := []struct {
		name    string
		in      string
		packed  bool
		deaf    bool
		want    int
		errText string
	}{
		{name: "plain JSON", in: plain, want: 3},
		{name: "the same, gzipped", in: plain, packed: true, want: 3},
		{
			name:    "something that is not statistics",
			in:      "{ not json",
			errText: "decoding corpus statistics",
		},
		{
			name:    "a reader that fails",
			deaf:    true,
			errText: "reading corpus statistics",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			in := io.Reader(strings.NewReader(tt.in))

			if tt.packed {
				var buf bytes.Buffer

				zw := gzip.NewWriter(&buf)
				_, err := zw.Write([]byte(tt.in))
				s.Require().NoError(err)
				s.Require().NoError(zw.Close())

				in = &buf
			}

			if tt.deaf {
				in = &failingReader{}
			}

			got, err := corpus.Load(in)

			if tt.errText != "" {
				s.Require().Error(err)
				s.Require().Contains(err.Error(), tt.errText)

				return
			}

			s.Require().NoError(err)
			s.Require().Equal(tt.want, got.Presets)
		})
	}
}

// TestDecode reads statistics out of bytes already in hand.
func (s *CorpusPublicTestSuite) TestDecode() {
	tests := []struct {
		name    string
		packed  []byte
		errText string
	}{
		{
			name:    "bytes that are neither JSON nor gzip",
			packed:  []byte("not gzip at all"),
			errText: "decoding",
		},
		{
			name:    "a gzip header with nothing after it",
			packed:  []byte{0x1f, 0x8b, 0x08, 0, 0, 0, 0, 0, 0, 0xff},
			errText: "decoding",
		},
		{
			name:    "a truncated gzip header",
			packed:  []byte{0x1f, 0x8b},
			errText: "opening",
		},
		{name: "nothing at all", errText: "decoding"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			_, err := corpus.Decode(tt.packed)

			s.Require().Error(err)
			s.Require().Contains(err.Error(), tt.errText)
		})
	}
}

// failingReader fails outright rather than reaching the end of its input, so
// the difference between "nothing there" and "could not read" is exercised.
type failingReader struct{}

func (*failingReader) Read([]byte) (int, error) { return 0, errors.New("boom") }

func TestCorpusPublicTestSuite(t *testing.T) {
	suite.Run(t, new(CorpusPublicTestSuite))
}
