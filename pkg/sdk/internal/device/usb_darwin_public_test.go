//go:build darwin

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

package device_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/internal/device"
)

// USBDarwinPublicTestSuite runs the macOS backend against the real registry.
//
// Listing and finding read the IOKit registry and open nothing, so they run
// on any Mac with or without a device attached, and cannot disturb one that
// is. Claiming an interface needs hardware, and lives in the device round
// trip instead.
type USBDarwinPublicTestSuite struct {
	suite.Suite
}

// TestList covers listing what is on the bus.
func (s *USBDarwinPublicTestSuite) TestList() {
	l := device.NewUSBLister()

	got, err := l.List(context.Background())

	s.Require().NoError(err)
	s.Require().NotNil(got)
	s.Require().NoError(l.Close())

	got, err = device.NewUSB(nil).List(context.Background())

	s.Require().NoError(err)
	s.Require().NotNil(got)
}

// TestSearch covers looking through the registry for devices.
//
// The matcher accepts nothing, so nothing is kept and nothing is opened,
// whatever is plugged in.
func (s *USBDarwinPublicTestSuite) TestSearch() {
	n, err := device.Search()

	s.Require().NoError(err)
	s.Require().Zero(n)
}

func TestUSBDarwinPublicTestSuite(t *testing.T) {
	suite.Run(t, new(USBDarwinPublicTestSuite))
}
