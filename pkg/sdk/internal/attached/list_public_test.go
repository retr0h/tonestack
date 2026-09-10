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

package attached_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/internal/attached"
	"github.com/retr0h/tonestack/pkg/sdk/internal/device"
)

type ListPublicTestSuite struct {
	suite.Suite
}

// lister reports a fixed set of descriptors, or fails.
type lister struct {
	descs []device.Descriptor
	err   error
}

func (l *lister) List(context.Context) ([]device.Descriptor, error) { return l.descs, l.err }

// Close is what List releases when it found its own lister.
func (l *lister) Close() error { return nil }

func stomp() device.Descriptor {
	return device.Descriptor{Vendor: 0x0e41, Product: 0x4246, Bus: 2, Address: 1}
}

// TestListWith prints what is attached.
func (s *ListPublicTestSuite) TestListWith() {
	tests := []struct {
		name    string
		descs   []device.Descriptor
		listErr error
		// nothing on the bus this project recognises.
		empty    bool
		contains []string
		errText  string
	}{
		{
			name:  "a device this project knows",
			descs: []device.Descriptor{stomp()},
			contains: []string{
				"HX Stomp",
				"0e41:4246",
				// The preset device id is what a caller actually needs.
				"2162694",
			},
		},
		{
			name:  "nothing attached",
			empty: true,
		},
		{
			// A bus holds keyboards and webcams. A list of those is not an
			// answer to what a preset can be written to.
			name:  "somebody else's hardware",
			descs: []device.Descriptor{{Vendor: 0x05ac, Product: 0x1234}},
			empty: true,
		},
		{
			name:    "a bus that will not answer",
			listErr: errors.New("bus unavailable"),
			errText: "finding devices",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			found, err := attached.ListWith(context.Background(),
				&lister{descs: tt.descs, err: tt.listErr})

			if tt.errText != "" {
				s.Require().Error(err)
				s.Require().Contains(err.Error(), tt.errText)

				return
			}

			s.Require().NoError(err)

			if tt.empty {
				s.Require().Empty(found.Devices)

				return
			}

			got := ""
			for _, d := range found.Devices {
				got += fmt.Sprintf("%s %04x:%04x %d.%d %d ",
					d.Model, d.Vendor, d.Product, d.Bus, d.Address, d.DeviceID)
			}

			for _, want := range tt.contains {
				s.Require().Contains(got, want)
			}
		})
	}
}

// TestList finds its own bus. One line — find a bus, hand it on, release it —
// and the only line in this package that needs hardware.
func (s *ListPublicTestSuite) TestList() {
	restore := attached.NewLister
	defer func() { attached.NewLister = restore }()

	attached.NewLister = func() attached.Closer {
		return &lister{descs: []device.Descriptor{stomp()}}
	}

	found, err := attached.List(context.Background())
	s.Require().NoError(err)
	s.Require().Len(found.Devices, 1)
	s.Require().Equal("HX Stomp", found.Devices[0].Model)
}

func TestListPublicTestSuite(t *testing.T) {
	suite.Run(t, new(ListPublicTestSuite))
}
