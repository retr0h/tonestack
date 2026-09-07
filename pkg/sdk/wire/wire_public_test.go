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
	"bytes"
	"encoding/hex"
	"io"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/wire"
)

type WirePublicTestSuite struct {
	suite.Suite
}

func (s *WirePublicTestSuite) TestEncodeLaysOutTheHeader() {
	got := wire.EncodeEnvelope(wire.Envelope{
		Originator: wire.FromHost,
		Service:    5,
		Body:       []byte{0xde, 0xad},
	})

	// originator 1, service 5, length 2, then the body — all little endian.
	s.Require().Equal("01000500020000 00dead",
		hex.EncodeToString(got[:7])+" "+hex.EncodeToString(got[7:]))
}

func (s *WirePublicTestSuite) TestEncodeAndDecodeAgree() {
	want := wire.Envelope{Originator: wire.FromHost, Service: 2, Body: []byte("hello")}

	got, rest, err := wire.DecodeEnvelope(wire.EncodeEnvelope(want))

	s.Require().NoError(err)
	s.Require().Equal(want, got)
	s.Require().Empty(rest)
}

func (s *WirePublicTestSuite) TestDecodeReturnsWhatFollows() {
	// A bulk read can carry more than one frame, so the remainder has to come
	// back rather than be dropped.
	raw := append(
		wire.EncodeEnvelope(wire.Envelope{Originator: wire.FromDevice, Body: []byte("one")}),
		wire.EncodeEnvelope(wire.Envelope{Originator: wire.FromDevice, Body: []byte("two")})...,
	)

	first, rest, err := wire.DecodeEnvelope(raw)
	s.Require().NoError(err)
	s.Require().Equal([]byte("one"), first.Body)

	second, rest, err := wire.DecodeEnvelope(rest)
	s.Require().NoError(err)
	s.Require().Equal([]byte("two"), second.Body)
	s.Require().Empty(rest)
}

func (s *WirePublicTestSuite) TestOriginatorSaysWhichEndSpoke() {
	// Host frames always carry 1 and device frames always 0, which is the
	// cheapest check that a stream is still aligned.
	f, _, err := wire.DecodeEnvelope(
		wire.EncodeEnvelope(wire.Envelope{Originator: wire.FromDevice}),
	)

	s.Require().NoError(err)
	s.Require().Equal(wire.FromDevice, f.Originator)
}

func (s *WirePublicTestSuite) TestDecodeRefusesWhatCannotBeAFrame() {
	tests := []struct {
		name string
		raw  []byte
		want error
	}{
		{"nothing at all", nil, wire.ErrShortFrame},
		{"half a header", []byte{1, 0, 5, 0}, wire.ErrShortFrame},
		{
			"a header promising more body than arrived",
			[]byte{1, 0, 5, 0, 0x10, 0, 0, 0, 0xde},
			wire.ErrShortFrame,
		},
		{
			"a length no real frame carries",
			[]byte{1, 0, 5, 0, 0xff, 0xff, 0xff, 0xff},
			wire.ErrBodyTooLarge,
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			_, _, err := wire.DecodeEnvelope(tc.raw)

			s.Require().ErrorIs(err, tc.want)
		})
	}
}

func (s *WirePublicTestSuite) TestReadTakesOneFrameFromAStream() {
	stream := bytes.NewReader(append(
		wire.EncodeEnvelope(
			wire.Envelope{Originator: wire.FromDevice, Service: 5, Body: []byte("a")},
		),
		wire.EncodeEnvelope(wire.Envelope{Originator: wire.FromDevice, Body: []byte("b")})...,
	))

	first, err := wire.ReadEnvelope(stream)
	s.Require().NoError(err)
	s.Require().Equal([]byte("a"), first.Body)

	second, err := wire.ReadEnvelope(stream)
	s.Require().NoError(err)
	s.Require().Equal([]byte("b"), second.Body)

	_, err = wire.ReadEnvelope(stream)
	s.Require().ErrorIs(err, io.EOF)
}

func (s *WirePublicTestSuite) TestReadRefusesWhatCannotBeAFrame() {
	tests := []struct {
		name    string
		raw     []byte
		want    error
		message string
	}{
		{"a truncated header", []byte{1, 0, 5}, io.ErrUnexpectedEOF, "header"},
		{
			"a body that never arrives",
			[]byte{1, 0, 5, 0, 0x10, 0, 0, 0},
			io.EOF, "body",
		},
		{
			"a length no real frame carries",
			[]byte{1, 0, 5, 0, 0xff, 0xff, 0xff, 0xff},
			wire.ErrBodyTooLarge, "too large",
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			_, err := wire.ReadEnvelope(bytes.NewReader(tc.raw))

			s.Require().ErrorIs(err, tc.want)
			s.Require().Contains(err.Error(), tc.message)
		})
	}
}

func TestWirePublicTestSuite(t *testing.T) {
	suite.Run(t, new(WirePublicTestSuite))
}
