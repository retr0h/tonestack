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

package result_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/chain"
	"github.com/retr0h/tonestack/pkg/sdk/result"
)

// ListingPublicTestSuite covers what a caller is handed for a setlist.
type ListingPublicTestSuite struct {
	suite.Suite
}

// TestUsed covers counting the slots that hold something.
func (s *ListingPublicTestSuite) TestUsed() {
	tests := []struct {
		name string
		in   result.Listing
		want int
	}{
		{name: "a setlist nobody has touched"},
		{
			name: "one preset in a hundred and twenty-eight slots",
			in: result.Listing{Slots: []result.Held{
				{Slot: 0, Name: "New Preset"},
				{Slot: 1, Name: "Mike Dirnt", Blocks: []chain.Block{{Model: "x"}}},
				{Slot: 2, Name: "New Preset"},
			}},
			want: 1,
		},
		{
			name: "every slot in use",
			in: result.Listing{Slots: []result.Held{
				{Blocks: []chain.Block{{Model: "x"}}},
				{Blocks: []chain.Block{{Model: "y"}}},
			}},
			want: 2,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.want, tt.in.Used())
		})
	}
}

// TestEmpty covers what says a slot holds nothing.
func (s *ListingPublicTestSuite) TestEmpty() {
	tests := []struct {
		name string
		in   result.Held
		want bool
	}{
		{
			// The name a slot shipped with, and nothing in it.
			name: "untouched",
			in:   result.Held{Name: "New Preset"},
			want: true,
		},
		{
			// Somebody named it and then emptied it. The name is no guide.
			name: "named, and holding nothing",
			in:   result.Held{Name: "Lead"},
			want: true,
		},
		{
			name: "holding a chain",
			in:   result.Held{Name: "Lead", Blocks: []chain.Block{{Model: "x"}}},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.want, tt.in.Empty())
		})
	}
}

func TestListingPublicTestSuite(t *testing.T) {
	suite.Run(t, new(ListingPublicTestSuite))
}
