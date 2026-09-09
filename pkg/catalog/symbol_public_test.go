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

package catalog_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/catalog"
)

// SymbolPublicTestSuite covers the table a device names its models by.
type SymbolPublicTestSuite struct {
	suite.Suite
}

// TestSymbol names a model by where it sits in the device's own table.
func (s *SymbolPublicTestSuite) TestSymbol() {
	tests := []struct {
		name string
		// a table holding nothing, rather than this suite's own.
		bare bool
		at   int

		want   catalog.ModelID
		params []string
		ok     bool
	}{
		{
			name:   "a position the table reaches",
			at:     1,
			want:   "HD2_Second",
			params: []string{"Drive"},
			ok:     true,
		},
		// A device generated against a different release can name a model
		// this catalog has never heard of, and answering with the wrong one
		// would be worse than answering with nothing.
		{name: "a position below the start of it", at: -1},
		{name: "one past the end", at: 2},
		{name: "one far past it", at: 9999},
		{name: "a catalog with no table at all", bare: true},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			c := &catalog.Catalog{Symbols: []catalog.Symbol{
				{ID: "HD2_First"},
				{ID: "HD2_Second", Params: []string{"Drive"}},
			}}

			if tt.bare {
				c = &catalog.Catalog{}
			}

			got, ok := c.Symbol(tt.at)

			s.Require().Equal(tt.ok, ok)

			if !tt.ok {
				return
			}

			s.Require().Equal(tt.want, got.ID)
			s.Require().Equal(tt.params, got.Params)
		})
	}
}

func (s *SymbolPublicTestSuite) TestSymbolNumber() {
	c := &catalog.Catalog{Symbols: []catalog.Symbol{
		{ID: "HD2_First"},
		{ID: "HD2_SecondMono"},
		{ID: "HD2_SecondStereo"},
		{ID: "HD2_ThirdStereo"},
	}}

	tests := []struct {
		name  string
		id    catalog.ModelID
		want  int
		found bool
	}{
		{
			name:  "a name the table carries as it stands",
			id:    "HD2_First",
			want:  0,
			found: true,
		},
		{
			name:  "a name the table carries with a mono suffix",
			id:    "HD2_Second",
			want:  1,
			found: true,
		},
		{
			name:  "one carried only in stereo",
			id:    "HD2_Third",
			want:  3,
			found: true,
		},
		{
			name: "a name the table does not carry",
			id:   "HD2_Nowhere",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, ok := c.SymbolNumber(tt.id)

			s.Require().Equal(tt.found, ok)
			s.Require().Equal(tt.want, got)
		})
	}
}

func TestSymbolPublicTestSuite(t *testing.T) {
	suite.Run(t, new(SymbolPublicTestSuite))
}
