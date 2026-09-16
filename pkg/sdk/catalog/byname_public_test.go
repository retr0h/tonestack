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

// ByNamePublicTestSuite covers reaching a catalog by the name of a device.
//
// Somebody naming their own pedal types what is printed on it, not the number
// a preset carries, so what matters is that the names people would write
// actually land.
type ByNamePublicTestSuite struct {
	suite.Suite
}

// TestDevices names every catalog that ships.
func (s *ByNamePublicTestSuite) TestDevices() {
	s.Require().Equal(
		[]string{"HX Stomp", "HX Stomp XL", "Helix Floor", "Helix LT"},
		catalog.Devices())
}

// TestForName covers the spellings somebody might reasonably type.
func (s *ByNamePublicTestSuite) TestForName() {
	tests := []struct {
		name   string
		asked  string
		device int
	}{
		{name: "as it is printed", asked: "HX Stomp", device: catalog.HXStomp},
		{name: "all lower", asked: "hx stomp", device: catalog.HXStomp},
		{name: "hyphenated", asked: "helix-lt", device: catalog.HelixLT},
		{name: "run together", asked: "helixfloor", device: catalog.HelixFloor},
		{name: "shouted", asked: "HX STOMP XL", device: catalog.HXStompXL},
		{name: "padded", asked: "  Helix Floor  ", device: catalog.HelixFloor},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := catalog.ForName(tt.asked)
			s.Require().NoError(err)
			s.Require().Equal(tt.device, got.DeviceID)
		})
	}
}

// TestForNameSomethingNothingShipsFor answers with what it does ship for.
//
// The useful half of the message: "that is not one" helps nobody without the
// list of ones that are.
func (s *ByNamePublicTestSuite) TestForNameSomethingNothingShipsFor() {
	got, err := catalog.ForName("Kemper")

	s.Require().Nil(got)
	s.Require().ErrorIs(err, catalog.ErrNoDevice)
	s.Require().Contains(err.Error(), `"Kemper"`)

	for _, name := range catalog.Devices() {
		s.Require().Contains(err.Error(), name)
	}
}

// TestForNameNothingAtAll is refused rather than taken as the default.
//
// An empty name reaching here is a caller that meant to pass one, and quietly
// answering with the HX Stomp's would hide the mistake.
func (s *ByNamePublicTestSuite) TestForNameNothingAtAll() {
	got, err := catalog.ForName("")

	s.Require().Nil(got)
	s.Require().ErrorIs(err, catalog.ErrNoDevice)
}

// TestEachNameReachesItsOwnCatalog is what makes the names worth having.
//
// One catalog answering to four names would pass every test above.
func (s *ByNamePublicTestSuite) TestEachNameReachesItsOwnCatalog() {
	stomp, err := catalog.ForName("HX Stomp")
	s.Require().NoError(err)

	floor, err := catalog.ForName("Helix Floor")
	s.Require().NoError(err)

	s.Require().Greater(len(floor.Blocks), len(stomp.Blocks),
		"the floor unit carries blocks the Stomp has not")
}

func TestByNamePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ByNamePublicTestSuite))
}
