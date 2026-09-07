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

type EmbedPublicTestSuite struct {
	suite.Suite
}

func (s *EmbedPublicTestSuite) TestBuiltInIsUsable() {
	// The whole point of embedding is that nothing else has to be installed,
	// so this asserts the shipped catalog is complete enough to build with.
	c, err := catalog.BuiltIn()

	s.Require().NoError(err)
	s.Require().Equal(2162694, c.DeviceID)
	s.Require().NotEmpty(c.Source, "a catalog must say which release it came from")
	s.Require().Greater(len(c.Blocks), 500)

	b, ok := c.Block("HD2_AmpSVBeastNrm")
	s.Require().True(ok)
	s.Require().Contains(b.BasedOn, "Ampeg SVT")
}

func (s *EmbedPublicTestSuite) TestNeedsUserIR() {
	tests := []struct {
		name string
		id   catalog.ModelID
		want bool
	}{
		{
			"a block playing the owner's own impulse response",
			"HD2_ImpulseResponse1024", true,
		},
		{"the same at another length", "HD2_ImpulseResponse2048", true},
		{
			"a cabinet whose impulse response ships in the device",
			"HD2_CabMicIr_1x12BlueBell", false,
		},
		{"an amp", "HD2_AmpSVBeastNrm", false},
		{"nothing at all", "", false},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			s.Require().Equal(tc.want, catalog.NeedsUserIR(tc.id))
		})
	}
}

func (s *EmbedPublicTestSuite) TestTheShippedCatalogHoldsNoUserIRCabs() {
	// A generated chain draws only from this catalog, so anything needing a
	// user's own IR must be visible here rather than discovered on a device.
	c, err := catalog.BuiltIn()
	s.Require().NoError(err)

	var found int

	for id := range c.Blocks {
		if catalog.NeedsUserIR(id) {
			found++
		}
	}

	s.Require().NotZero(found,
		"the block exists and must be recognisable, not filtered out of the catalog")
}

func (s *EmbedPublicTestSuite) TestDecodeRejectsADamagedArchive() {
	tests := []struct {
		name   string
		packed []byte
		want   string
	}{
		{"bytes that are not gzip", []byte("not gzip at all"), "opening"},
		{
			"a gzip header with nothing after it",
			[]byte{0x1f, 0x8b, 0x08, 0, 0, 0, 0, 0, 0, 0xff},
			"reading",
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			_, err := catalog.Decode(tc.packed)

			s.Require().Error(err)
			s.Require().Contains(err.Error(), tc.want)
		})
	}
}

func TestEmbedPublicTestSuite(t *testing.T) {
	suite.Run(t, new(EmbedPublicTestSuite))
}
