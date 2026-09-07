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
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/vmihailenco/msgpack/v5"

	"github.com/retr0h/tonestack/pkg/sdk"
	"github.com/retr0h/tonestack/pkg/sdk/wire"
)

// WriteTestSuite covers putting a preset on a device.
//
// Nothing here reaches hardware. What it checks are the two rules a device
// enforces and punishes: a message goes out in pieces it can pace, and a
// write is not finished when it says it was accepted.
type WriteTestSuite struct {
	suite.Suite
}

// answer encodes a reply carrying one status.
func (s *WriteTestSuite) answer(txn uint64, status int) []byte {
	var buf bytes.Buffer

	enc := msgpack.NewEncoder(&buf)
	s.Require().NoError(enc.EncodeMapLen(2))
	s.Require().NoError(enc.EncodeInt(102))
	s.Require().NoError(enc.EncodeUint(txn))
	s.Require().NoError(enc.EncodeInt(103))
	s.Require().NoError(enc.EncodeInt(int64(status)))

	return sdk.Reply(sdk.DataChannel, buf.Bytes())
}

// accepted then done is what a device says about a write it completed.
func (s *WriteTestSuite) completes() *device {
	return answers(
		s.answer(sdk.FirstTxn, 1),
		s.answer(sdk.FirstTxn, 0),
	)
}

// SetupTest shortens the wait for a commit, since no test here is waiting on
// hardware.
func (s *WriteTestSuite) SetupTest() {
	was := *sdk.CommitBudget
	*sdk.CommitBudget = 50 * time.Millisecond

	// The flash settle is a real wait on hardware and dead time here.
	flash := *sdk.FlashBudget
	*sdk.FlashBudget = 0

	s.T().Cleanup(func() {
		*sdk.CommitBudget = was
		*sdk.FlashBudget = flash
	})
}

// session returns one with its channels open over a scripted device.
func (s *WriteTestSuite) session(d *device) *sdk.Session {
	out := sdk.NewTestSession(d, d)
	out.OpenChannels()

	return out
}

func (s *WriteTestSuite) TestAMessageGoesOutInPiecesADeviceCanPace() {
	// A device takes 256 bytes of stream data per frame and paces the sender
	// with acknowledgements. Sending a whole preset at once fills its receive
	// window and stalls the endpoint, and the interface will not be claimed
	// again until the device is power cycled.
	d := s.completes()

	err := s.session(d).WritePreset(
		context.Background(), 0, 3, bytes.Repeat([]byte{0x2a}, 2000))

	s.Require().NoError(err)

	// Two thousand bytes of document, plus the envelope around it, in pieces
	// of 256.
	s.Require().GreaterOrEqual(len(d.sent), 2000/sdk.StreamChunk,
		"one frame would not have fitted")

	for i, frame := range d.sent {
		s.Require().LessOrEqual(len(frame), sdk.StreamChunk+wire.FrameSize+wire.ChannelSize,
			"frame %d is larger than the device takes", i)
	}
}

func (s *WriteTestSuite) TestAWriteWaitsForTheDeviceToFinish() {
	// The reply says only that the device took it. A client that treats that
	// as completion races its next write against a commit still running, and
	// a device tolerates about a dozen of those before it stops accepting
	// writes at all.
	d := s.completes()

	s.Require().NoError(s.session(d).WritePreset(
		context.Background(), 0, 3, []byte{0x01}))

	s.Require().Empty(d.replies, "both answers were read")
}

func (s *WriteTestSuite) TestAWriteThatFinishesImmediately() {
	// Nothing says a device must defer. One that answers `done` is done.
	d := answers(s.answer(sdk.FirstTxn, 0))

	s.Require().NoError(s.session(d).WritePreset(
		context.Background(), 0, 3, []byte{0x01}))
}

func (s *WriteTestSuite) TestReportsAWriteThatNeverFinishes() {
	// A device that takes a write and goes quiet is not a device that
	// finished, and saying so beats reporting success.
	d := answers(s.answer(sdk.FirstTxn, 1))

	err := s.session(d).WritePreset(context.Background(), 0, 3, []byte{0x01})

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "never said it finished")
}

func (s *WriteTestSuite) TestReportsAWriteTheDeviceRefuses() {
	d := answers(s.answer(sdk.FirstTxn, 255))

	err := s.session(d).WritePreset(context.Background(), 0, 3, []byte{0x01})

	s.Require().ErrorIs(err, wire.ErrRefused)
}

func (s *WriteTestSuite) TestReportsABusItCannotWriteTo() {
	err := s.session(&device{writeErr: errors.New("boom")}).WritePreset(
		context.Background(), 0, 3, []byte{0x01})

	s.Require().Error(err)
}

func (s *WriteTestSuite) TestAWriteOnAChannelNobodyOpened() {
	d := s.completes()

	err := sdk.NewTestSession(d, d).Write(context.Background(), 5, nil)

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "no data channel")
}

func (s *WriteTestSuite) TestNamingWhatItWrites() {
	// A paste or an import carries the name; editing a preset in place leaves
	// whatever the slot was called.
	d := s.completes()

	s.Require().NoError(s.session(d).WriteNamedPreset(
		context.Background(), 0, 3, "Mike Dirnt", []byte{0x01}))

	joined := bytes.Join(d.sent, nil)
	s.Require().Contains(string(joined), "Mike Dirnt\x00",
		"a device reads an unterminated name as running into what follows")
}

func (s *WriteTestSuite) TestACommitBesideAnswersMeantForSomebodyElse() {
	// A device sends notifications unasked while a write is committing. One
	// that will not decode, and one carrying another transaction, are both
	// somebody else's business.
	d := answers(
		s.answer(sdk.FirstTxn, 1),
		sdk.Reply(sdk.ControlChannel, []byte{0xc1}),
		s.answer(sdk.FirstTxn+7, 0),
		s.answer(sdk.FirstTxn, 0),
	)

	s.Require().NoError(s.session(d).WritePreset(
		context.Background(), 0, 3, []byte{0x01}))
}

func (s *WriteTestSuite) TestADeviceThatRefusesWhileCommitting() {
	d := answers(
		s.answer(sdk.FirstTxn, 1),
		s.answer(sdk.FirstTxn, 255),
	)

	err := s.session(d).WritePreset(context.Background(), 0, 3, []byte{0x01})

	s.Require().ErrorIs(err, wire.ErrRefused)
}

func (s *WriteTestSuite) TestABusThatStopsListeningWhileCommitting() {
	// Bytes that arrived are acknowledged. A device that stops taking those
	// has to be reported: an unacknowledged stream stalls.
	d := answers(s.answer(sdk.FirstTxn, 1), s.answer(sdk.FirstTxn+7, 0))
	out := &sdk.FailAfter{Sender: d, OK: 1, Err: errors.New("boom")}

	session := sdk.NewTestSession(out, d)
	session.OpenChannels()

	err := session.WritePreset(context.Background(), 0, 3, []byte{0x01})

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "boom")
}

func TestWriteTestSuite(t *testing.T) {
	suite.Run(t, new(WriteTestSuite))
}
