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

func (s *DiscoverPublicTestSuite) TestModelForFindsAKnownProduct() {
	m, ok := sdk.ModelFor(0x4246)

	s.Require().True(ok)
	s.Require().Equal("HX Stomp", m.Name)
	s.Require().Equal(2162694, m.DeviceID)
}

func (s *DiscoverPublicTestSuite) TestModelForReportsFalseForUnknown() {
	_, ok := sdk.ModelFor(0xFFFF)

	s.Require().False(ok)
}

func (s *DiscoverPublicTestSuite) TestDevicesNamesARecognisedDevice() {
	got, err := sdk.Devices(context.Background(),
		s.lister([]sdk.Descriptor{stomp()}, nil))

	s.Require().NoError(err)
	s.Require().Len(got, 1)
	s.Require().Equal("HX Stomp", got[0].Model)
	s.Require().Equal(2162694, got[0].DeviceID)
	s.Require().Equal(2, got[0].Descriptor.Bus)
}

func (s *DiscoverPublicTestSuite) TestDevicesIgnoresOtherVendors() {
	got, err := sdk.Devices(context.Background(), s.lister([]sdk.Descriptor{
		{Vendor: 0x05ac, Product: 0x1234},
		stomp(),
	}, nil))

	s.Require().NoError(err)
	s.Require().Len(got, 1)
	s.Require().Equal("HX Stomp", got[0].Model)
}

func (s *DiscoverPublicTestSuite) TestDevicesIgnoresUnrecognisedLine6Products() {
	got, err := sdk.Devices(context.Background(), s.lister([]sdk.Descriptor{
		{Vendor: sdk.VendorID, Product: 0xBEEF},
	}, nil))

	s.Require().NoError(err)
	s.Require().Empty(got)
}

func (s *DiscoverPublicTestSuite) TestDevicesReturnsEmptyForAnEmptyBus() {
	got, err := sdk.Devices(context.Background(), s.lister(nil, nil))

	s.Require().NoError(err)
	s.Require().Empty(got)
}

func (s *DiscoverPublicTestSuite) TestDevicesPropagatesListerFailure() {
	boom := errors.New("bus unavailable")

	_, err := sdk.Devices(context.Background(), s.lister(nil, boom))

	s.Require().ErrorIs(err, boom)
}

func (s *DiscoverPublicTestSuite) TestFirstReturnsTheOnlyDevice() {
	got, err := sdk.First(context.Background(),
		s.lister([]sdk.Descriptor{stomp()}, nil))

	s.Require().NoError(err)
	s.Require().Equal("HX Stomp", got.Model)
}

func (s *DiscoverPublicTestSuite) TestFirstPrefersBusOrderWhenSeveralAttached() {
	xl := sdk.Descriptor{Vendor: sdk.VendorID, Product: 0x4253, Bus: 1, Address: 4}

	got, err := sdk.First(context.Background(),
		s.lister([]sdk.Descriptor{xl, stomp()}, nil))

	s.Require().NoError(err)
	s.Require().Equal("HX Stomp XL", got.Model)
}

func (s *DiscoverPublicTestSuite) TestFirstReportsNoDeviceOnAnEmptyBus() {
	_, err := sdk.First(context.Background(), s.lister(nil, nil))

	s.Require().ErrorIs(err, sdk.ErrNoDevice)
}

func (s *DiscoverPublicTestSuite) TestFirstPropagatesListerFailure() {
	_, err := sdk.First(context.Background(),
		s.lister(nil, errors.New("bus unavailable")))

	s.Require().Error(err)
}

func TestDiscoverPublicTestSuite(t *testing.T) {
	suite.Run(t, new(DiscoverPublicTestSuite))
}
