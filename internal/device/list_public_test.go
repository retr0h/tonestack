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
	"io"
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

// TestListWith prints what is attached.
func (s *ListPublicTestSuite) TestListWith() {
	tests := []struct {
		name     string
		descs    []sdk.Descriptor
		listErr  error
		deaf     bool
		contains []string
		errText  string
	}{
		{
			name:  "a device this project knows",
			descs: []sdk.Descriptor{stomp()},
			contains: []string{
				"HX Stomp",
				"0e41:4246",
				// The preset device id is what a caller actually needs.
				"2162694",
			},
		},
		{
			name:     "nothing attached",
			contains: []string{"no Helix devices attached"},
		},
		{
			name:     "somebody else's hardware",
			descs:    []sdk.Descriptor{{Vendor: 0x05ac, Product: 0x1234}},
			contains: []string{"no Helix devices attached"},
		},
		{
			name:    "a bus that will not answer",
			listErr: errors.New("bus unavailable"),
			errText: "finding devices",
		},
		{
			// A tabwriter buffers, so the flush is where a failing writer
			// surfaces.
			name:    "a writer that fails, with a device to print",
			descs:   []sdk.Descriptor{stomp()},
			deaf:    true,
			errText: "boom",
		},
		{
			name:    "a writer that fails, with nothing to print",
			deaf:    true,
			errText: "boom",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var buf bytes.Buffer

			out := io.Writer(&buf)
			if tt.deaf {
				out = &failingWriter{}
			}

			err := device.ListWith(context.Background(), out,
				&lister{descs: tt.descs, err: tt.listErr})

			if tt.errText != "" {
				s.Require().Error(err)
				s.Require().Contains(err.Error(), tt.errText)

				return
			}

			s.Require().NoError(err)

			for _, want := range tt.contains {
				s.Require().Contains(buf.String(), want)
			}
		})
	}
}

// TestList finds its own bus. One line — find a bus, hand it on, release it —
// and the only line in this package that needs hardware.
func (s *ListPublicTestSuite) TestList() {
	restore := *device.NewLister
	defer func() { *device.NewLister = restore }()

	*device.NewLister = func() device.Closer {
		return &lister{descs: []sdk.Descriptor{stomp()}}
	}

	var out bytes.Buffer
	s.Require().NoError(device.List(context.Background(), &out))
	s.Require().Contains(out.String(), "HX Stomp")
}

type failingWriter struct{}

func (*failingWriter) Write([]byte) (int, error) { return 0, errors.New("boom") }

func TestListPublicTestSuite(t *testing.T) {
	suite.Run(t, new(ListPublicTestSuite))
}
