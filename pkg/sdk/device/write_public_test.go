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
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/vmihailenco/msgpack/v5"

	"github.com/retr0h/tonestack/pkg/sdk/device"
	"github.com/retr0h/tonestack/pkg/sdk/device/wire"
)

// WritePublicTestSuite covers putting a preset on a device.
//
// Nothing here reaches hardware. What it checks are the two rules a device
// enforces and punishes: a message goes out in pieces it can pace, and a
// write is not finished when it says it was accepted.
type WritePublicTestSuite struct {
	suite.Suite
}

// answer encodes a reply carrying one status.
func (s *WritePublicTestSuite) answer(txn uint64, status int) []byte {
	var buf bytes.Buffer

	enc := msgpack.NewEncoder(&buf)
	s.Require().NoError(enc.EncodeMapLen(2))
	s.Require().NoError(enc.EncodeInt(102))
	s.Require().NoError(enc.EncodeUint(txn))
	s.Require().NoError(enc.EncodeInt(103))
	s.Require().NoError(enc.EncodeInt(int64(status)))

	return device.Reply(device.DataChannel, buf.Bytes())
}

// accepted then done is what a device says about a write it completed.
func (s *WritePublicTestSuite) completes() *scripted {
	return answers(
		s.answer(device.FirstTxn, 1),
		s.answer(device.FirstTxn, 0),
	)
}

// SetupTest shortens the wait for a commit, since no test here is waiting on
// hardware.
func (s *WritePublicTestSuite) SetupTest() {
	was := *device.CommitBudget
	*device.CommitBudget = 50 * time.Millisecond

	// The flash settle is a real wait on hardware and dead time here.
	flash := *device.FlashBudget
	*device.FlashBudget = 0

	s.T().Cleanup(func() {
		*device.CommitBudget = was
		*device.FlashBudget = flash
	})
}

// session returns one with its channels open over a scripted device.
func (s *WritePublicTestSuite) session(d *scripted) *device.Session {
	out := device.NewTestSession(d, d)
	out.OpenChannels()

	return out
}

// TestWritePreset puts a document into a slot.
//
// Both statuses have been seen on hardware for a write that landed, so
// neither is read: the erase and program that follow never reach the wire.
func (s *WritePublicTestSuite) TestWritePreset() {
	tests := []struct {
		name   string
		device func() *scripted
		is     error
		says   string
	}{
		{
			name:   "a device that takes it and gets on with the erase",
			device: func() *scripted { return answers(s.answer(device.FirstTxn, 1)) },
		},
		{
			// Nothing says a device must defer. One that answers done is
			// done.
			name:   "one that says it finished",
			device: func() *scripted { return answers(s.answer(device.FirstTxn, 0)) },
		},
		{
			// A device sends notifications unasked while a write commits.
			// One that will not decode, and one carrying another
			// transaction, are both somebody else's business.
			name: "one talking about something else at the same time",
			device: func() *scripted {
				return answers(
					s.answer(device.FirstTxn, 1),
					device.Reply(device.ControlChannel, []byte{0xc1}),
					s.answer(device.FirstTxn+7, 0),
					s.answer(device.FirstTxn, 0),
				)
			},
		},
		{
			// What a wrongly tagged document drew: the device answers, and
			// what it answers is no.
			name:   "one that refuses it",
			device: func() *scripted { return answers(s.answer(device.FirstTxn, 255)) },
			is:     wire.ErrRefused,
		},
		{
			// Saying nothing at all is a different thing from answering and
			// getting on with the erase.
			name:   "one that never answers",
			device: func() *scripted { return answers() },
			says:   "no reply",
		},
		{
			name:   "a bus that cannot be written to",
			device: func() *scripted { return &scripted{writeErr: errors.New("boom")} },
			says:   "boom",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			err := s.session(tt.device()).WritePreset(
				context.Background(), 0, 3, []byte{0x01})

			if tt.is == nil && tt.says == "" {
				s.Require().NoError(err)

				return
			}

			s.Require().Error(err)

			if tt.is != nil {
				s.Require().ErrorIs(err, tt.is)
			}

			if tt.says != "" {
				s.Require().Contains(err.Error(), tt.says)
			}
		})
	}
}

// TestAMessageGoesOutInPiecesADeviceCanPace is a property of the transfer
// rather than a case of the call.
//
// A device takes 256 bytes of stream data per frame and paces the sender with
// acknowledgements. Sending a whole preset at once fills its receive window
// and stalls the endpoint, and the interface will not be claimed again until
// the device is power cycled.
func (s *WritePublicTestSuite) TestAMessageGoesOutInPiecesADeviceCanPace() {
	d := s.completes()

	err := s.session(d).WritePreset(
		context.Background(), 0, 3, bytes.Repeat([]byte{0x2a}, 2000))

	s.Require().NoError(err)

	// Two thousand bytes of document, plus the envelope around it, in pieces
	// of 256.
	s.Require().GreaterOrEqual(len(d.sent), 2000/device.StreamChunk,
		"one frame would not have fitted")

	for i, frame := range d.sent {
		s.Require().LessOrEqual(len(frame), device.StreamChunk+wire.FrameSize+wire.ChannelSize,
			"frame %d is larger than the device takes", i)
	}
}

// TestAWriteIsPacedForTheFlash covers the wait that is real.
//
// Nothing on the wire says when the erase and program finish, so a second
// write landing on the first stacks its commit. The pause is the only thing
// keeping them apart.
func (s *WritePublicTestSuite) TestAWriteIsPacedForTheFlash() {
	was := *device.FlashBudget
	*device.FlashBudget = 40 * time.Millisecond

	defer func() { *device.FlashBudget = was }()

	started := time.Now()

	s.Require().NoError(s.session(answers(s.answer(device.FirstTxn, 0))).
		WritePreset(context.Background(), 0, 3, []byte{0x01}))

	s.Require().GreaterOrEqual(time.Since(started), 40*time.Millisecond)
}

// TestWriteNamedPreset carries the name the slot takes.
//
// A paste or an import carries one; editing a preset in place leaves whatever
// the slot was called.
func (s *WritePublicTestSuite) TestWriteNamedPreset() {
	d := s.completes()

	s.Require().NoError(s.session(d).WriteNamedPreset(
		context.Background(), 0, 3, "Mike Dirnt", []byte{0x01}))

	s.Require().Contains(string(bytes.Join(d.sent, nil)), "Mike Dirnt\x00",
		"a device reads an unterminated name as running into what follows")
}

// TestAWriteOnAChannelNobodyOpened covers a session that never handshook.
func (s *WritePublicTestSuite) TestAWriteOnAChannelNobodyOpened() {
	d := s.completes()

	err := device.NewTestSession(d, d).Write(context.Background(), 5, nil)

	s.Require().ErrorContains(err, "no data channel")
}

func TestWriteTestSuite(t *testing.T) {
	suite.Run(t, new(WritePublicTestSuite))
}
