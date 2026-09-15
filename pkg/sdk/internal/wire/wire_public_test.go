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

package wire_test

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/internal/wire"
)

type WirePublicTestSuite struct {
	suite.Suite
}

// TestEncodeEnvelope lays out the header a device reads.
func (s *WirePublicTestSuite) TestEncodeEnvelope() {
	tests := []struct {
		name string
		env  wire.Envelope
		want string
	}{
		{
			// Originator 1, service 5, length 2, then the body — all little
			// endian.
			name: "a frame from the host",
			env: wire.Envelope{
				Originator: wire.FromHost,
				Service:    5,
				Body:       []byte{0xde, 0xad},
			},
			want: "0100050002000000dead",
		},
		{
			// Host frames always carry 1 and device frames always 0, which is
			// the cheapest check that a stream is still aligned.
			name: "one from the device, carrying nothing",
			env:  wire.Envelope{Originator: wire.FromDevice},
			want: "00000000000000 00",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().Equal(
				strip(tt.want), hex.EncodeToString(wire.EncodeEnvelope(tt.env)))
		})
	}
}

// strip removes the spaces used to group a hex fixture.
func strip(
	in string,
) string {
	out := make([]byte, 0, len(in))

	for i := range len(in) {
		if in[i] != ' ' {
			out = append(out, in[i])
		}
	}

	return string(out)
}

// TestDecodeEnvelope reads a frame back, and says what follows it.
func (s *WirePublicTestSuite) TestDecodeEnvelope() {
	tests := []struct {
		name string
		// frames to encode and read back, or bytes written by hand.
		frames []wire.Envelope
		raw    []byte
		err    error
		// errText is a part of the message a person reads.
		errText string
	}{
		{
			name: "a frame from the host",
			frames: []wire.Envelope{
				{Originator: wire.FromHost, Service: 2, Body: []byte("hello")},
			},
		},
		{
			// A bulk read can carry more than one frame, so the remainder has
			// to come back rather than be dropped.
			name: "two frames in one transfer",
			frames: []wire.Envelope{
				{Originator: wire.FromDevice, Body: []byte("one")},
				{Originator: wire.FromDevice, Body: []byte("two")},
			},
		},
		{name: "nothing at all", raw: []byte{}, err: wire.ErrShortFrame},
		{
			name: "half a header",
			raw:  []byte{1, 0, 5, 0},
			err:  wire.ErrShortFrame,
		},
		{
			name: "a header promising more body than arrived",
			raw:  []byte{1, 0, 5, 0, 0x10, 0, 0, 0, 0xde},
			err:  wire.ErrShortFrame,
		},
		{
			name:    "a length no real frame carries",
			raw:     []byte{1, 0, 5, 0, 0xff, 0xff, 0xff, 0xff},
			err:     wire.ErrBodyTooLarge,
			errText: "too large",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			if tt.err != nil {
				_, _, err := wire.DecodeEnvelope(tt.raw)

				s.Require().ErrorIs(err, tt.err)
				s.Require().Contains(err.Error(), tt.errText)

				return
			}

			var raw []byte
			for _, env := range tt.frames {
				raw = append(raw, wire.EncodeEnvelope(env)...)
			}

			for _, want := range tt.frames {
				got, rest, err := wire.DecodeEnvelope(raw)

				s.Require().NoError(err)
				s.Require().Equal(want, got)

				raw = rest
			}

			s.Require().Empty(raw)
		})
	}
}

func TestWirePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(WirePublicTestSuite))
}
