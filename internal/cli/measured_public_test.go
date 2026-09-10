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

package cli_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/internal/cli"
	"github.com/retr0h/tonestack/pkg/sdk"
	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/corpus"
)

type MeasuredPublicTestSuite struct {
	suite.Suite
}

// stats are measurements shaped to reach every grading band.
//
// Agreement is graded as a share of each parameter's range, so a fixture that
// only ever agrees says nothing about whether the grading works.
func (s *MeasuredPublicTestSuite) stats() *corpus.Stats {
	return &corpus.Stats{
		Device: "HX Stomp", Presets: 721,
		Models: map[catalog.ModelID]corpus.ModelStats{
			"HD2_AmpTestBass": {
				Uses: 40,
				Params: map[string]corpus.ParamStats{
					// Nobody varies it.
					"Drive": {N: 40, Median: 0.440, P25: 0.44, P75: 0.44},
					"Bass":  {N: 40, Median: 0.5, P25: 0.45, P75: 0.55},
					"Mid":   {N: 40, Median: 0.5, P25: 0.4, P75: 0.7},
					// Everybody disagrees.
					"Treble": {N: 40, Median: 0.5, P25: 0.1, P75: 0.9},
					// The catalog does not carry it, so there is no range to
					// grade against.
					"Ghost": {N: 3, Median: 0.5, P25: 0.2, P75: 0.8},
				},
			},
		},
		Grammar: map[string]corpus.Grammar{
			"bass": {
				Chains: 40,
				// Drive and amp are used equally often, so their order is
				// decided by the tiebreak rather than by the count. Reverb
				// is rarer, so the count decides it.
				Categories: map[catalog.Category]corpus.CategoryStats{
					catalog.CategoryDrive:  {Chains: 40, Before: 40},
					catalog.CategoryAmp:    {Chains: 40},
					catalog.CategoryReverb: {Chains: 9, After: 9},
				},
			},
		},
	}
}

// cat carries a range for every parameter but Ghost, which is what makes the
// ungraded case a case rather than an accident.
func (s *MeasuredPublicTestSuite) cat() *catalog.Catalog {
	knob := catalog.Param{Type: catalog.ParamFloat, Min: 0, Max: 1}

	return &catalog.Catalog{Blocks: map[catalog.ModelID]catalog.Block{
		"HD2_AmpTestBass": {
			ID: "HD2_AmpTestBass", Name: "Test Bass Amp",
			Category: catalog.CategoryAmp,
			Params: map[string]catalog.Param{
				"Drive": knob, "Bass": knob, "Mid": knob, "Treble": knob,
			},
		},
	}}
}

// TestMeasured covers what the corpus says, in both of its shapes.
func (s *MeasuredPublicTestSuite) TestMeasured() {
	tests := []struct {
		name string
		in   sdk.Measured
		to   io.Writer
		want []string
		err  bool
	}{
		{
			name: "one model's distributions",
			in: sdk.Measured{
				Stats: s.stats(), Catalog: s.cat(), Model: "HD2_AmpTestBass",
			},
			want: []string{
				// The model is named, not just identified.
				"Test Bass Amp",
				"Drive",
				// The median is what people actually set.
				"0.440",
				// Every band must be reachable, or the grading says nothing.
				"unanimous", "close", "loose", "none",
				// A parameter the catalog does not carry has no range to
				// grade against.
				"Ghost",
			},
		},
		{
			// The corpus measures whatever presets contained; a catalog for
			// one device will not carry all of it. Showing the identifier is
			// more useful than pretending the model does not exist.
			name: "a measured model the catalog does not name",
			in: sdk.Measured{
				Stats:   s.stats(),
				Catalog: &catalog.Catalog{},
				Model:   "HD2_AmpTestBass",
			},
			want: []string{"HD2_AmpTestBass", "Drive"},
		},
		{
			name: "the grammar of a chain",
			in:   sdk.Measured{Stats: s.stats()},
			want: []string{"bass", "drive", "100%"},
		},
		{
			name: "an instrument nobody measured",
			in:   sdk.Measured{Stats: s.stats(), Instrument: "guitar"},
			want: []string{"nothing measured"},
		},
		{
			name: "nowhere to write the grammar",
			in:   sdk.Measured{Stats: s.stats()},
			to:   &brokenWriter{},
			err:  true,
		},
		{
			name: "nowhere to write a model",
			in: sdk.Measured{
				Stats: s.stats(), Catalog: s.cat(), Model: "HD2_AmpTestBass",
			},
			to:  &brokenWriter{},
			err: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var buf bytes.Buffer

			to := tt.to
			if to == nil {
				to = &buf
			}

			err := cli.Measured(to, tt.in)

			if tt.err {
				s.Require().Error(err)

				return
			}

			s.Require().NoError(err)

			for _, want := range tt.want {
				s.Require().Contains(buf.String(), want)
			}
		})
	}
}

// TestMeasuredDoesNotShuffle covers two categories used equally often, which
// must not swap places between runs.
func (s *MeasuredPublicTestSuite) TestMeasuredDoesNotShuffle() {
	var first string

	for range 3 {
		var buf bytes.Buffer

		s.Require().NoError(cli.Measured(&buf, sdk.Measured{Stats: s.stats()}))

		if first == "" {
			first = buf.String()

			continue
		}

		s.Require().Equal(first, buf.String())
	}
}

func TestMeasuredPublicTestSuite(t *testing.T) {
	suite.Run(t, new(MeasuredPublicTestSuite))
}
