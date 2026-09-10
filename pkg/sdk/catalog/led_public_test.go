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

	"github.com/retr0h/tonestack/pkg/sdk/catalog"
)

// LEDPublicTestSuite covers naming the colour a device reports by number.
type LEDPublicTestSuite struct {
	suite.Suite
}

func (s *LEDPublicTestSuite) TestNamesAColourByItsPosition() {
	// The order is the contract: a device sends the position, not the name.
	// Two of these are confirmed against hardware — a switch set to Green in
	// HX Edit reports 6, and one set to Violet reports 9.
	c, err := catalog.BuiltIn()
	s.Require().NoError(err)

	for n, want := range map[int]string{0: "auto color", 6: "green", 9: "violet"} {
		got, ok := c.LEDColour(n)

		s.Require().True(ok, "%d", n)
		s.Require().Equal(want, got, "colour %d", n)
	}
}

func (s *LEDPublicTestSuite) TestAColourThisCatalogDoesNotKnow() {
	// A device on newer firmware may know colours the release this was
	// generated from did not.
	c := &catalog.Catalog{LEDColours: []string{"auto color"}}

	for _, n := range []int{-1, 1, 999} {
		_, ok := c.LEDColour(n)

		s.Require().False(ok, "%d", n)
	}
}

func TestLEDPublicTestSuite(t *testing.T) {
	suite.Run(t, new(LEDPublicTestSuite))
}
