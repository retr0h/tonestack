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

	"github.com/retr0h/tonestack/pkg/cli"
	"github.com/retr0h/tonestack/pkg/sdk"
	"github.com/retr0h/tonestack/pkg/sdk/catalog"
)

type BlocksPublicTestSuite struct {
	suite.Suite
}

// amp is a block carrying everything a block can say about itself.
func amp() catalog.Block {
	return catalog.Block{
		ID: "HD2_AmpTestBass", Name: "Test Bass Amp",
		Category: catalog.CategoryAmp, Subcategory: "Bass",
		BasedOn: "Ampeg SVT (normal channel)",
		DSP:     catalog.DSPCost{Mono: 26.67, Stereo: 40.10, Prov: catalog.ProvOfficial},
		Prov:    catalog.ProvOfficial,
		Params: map[string]catalog.Param{
			"Drive":  {Key: "Drive", Type: catalog.ParamFloat, Min: 0, Max: 1},
			"Bright": {Key: "Bright", Type: catalog.ParamBool},
		},
	}
}

// TestBlocks covers listing what a device can do.
func (s *BlocksPublicTestSuite) TestBlocks() {
	tests := []struct {
		name string
		in   sdk.Blocks
		to   io.Writer
		want []string
		err  bool
	}{
		{
			name: "every block, with a count",
			in: sdk.Blocks{
				Device: "HX Stomp", Source: "HX Edit 3.82",
				Total: 2, Matched: []catalog.Block{amp()},
			},
			want: []string{
				"HD2_AmpTestBass", "Test Bass Amp", "Ampeg SVT",
				// The count says what the filter did.
				"1 of 2 blocks",
				// A catalog is only true of the release it came from.
				"HX Edit 3.82",
			},
		},
		{
			// Nothing recorded where it came from, which is different from a
			// source nobody recognises.
			name: "one that cannot name its source",
			in:   sdk.Blocks{Device: "HX Stomp", Total: 2, Matched: []catalog.Block{amp()}},
			want: []string{"source unknown"},
		},
		{
			name: "a filter nothing matches",
			in:   sdk.Blocks{Device: "HX Stomp", Source: "HX Edit", Total: 2},
			// A summary counting nothing is not drawn; the empty line is
			// the whole answer.
			want: []string{"no blocks match"},
		},
		{
			name: "nowhere to write it",
			in: sdk.Blocks{
				Device: "HX Stomp", Total: 2, Matched: []catalog.Block{amp()},
			},
			to:  &brokenWriter{},
			err: true,
		},
		{
			name: "nowhere to write the empty case either",
			in:   sdk.Blocks{Device: "HX Stomp", Total: 2},
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

			err := cli.Blocks(to, tt.in)

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

// TestBlock covers one block and everything it accepts.
func (s *BlocksPublicTestSuite) TestBlock() {
	tests := []struct {
		name   string
		in     catalog.Block
		to     io.Writer
		want   []string
		absent []string
		err    bool
	}{
		{
			name: "a block, its costs and its parameters",
			in:   amp(),
			want: []string{
				"Ampeg SVT (normal channel)", "amp (Bass)",
				"26.67 mono", "40.10 stereo",
				"Drive", "0..1",
				// A bool has no range.
				"Bright", "—",
			},
		},
		{
			name: "one with no stereo cost does not claim one",
			in: catalog.Block{
				ID: "HD2_DriveTest", Name: "Test Drive",
				Category: catalog.CategoryDrive,
				DSP:      catalog.DSPCost{Mono: 8.55, Prov: catalog.ProvOfficial},
				Prov:     catalog.ProvOfficial,
			},
			absent: []string{"stereo"},
			want:   []string{"no parameters"},
		},
		{
			// A DSP cost that was inferred must not read as Line 6's own
			// figure.
			name: "a figure nobody stated is marked",
			in: catalog.Block{
				ID: "HD2_Guessed", Name: "Guessed",
				Category: catalog.CategoryDrive,
				DSP:      catalog.DSPCost{Mono: 5, Prov: catalog.ProvAssumed},
				Prov:     catalog.ProvAssumed,
			},
			want: []string{"assumed"},
		},
		{
			name: "nowhere to write it",
			in:   amp(),
			to:   &brokenWriter{},
			err:  true,
		},
		{
			// The parameters are a second write, so a writer that survives
			// the first still has to be reported.
			name: "nowhere to write the parameters",
			in:   amp(),
			to:   &stops{ok: 1},
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

			err := cli.Block(to, tt.in)

			if tt.err {
				s.Require().Error(err)
				s.Require().Contains(err.Error(), "reporting")

				return
			}

			s.Require().NoError(err)

			for _, want := range tt.want {
				s.Require().Contains(buf.String(), want)
			}

			for _, gone := range tt.absent {
				s.Require().NotContains(buf.String(), gone)
			}
		})
	}
}

func TestBlocksPublicTestSuite(t *testing.T) {
	suite.Run(t, new(BlocksPublicTestSuite))
}
