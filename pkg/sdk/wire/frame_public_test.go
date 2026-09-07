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

	"github.com/retr0h/tonestack/pkg/sdk/wire"
)

// These are real bytes. Every expectation below was taken from a recorded HX
// Stomp session, so a change that still compiles and still round-trips but no
// longer matches what the device was actually sent will fail here.
type FramePublicTestSuite struct {
	suite.Suite
}

func (s *FramePublicTestSuite) TestEncodesTheCapturedHandshake() {
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

func (s *FramePublicTestSuite) TestDecodesWhatItEncodes() {
	want := wire.Frame{
		Flags: wire.FlagNormal, DeviceNode: 0x1001, HostNode: 0x03ef,
		Seq: 7, Type: wire.MsgData, Ack: wire.AckBase + 42,
		Payload: []byte{1, 2, 3},
	}

	got, rest, err := wire.DecodeFrame(wire.EncodeFrame(want))

	s.Require().NoError(err)
	s.Require().Equal(want, got)
	s.Require().Empty(rest, "padding is skipped, not returned as another frame")
}

func (s *FramePublicTestSuite) TestDecodesSeveralFromOneTransfer() {
	// The device coalesces frames into one bulk transfer, so a reader that
	// decodes only the first silently loses the rest.
	raw := append(
		wire.EncodeFrame(wire.Frame{DeviceNode: 0x1001, Payload: []byte("one")}),
		wire.EncodeFrame(wire.Frame{DeviceNode: 0x1002, Payload: []byte("two")})...,
	)

	first, rest, err := wire.DecodeFrame(raw)
	s.Require().NoError(err)
	s.Require().Equal([]byte("one"), first.Payload)

	second, rest, err := wire.DecodeFrame(rest)
	s.Require().NoError(err)
	s.Require().Equal([]byte("two"), second.Payload)
	s.Require().Empty(rest)
}

func (s *FramePublicTestSuite) TestToleratesAFrameThatWasNotPadded() {
	// Padding is what the host sends. A transfer that simply ends on an
	// unaligned boundary must not be read past.
	raw := wire.EncodeFrame(wire.Frame{DeviceNode: 0x1001, Payload: []byte{1, 2, 3}})

	got, rest, err := wire.DecodeFrame(raw[:len(raw)-1])

	s.Require().NoError(err)
	s.Require().Equal([]byte{1, 2, 3}, got.Payload)
	s.Require().Empty(rest)
}

func (s *FramePublicTestSuite) TestCarriesDataIsTestedAsABit() {
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

func (s *FramePublicTestSuite) TestRefusesWhatCannotBeAFrame() {
	tests := []struct {
		name string
		raw  []byte
	}{
		{"nothing at all", nil},
		{"less than two headers", make([]byte, 12)},
		{
			"a frame declaring less than a channel header",
			[]byte{0x04, 0, 0, 0x18, 1, 0x10, 0xef, 3, 0, 0, 0, 0, 0, 0, 0, 0},
		},
		{
			"a frame declaring more than arrived",
			[]byte{0xff, 0, 0, 0x18, 1, 0x10, 0xef, 3, 0, 0, 0, 0, 0, 0, 0, 0},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			_, _, err := wire.DecodeFrame(tc.raw)

			s.Require().ErrorIs(err, wire.ErrShortTransfer)
		})
	}
}

func TestFramePublicTestSuite(t *testing.T) {
	suite.Run(t, new(FramePublicTestSuite))
}
