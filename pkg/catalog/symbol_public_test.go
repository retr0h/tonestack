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

func (s *SymbolPublicTestSuite) TestNamesAModelByItsPosition() {
	c := &catalog.Catalog{Symbols: []catalog.Symbol{
		{ID: "HD2_First"},
		{ID: "HD2_Second", Params: []string{"Drive"}},
	}}

	got, ok := c.Symbol(1)

	s.Require().True(ok)
	s.Require().Equal(catalog.ModelID("HD2_Second"), got.ID)
	s.Require().Equal([]string{"Drive"}, got.Params)
}

func (s *SymbolPublicTestSuite) TestRefusesAPositionTheTableDoesNotReach() {
	// A device generated against a different release can name a model this
	// catalog has never heard of, and answering with the wrong one would be
	// worse than answering with nothing.
	c := &catalog.Catalog{Symbols: []catalog.Symbol{{ID: "HD2_Only"}}}

	for _, n := range []int{-1, 1, 9999} {
		_, ok := c.Symbol(n)

		s.Require().False(ok, "%d", n)
	}
}

func (s *SymbolPublicTestSuite) TestACatalogWithNoTable() {
	_, ok := (&catalog.Catalog{}).Symbol(0)

	s.Require().False(ok)
}

func TestSymbolPublicTestSuite(t *testing.T) {
	suite.Run(t, new(SymbolPublicTestSuite))
}
