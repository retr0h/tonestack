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

	"github.com/retr0h/tonestack/pkg/sdk/device/wire"
)

// These are real bytes. Every expectation below was taken from a recorded HX
// Stomp session, so a change that still compiles and still round-trips but no
// longer matches what the device was actually sent will fail here.
type FramePublicTestSuite struct {
	suite.Suite
}

// TestEncodeFrame writes what the device was actually sent.
func (s *FramePublicTestSuite) TestEncodeFrame() {
	tests := []struct {
		name  string
		frame wire.Frame
		want  string
	}{
		{
			"the hello that opens the control channel",
			wire.Frame{
				Flags: wire.FlagHandshake, DeviceNode: 0x1001, HostNode: 0x03ef,
				Seq: 0, Type: wire.MsgHello, Ack: 0x21000100,
				Payload: []byte{0x00, 0x10, 0x00, 0x00},
			},
			"0c0000280110ef03000000020001002100100000",
		},
		{
			"opening a service, whose body is one byte naming it",
			wire.Frame{
				Flags: wire.FlagNormal, DeviceNode: 0x1001, HostNode: 0x03ef,
				Seq: 2, Type: wire.MsgData, Ack: wire.AckBase,
				Payload: wire.EncodeEnvelope(wire.Envelope{
					Originator: wire.FromHost, Service: 5, Body: []byte{0x05},
				}),
			},
			"110000180110ef030002000400100000010005000100000005000000",
		},
		{
			"a bare acknowledgement",
			wire.Frame{
				Flags: wire.FlagNormal, DeviceNode: 0x1001, HostNode: 0x03ef,
				Seq: 3, Type: wire.MsgAck, Ack: wire.AckBase,
			},
			"080000180110ef030003000800100000",
		},
		{
			"the request that lists presets",
			wire.Frame{
				Flags: wire.FlagNormal, DeviceNode: 0x1001, HostNode: 0x03ef,
				Seq: 4, Type: wire.MsgData, Ack: wire.AckBase + 9,
				Payload: wire.EncodeEnvelope(wire.Envelope{
					Originator: wire.FromHost, Service: 2,
					Body: []byte{
						0x83, 0x66, 0xcd, 0x03, 0xe8,
						0x64, 0x01, 0x65, 0x82, 0x6b, 0x00, 0x65, 0x02,
					},
				}),
			},
			"1d0000180110ef030004000409100000010002000d000000" +
				"8366cd03e8640165826b006502000000",
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			s.Require().Equal(tc.want, hex.EncodeToString(wire.EncodeFrame(tc.frame)))
		})
	}
}

// TestDecodeFrame reads frames back out of a transfer.
func (s *FramePublicTestSuite) TestDecodeFrame() {
	tests := []struct {
		name string
		// frames to encode and read back, or bytes written by hand.
		frames []wire.Frame
		raw    []byte
		// bytes dropped off the end of what was encoded.
		trim int
		err  bool
	}{
		{
			name: "a frame it encoded, padding and all",
			frames: []wire.Frame{{
				Flags: wire.FlagNormal, DeviceNode: 0x1001, HostNode: 0x03ef,
				Seq: 7, Type: wire.MsgData, Ack: wire.AckBase + 42,
				Payload: []byte{1, 2, 3},
			}},
		},
		{
			// The device coalesces frames into one bulk transfer, so a reader
			// that decodes only the first silently loses the rest.
			name: "several from one transfer",
			frames: []wire.Frame{
				{DeviceNode: 0x1001, Payload: []byte("one")},
				{DeviceNode: 0x1002, Payload: []byte("two")},
			},
		},
		{
			// Padding is what the host sends. A transfer that simply ends on
			// an unaligned boundary must not be read past.
			name:   "a frame that was not padded",
			frames: []wire.Frame{{DeviceNode: 0x1001, Payload: []byte{1, 2, 3}}},
			trim:   1,
		},
		{name: "nothing at all", err: true},
		{name: "less than two headers", raw: make([]byte, 12), err: true},
		{
			name: "a frame declaring less than a channel header",
			raw:  []byte{0x04, 0, 0, 0x18, 1, 0x10, 0xef, 3, 0, 0, 0, 0, 0, 0, 0, 0},
			err:  true,
		},
		{
			name: "a frame declaring more than arrived",
			raw:  []byte{0xff, 0, 0, 0x18, 1, 0x10, 0xef, 3, 0, 0, 0, 0, 0, 0, 0, 0},
			err:  true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			if tt.err {
				_, _, err := wire.DecodeFrame(tt.raw)

				s.Require().ErrorIs(err, wire.ErrShortTransfer)

				return
			}

			var raw []byte
			for _, f := range tt.frames {
				raw = append(raw, wire.EncodeFrame(f)...)
			}

			raw = raw[:len(raw)-tt.trim]

			for _, want := range tt.frames {
				got, rest, err := wire.DecodeFrame(raw)

				s.Require().NoError(err)
				s.Require().Equal(want, got)

				raw = rest
			}

			s.Require().Empty(raw, "padding is skipped, not read as another frame")
		})
	}
}

func (s *FramePublicTestSuite) TestCarriesData() {
	// The device sets the data bit alongside an acknowledgement and alongside
	// a keep-alive. Comparing for equality drops both.
	tests := []struct {
		name string
		typ  uint16
		want bool
	}{
		{"plain data", wire.MsgData, true},
		{"data with an acknowledgement", wire.MsgData | wire.MsgAck, true},
		{"data on a keep-alive", wire.MsgData | wire.MsgKeepAlive, true},
		{"a bare acknowledgement", wire.MsgAck, false},
		{"a channel opening", wire.MsgHello, false},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			s.Require().Equal(tc.want, wire.Frame{Type: tc.typ}.CarriesData())
		})
	}
}

func TestFramePublicTestSuite(t *testing.T) {
	suite.Run(t, new(FramePublicTestSuite))
}
