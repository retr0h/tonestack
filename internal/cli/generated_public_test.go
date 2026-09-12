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

type GeneratedPublicTestSuite struct {
	suite.Suite
}

// TestCatalogued covers saying what a catalog generation run produced.
func (s *GeneratedPublicTestSuite) TestCatalogued() {
	built := sdk.Catalogued{
		Path:   "resources/schemas/hx-stomp.catalog.json",
		Device: "HX Stomp", Source: "HX Edit 3.82",
		Blocks: 681, Named: 547,
	}

	tests := []struct {
		name string
		in   sdk.Catalogued
		to   io.Writer
		want []string
		err  bool
	}{
		{
			name: "what went into it",
			in:   built,
			want: []string{
				"681 blocks for HX Stomp",
				// A catalog is only true of the release it came from.
				"HX Edit 3.82",
				// The gap between a model list and knowledge.
				"547 mapped to real gear",
			},
		},
		{name: "nowhere to say it", in: built, to: &brokenWriter{}, err: true},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var buf bytes.Buffer

			to := tt.to
			if to == nil {
				to = &buf
			}

			err := cli.Catalogued(to, tt.in)

			if tt.err {
				s.Require().Error(err)
				s.Require().Contains(err.Error(), "reporting")

				return
			}

			s.Require().NoError(err)

			for _, want := range tt.want {
				s.Require().Contains(buf.String(), want)
			}
		})
	}
}

// counted is a measuring run over a corpus holding one amp and one grammar.
func (s *GeneratedPublicTestSuite) counted() sdk.Counted {
	return sdk.Counted{
		Path: "resources/schemas/corpus.json.gz",
		Stats: &corpus.Stats{
			Device: "HX Stomp", Presets: 721,
			Models: map[catalog.ModelID]corpus.ModelStats{
				"HD2_AmpSVBeastNrm": {
					Uses: 40,
					Params: map[string]corpus.ParamStats{
						"Drive": {N: 40, Median: 0.44},
						"Bass":  {N: 40, Median: 0.5},
					},
				},
			},
			Grammar: map[string]corpus.Grammar{
				"bass": {
					Chains: 40,
					Categories: map[catalog.Category]corpus.CategoryStats{
						catalog.CategoryDrive: {Chains: 40, Before: 40},
						catalog.CategoryAmp:   {Chains: 40},
					},
				},
			},
		},
	}
}

// TestCounted covers saying what a corpus measuring run produced.
func (s *GeneratedPublicTestSuite) TestCounted() {
	tests := []struct {
		name string
		in   sdk.Counted
		to   io.Writer
		want []string
		err  bool
	}{
		{
			name: "what it measured",
			in:   s.counted(),
			want: []string{
				"721 presets measured",
				"bass",
				// Every drive in this corpus precedes the amp.
				"100%",
				// Two parameter distributions across one model.
				"1 models, 2 parameter distributions",
				"wrote resources/schemas/corpus.json.gz",
			},
		},
		{
			// Position is the whole point of the grammar, and a corpus with
			// no amp in it cannot say what comes before one.
			name: "a corpus that held no amp",
			in: sdk.Counted{
				Path:  "out.gz",
				Stats: &corpus.Stats{Device: "HX Stomp"},
			},
			want: []string{"no chains held an amp, so nothing could be ordered"},
		},
		{
			name: "nowhere to say it",
			in:   s.counted(),
			to:   &brokenWriter{},
			err:  true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var buf bytes.Buffer

			to := tt.to
			if to == nil {
				to = &buf
			}

			err := cli.Counted(to, tt.in)

			if tt.err {
				s.Require().Error(err)
				s.Require().Contains(err.Error(), "reporting")

				return
			}

			s.Require().NoError(err)

			for _, want := range tt.want {
				s.Require().Contains(buf.String(), want)
			}
		})
	}
}

func TestGeneratedPublicTestSuite(t *testing.T) {
	suite.Run(t, new(GeneratedPublicTestSuite))
}
