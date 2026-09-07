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
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/internal/device"
	"github.com/retr0h/tonestack/pkg/sdk"
)

type ListPublicTestSuite struct {
	suite.Suite
}

// lister reports a fixed set of descriptors, or fails.
type lister struct {
	descs []sdk.Descriptor
	err   error
}

func (l *lister) List(context.Context) ([]sdk.Descriptor, error) { return l.descs, l.err }

// Close is what List releases when it found its own lister.
func (l *lister) Close() error { return nil }

func stomp() sdk.Descriptor {
	return sdk.Descriptor{Vendor: 0x0e41, Product: 0x4246, Bus: 2, Address: 1}
}

func (s *ListPublicTestSuite) TestListsARecognisedDevice() {
	var out bytes.Buffer

	err := device.ListWith(context.Background(), &out,
		&lister{descs: []sdk.Descriptor{stomp()}})

	s.Require().NoError(err)
	s.Require().Contains(out.String(), "HX Stomp")
	s.Require().Contains(out.String(), "0e41:4246")
	s.Require().Contains(out.String(), "2162694",
		"the preset device id is what a caller actually needs")
}

func (s *ListPublicTestSuite) TestSaysSoWhenNothingIsAttached() {
	var out bytes.Buffer

	err := device.ListWith(context.Background(), &out, &lister{})

	s.Require().NoError(err)
	s.Require().Contains(out.String(), "no Helix devices attached")
}

func (s *ListPublicTestSuite) TestIgnoresOtherVendors() {
	var out bytes.Buffer

	err := device.ListWith(context.Background(), &out, &lister{
		descs: []sdk.Descriptor{{Vendor: 0x05ac, Product: 0x1234}},
	})

	s.Require().NoError(err)
	s.Require().Contains(out.String(), "no Helix devices attached")
}

func (s *ListPublicTestSuite) TestReportsAFailingBus() {
	var out bytes.Buffer

	err := device.ListWith(context.Background(), &out,
		&lister{err: errors.New("bus unavailable")})

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "finding devices")
}

func (s *ListPublicTestSuite) TestReportsAFailingWriter() {
	// A tabwriter buffers, so the flush is where a failing writer surfaces.
	err := device.ListWith(context.Background(), &failingWriter{},
		&lister{descs: []sdk.Descriptor{stomp()}})
	s.Require().Error(err)

	err = device.ListWith(context.Background(), &failingWriter{}, &lister{})
	s.Require().Error(err, "the empty case reports too")
}

type failingWriter struct{}

func (*failingWriter) Write([]byte) (int, error) { return 0, errors.New("boom") }

func TestListPublicTestSuite(t *testing.T) {
	suite.Run(t, new(ListPublicTestSuite))
}

func (s *ListPublicTestSuite) TestListFindsItsOwnBus() {
	// One line — find a bus, hand it on, release it — and the only line in
	// this package that needs hardware.
	restore := *device.NewLister
	defer func() { *device.NewLister = restore }()

	*device.NewLister = func() device.Closer {
		return &lister{descs: []sdk.Descriptor{stomp()}}
	}

	var out bytes.Buffer
	s.Require().NoError(device.List(context.Background(), &out))
	s.Require().Contains(out.String(), "HX Stomp")
}
