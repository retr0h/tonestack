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
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/cli"
	"github.com/retr0h/tonestack/pkg/sdk"
	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/chain"
)

// ListingPublicTestSuite covers drawing what a setlist holds.
type ListingPublicTestSuite struct {
	suite.Suite

	cat *catalog.Catalog
}

func (s *ListingPublicTestSuite) SetupSuite() {
	var err error

	s.cat, err = catalog.BuiltIn()
	s.Require().NoError(err)
}

// listing returns one slot in use and one holding nothing.
func (s *ListingPublicTestSuite) listing() sdk.Listing {
	return sdk.Listing{
		Name: "HX Stomp",
		Slots: []sdk.Held{
			{Slot: 0, Name: "Chunky Monkey", Blocks: []chain.Block{
				{Model: "HD2_AmpSVBeastNrm"},
				{Model: "HD2_Cab8x10SVBeast"},
			}},
			{Slot: 1, Name: "New Preset"},
		},
	}
}

// TestListing covers what a reader sees, and what they are spared.
func (s *ListingPublicTestSuite) TestListing() {
	tests := []struct {
		name     string
		in       sdk.Listing
		all      bool
		contains []string
		absent   []string
	}{
		{
			// The empty slot is left out, because a listing is for finding
			// the preset you meant.
			name:     "the slots in use",
			in:       s.listing(),
			contains: []string{"HX Stomp", "01A", "Chunky Monkey", "2 slots", "1 in use"},
			absent:   []string{"New Preset", "empty"},
		},
		{
			// And shown when asked, because that is how somebody finds
			// somewhere to put one.
			name:     "every slot, including the gaps",
			in:       s.listing(),
			all:      true,
			contains: []string{"01B", "New Preset", "empty"},
		},
		{
			name:     "a setlist with nothing in it",
			in:       sdk.Listing{Name: "HX Stomp"},
			contains: []string{"no presets", "0 slots", "0 in use"},
		},
		{
			// One slot rather than "1 slots".
			name: "a setlist of one",
			in: sdk.Listing{Name: "HX Stomp", Slots: []sdk.Held{
				{Slot: 0, Name: "Only", Blocks: []chain.Block{{Model: "HD2_AmpSVBeastNrm"}}},
			}},
			contains: []string{"1 slot ", "1 in use"},
		},
		{
			// A model this catalog cannot name is shown as a gap in the
			// chain rather than left out of it.
			name: "a block nothing names",
			in: sdk.Listing{Name: "HX Stomp", Slots: []sdk.Held{
				{Slot: 0, Name: "Odd", Blocks: []chain.Block{{Model: "HD2_FromNewerFirmware"}}},
			}},
			contains: []string{"?"},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var out bytes.Buffer

			s.Require().NoError(cli.Listing(&out, tt.in, s.cat, tt.all))

			for _, want := range tt.contains {
				s.Require().Contains(out.String(), want)
			}

			for _, gone := range tt.absent {
				s.Require().NotContains(out.String(), gone)
			}
		})
	}
}

// TestPlural covers a count reading as English.
func (s *ListingPublicTestSuite) TestPlural() {
	tests := []struct {
		name string
		n    int
		want string
	}{
		{name: "none", n: 0, want: "0 slots"},
		{name: "one", n: 1, want: "1 slot"},
		{name: "several", n: 4, want: "4 slots"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.want, cli.Plural(tt.n, "slot"))
		})
	}
}

// TestFlow covers reading a chain as one line.
func (s *ListingPublicTestSuite) TestFlow() {
	tests := []struct {
		name   string
		blocks []chain.Block
		want   string
	}{
		{name: "nothing in the chain", want: ""},
		{
			name:   "a model this catalog does not carry",
			blocks: []chain.Block{{Model: "HD2_FromNewerFirmware"}},
			want:   "?",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.want, cli.Flow(&bytes.Buffer{}, tt.blocks, s.cat))
		})
	}
}

func TestListingPublicTestSuite(t *testing.T) {
	suite.Run(t, new(ListingPublicTestSuite))
}
