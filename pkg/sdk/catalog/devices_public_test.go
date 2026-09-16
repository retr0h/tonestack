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

// DevicesPublicTestSuite covers the catalog each device gets.
//
// One model table filtered four ways, so what matters is that each device
// answers with its own name and id rather than the HX Stomp's, and that a
// device nothing was generated for says so.
type DevicesPublicTestSuite struct {
	suite.Suite
}

// TestFor covers every device a catalog ships for.
func (s *DevicesPublicTestSuite) TestFor() {
	tests := []struct {
		name   string
		device int
		want   string
	}{
		{
			name:   "the device everything was written against",
			device: catalog.HXStomp, want: "HX Stomp",
		},
		{name: "the bigger Stomp", device: catalog.HXStompXL, want: "HX Stomp XL"},
		{name: "the floor unit", device: catalog.HelixFloor, want: "Helix Floor"},
		{name: "the LT", device: catalog.HelixLT, want: "Helix LT"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := catalog.For(tt.device)
			s.Require().NoError(err)

			s.Require().Equal(tt.want, got.Device)
			s.Require().Equal(tt.device, got.DeviceID)
			s.Require().NotEmpty(got.Blocks)
			s.Require().NotEmpty(got.Symbols)
		})
	}
}

// TestForADeviceNothingShipsFor names what was asked for.
func (s *DevicesPublicTestSuite) TestForADeviceNothingShipsFor() {
	got, err := catalog.For(1)

	s.Require().Nil(got)
	s.Require().ErrorIs(err, catalog.ErrNoDevice)
	s.Require().Contains(err.Error(), "device 1")
}

// TestTheDevicesDiffer checks the filtering actually filtered.
//
// A Helix Floor carries blocks an HX Stomp does not. Four copies of one
// catalog under four names would pass every other test here.
func (s *DevicesPublicTestSuite) TestTheDevicesDiffer() {
	stomp, err := catalog.For(catalog.HXStomp)
	s.Require().NoError(err)

	floor, err := catalog.For(catalog.HelixFloor)
	s.Require().NoError(err)

	s.Require().Greater(len(floor.Blocks), len(stomp.Blocks),
		"the floor unit carries blocks the Stomp has not")
}

// TestBuiltInIsTheStomp holds the promise every existing caller relies on.
func (s *DevicesPublicTestSuite) TestBuiltInIsTheStomp() {
	built, err := catalog.BuiltIn()
	s.Require().NoError(err)

	stomp, err := catalog.For(catalog.HXStomp)
	s.Require().NoError(err)

	s.Require().Equal(stomp.DeviceID, built.DeviceID)
	s.Require().Equal(stomp.Device, built.Device)
}

func TestDevicesPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(DevicesPublicTestSuite))
}
