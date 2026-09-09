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

package sdk_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/retr0h/tonestack/pkg/sdk"
	"github.com/retr0h/tonestack/pkg/sdk/mocks"
)

type DiscoverPublicTestSuite struct {
	suite.Suite

	ctrl *gomock.Controller
}

func (s *DiscoverPublicTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
}

// lister returns a Lister reporting descs, or failing with err.
func (s *DiscoverPublicTestSuite) lister(
	descs []sdk.Descriptor,
	err error,
) *mocks.MockLister {
	m := mocks.NewMockLister(s.ctrl)
	m.EXPECT().List(gomock.Any()).Return(descs, err).AnyTimes()

	return m
}

// stomp returns the descriptor a real HX Stomp reports.
func stomp() sdk.Descriptor {
	return sdk.Descriptor{Vendor: 0x0e41, Product: 0x4246, Bus: 2, Address: 1}
}

// TestModelFor names a device by the product it reports on the bus.
func (s *DiscoverPublicTestSuite) TestModelFor() {
	tests := []struct {
		name     string
		product  uint16
		model    string
		deviceID int
	}{
		{
			name:     "a product the table carries",
			product:  0x4246,
			model:    "HX Stomp",
			deviceID: 2162694,
		},
		{
			// A Line 6 device this table has never seen is still on the bus,
			// it just cannot be named or written for.
			name:    "one it does not",
			product: 0xFFFF,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, ok := sdk.ModelFor(tt.product)

			if tt.model == "" {
				s.Require().False(ok)

				return
			}

			s.Require().True(ok)
			s.Require().Equal(tt.model, got.Name)
			s.Require().Equal(tt.deviceID, got.DeviceID)
		})
	}
}

// TestDevices names every Helix on the bus and ignores everything else.
func (s *DiscoverPublicTestSuite) TestDevices() {
	boom := errors.New("bus unavailable")

	tests := []struct {
		name   string
		descs  []sdk.Descriptor
		fails  error
		models []string
		// the detail beside the name, for a device the table carries.
		deviceID int
		bus      int
	}{
		{
			// Two identifier systems: a device answers to a USB product on
			// the bus and is named by a different number inside a preset.
			// Both are needed and neither derives from the other.
			name:     "one this table can name",
			descs:    []sdk.Descriptor{stomp()},
			models:   []string{"HX Stomp"},
			deviceID: 2162694,
			bus:      2,
		},
		{
			name: "another vendor's device alongside it",
			descs: []sdk.Descriptor{
				{Vendor: 0x05ac, Product: 0x1234},
				stomp(),
			},
			models: []string{"HX Stomp"},
		},
		{
			// Line 6 make more than this table knows about.
			name:   "a Line 6 product nothing recognises",
			descs:  []sdk.Descriptor{{Vendor: sdk.VendorID, Product: 0xBEEF}},
			models: []string{},
		},
		{
			name:   "an empty bus",
			models: []string{},
		},
		{
			name:  "a bus that cannot be looked at",
			fails: boom,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := sdk.Devices(context.Background(),
				s.lister(tt.descs, tt.fails))

			if tt.fails != nil {
				s.Require().ErrorIs(err, tt.fails)

				return
			}

			s.Require().NoError(err)
			s.Require().Equal(tt.models, names(got))

			if tt.deviceID == 0 {
				return
			}

			s.Require().Equal(tt.deviceID, got[0].DeviceID)
			s.Require().Equal(tt.bus, got[0].Descriptor.Bus)
		})
	}
}

// TestFirst picks one when a person did not say which.
func (s *DiscoverPublicTestSuite) TestFirst() {
	xl := sdk.Descriptor{Vendor: sdk.VendorID, Product: 0x4253, Bus: 1, Address: 4}

	tests := []struct {
		name  string
		descs []sdk.Descriptor
		fails error
		model string
		is    error
	}{
		{
			name:  "the only one attached",
			descs: []sdk.Descriptor{stomp()},
			model: "HX Stomp",
		},
		{
			// Bus order rather than the order the bus happened to answer in,
			// so two runs with the same hardware pick the same device.
			name:  "the earliest on the bus when several are",
			descs: []sdk.Descriptor{xl, stomp()},
			model: "HX Stomp XL",
		},
		{
			name: "an empty bus",
			is:   sdk.ErrNoDevice,
		},
		{
			name:  "a bus that cannot be looked at",
			fails: errors.New("bus unavailable"),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := sdk.First(context.Background(),
				s.lister(tt.descs, tt.fails))

			if tt.model == "" {
				s.Require().Error(err)

				if tt.is != nil {
					s.Require().ErrorIs(err, tt.is)
				}

				return
			}

			s.Require().NoError(err)
			s.Require().Equal(tt.model, got.Model)
		})
	}
}

// names is what a discovery answered with, in order.
func names(found []sdk.Device) []string {
	out := make([]string, 0, len(found))
	for _, d := range found {
		out = append(out, d.Model)
	}

	return out
}

func TestDiscoverPublicTestSuite(t *testing.T) {
	suite.Run(t, new(DiscoverPublicTestSuite))
}
