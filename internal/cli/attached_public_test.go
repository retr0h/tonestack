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

package cli_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/internal/cli"
	"github.com/retr0h/tonestack/pkg/sdk"
)

type AttachedPublicTestSuite struct {
	suite.Suite
}

// TestAttached covers saying what is on the bus.
func (s *AttachedPublicTestSuite) TestAttached() {
	stomp := sdk.Attachment{
		Model: "HX Stomp", DeviceID: 2162694,
		Vendor: 0x0e41, Product: 0x4246,
		Bus: 20, Address: 3,
	}

	tests := []struct {
		name string
		in   sdk.Attached
		to   io.Writer
		want []string
		err  bool
	}{
		{
			name: "a device this project knows",
			in:   sdk.Attached{Devices: []sdk.Attachment{stomp}},
			want: []string{
				"HX Stomp",
				"0e41:4246",
				"20.3",
				// The preset device id is what a caller actually needs.
				"2162694",
			},
		},
		{
			name: "nothing attached",
			in:   sdk.Attached{},
			want: []string{"no Helix devices attached"},
		},
		{
			// A tabwriter buffers, so the flush is where a failing writer
			// surfaces.
			name: "nowhere to write it, with a device to print",
			in:   sdk.Attached{Devices: []sdk.Attachment{stomp}},
			to:   &brokenWriter{},
			err:  true,
		},
		{
			name: "nowhere to write it, with nothing to print",
			in:   sdk.Attached{},
			to:   &brokenWriter{},
			err:  true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var buf bytes.Buffer

			to := tt.to
			if to == nil {
				to = &buf
			}

			err := cli.Attached(to, tt.in)

			if tt.err {
				s.Require().Error(err)
				s.Require().Contains(err.Error(), "reporting")

				return
			}

			s.Require().NoError(err)

			for _, want := range tt.want {
				s.Require().Contains(buf.String(), want)
			}
		})
	}
}

func TestAttachedPublicTestSuite(t *testing.T) {
	suite.Run(t, new(AttachedPublicTestSuite))
}
