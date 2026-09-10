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

	"github.com/retr0h/tonestack/pkg/sdk"
	"github.com/retr0h/tonestack/pkg/sdk/device"
)

// bus stands in for the one thing this library needs hardware for.
type bus struct {
	descs []device.Descriptor
	err   error
}

func (b *bus) List(context.Context) ([]device.Descriptor, error) {
	return b.descs, b.err
}

func (*bus) Close() error { return nil }

type ClientPublicTestSuite struct {
	suite.Suite
}

// stand puts a bus in front of the Client and gives back what undoes it.
func (s *ClientPublicTestSuite) stand(b *bus) func() {
	restore := *sdk.NewLister
	*sdk.NewLister = func() sdk.Closer { return b }

	return func() { *sdk.NewLister = restore }
}

// TestNew covers building a Client.
func (s *ClientPublicTestSuite) TestNew() {
	tests := []struct {
		name string
		opts []sdk.Option
	}{
		{
			// The common case is somebody who wants the built-in catalog,
			// the built-in statistics and whatever is plugged in.
			name: "with nothing said about it",
		},
		{
			name: "with an option that says nothing yet",
			opts: []sdk.Option{func(*sdk.Options) {}},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().NotNil(sdk.New(tt.opts...))
		})
	}
}

// TestDevices covers reporting what is attached.
func (s *ClientPublicTestSuite) TestDevices() {
	stomp := device.Descriptor{Vendor: 0x0e41, Product: 0x4246, Bus: 20, Address: 3}

	tests := []struct {
		name  string
		bus   *bus
		want  int
		first string
		err   bool
	}{
		{
			name:  "a device this project knows",
			bus:   &bus{descs: []device.Descriptor{stomp}},
			want:  1,
			first: "HX Stomp",
		},
		{
			// A bus holds keyboards and webcams. Those are not an answer to
			// what a preset can be written to.
			name: "somebody else's hardware",
			bus:  &bus{descs: []device.Descriptor{{Vendor: 0x05ac, Product: 0x1234}}},
		},
		{name: "nothing attached", bus: &bus{}},
		{
			name: "a bus that will not answer",
			bus:  &bus{err: errors.New("bus unavailable")},
			err:  true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			defer s.stand(tt.bus)()

			found, err := sdk.New().Devices(context.Background())

			if tt.err {
				s.Require().Error(err)

				return
			}

			s.Require().NoError(err)
			s.Require().Len(found.Devices, tt.want)

			if tt.first != "" {
				s.Require().Equal(tt.first, found.Devices[0].Model)
			}
		})
	}
}

func TestClientPublicTestSuite(t *testing.T) {
	suite.Run(t, new(ClientPublicTestSuite))
}
