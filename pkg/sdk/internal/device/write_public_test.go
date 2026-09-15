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
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/vmihailenco/msgpack/v5"
	"go.uber.org/mock/gomock"

	"github.com/retr0h/tonestack/pkg/sdk/internal/device"
	"github.com/retr0h/tonestack/pkg/sdk/internal/wire"
)

// WritePublicTestSuite covers putting a preset on a device.
//
// Nothing here reaches hardware. What it checks are the two rules a device
// enforces and punishes: a message goes out in pieces it can pace, and a
// write is not finished when it says it was accepted.
type WritePublicTestSuite struct {
	suite.Suite

	ctrl *gomock.Controller
}

// answer encodes a reply carrying one status.
func (s *WritePublicTestSuite) answer(
	txn uint64,
	status int,
) []byte {
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
func (s *WritePublicTestSuite) completes() *deviceDouble {
	return answers(s.ctrl,
		s.answer(device.FirstTxn, 1),
		s.answer(device.FirstTxn, 0),
	)
}

// SetupTest makes the controller. The test budgets drop the flash settle,
// which is a real wait on hardware and dead time here, and give a write a
// commit budget that outlasts the reply budget, as on hardware.
func (s *WritePublicTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
}

// stream puts back together the message a session sent, from the data frames
// it went out in, and returns the body of its envelope.
func (s *WritePublicTestSuite) stream(
	d *deviceDouble,
) []byte {
	var joined []byte

	for _, raw := range d.frames() {
		f, _, err := wire.DecodeFrame(raw)
		s.Require().NoError(err)

		if f.Type == wire.MsgData {
			joined = append(joined, f.Payload...)
		}
	}

	env, _, err := wire.DecodeEnvelope(joined)
	s.Require().NoError(err, "the whole envelope arrived")

	return env.Body
}

// session returns one with its channels open over a scripted device.
func (s *WritePublicTestSuite) session(
	d *deviceDouble,
) *device.Session {
	out := device.NewTestSession(s.T(), d.out, d.in)
	out.OpenChannels()

	return out
}

// TestWritePreset puts a document into a slot.
//
// Both 0 and 1 have been seen on hardware for a write that landed, so either
// is success: the erase and program that follow never reach the wire. A
// refusal, or a status nobody has seen, fails.
func (s *WritePublicTestSuite) TestWritePreset() {
	large := bytes.Repeat([]byte{0x2a}, 2000)

	tests := []struct {
		name     string
		device   func() *deviceDouble
		document []byte
		// a caller who stopped waiting before the write, and one who stops
		// once the first chunk has gone.
		cancelled      bool
		cancelsOnWrite bool
		// dead is a session whose read loop has already ended when the write
		// is asked for.
		dead bool
		// nothing reached the device, or all of the message did.
		nothingSent bool
		whole       bool
		// a commit budget of its own, when that is the point.
		commitBudget time.Duration
		// a call's reply budget of its own, shorter than the commit budget.
		replyBudget time.Duration
		is          error
		says        string
	}{
		{
			// The commit budget bounds waiting for the answer, never sending
			// the message. A budget that ran out between chunks left the
			// device holding half a preset, which is the stall a started
			// write exists to prevent.
			name: "a commit budget that runs out while the message is going out",
			device: func() *deviceDouble {
				return answers(s.ctrl)
			},
			document:     large,
			commitBudget: 20 * time.Millisecond,
			whole:        true,
			is:           context.DeadlineExceeded,
			says:         "commit budget",
		},
		{
			// Nothing has gone out, so nothing is owed: the write is not
			// started.
			name:        "a caller who stopped waiting before it began",
			device:      func() *deviceDouble { return answers(s.ctrl, s.answer(device.FirstTxn, 0)) },
			cancelled:   true,
			nothingSent: true,
			is:          context.Canceled,
		},
		{
			// No read is posted to catch the answer, so not one chunk goes
			// out, and the error that ended the loop is what the write says.
			name:        "a session the bus has already ended",
			device:      func() *deviceDouble { return readFails(s.ctrl, errors.New("the bus went away")) },
			dead:        true,
			nothingSent: true,
			is:          device.ErrBus,
			says:        "reading from the device",
		},
		{
			// A device fed half a message and then a burst is the stall that
			// needs a power cycle. Once the first chunk is out, the message
			// is finished and its answer read, whoever stopped waiting.
			name:           "a caller who stops waiting after the first chunk",
			device:         func() *deviceDouble { return answers(s.ctrl, s.answer(device.FirstTxn, 0)) },
			document:       large,
			cancelsOnWrite: true,
			whole:          true,
		},
		{
			// A read between chunks is what paces the sender, so one that
			// fails stops the message where it is: the device is owed
			// nothing more once nothing is left to pace it, and a chunk sent
			// unpaced into a pedal that stopped answering is the stall a
			// hardware trace showed.
			name: "a bus that cannot be read from partway through",
			device: func() *deviceDouble {
				return readFailsAfter(s.ctrl, errors.New("the bus went away"), 1)
			},
			document: large,
			is:       device.ErrBus,
			says:     "reading from the device",
		},
		{
			// A write is waited on for the commit budget, not the shorter one
			// a call gets. A device still committing answers late, and giving
			// up on it at the reply budget races the next write against it.
			name: "a device that answers after a call would have given up",
			device: func() *deviceDouble {
				return late(s.ctrl, 300*time.Millisecond, s.answer(device.FirstTxn, 0))
			},
			replyBudget:  20 * time.Millisecond,
			commitBudget: time.Minute,
		},
		{
			name:   "a device that takes it and gets on with the erase",
			device: func() *deviceDouble { return answers(s.ctrl, s.answer(device.FirstTxn, 1)) },
		},
		{
			// Nothing says a device must defer. One that answers done is
			// done.
			name:   "one that says it finished",
			device: func() *deviceDouble { return answers(s.ctrl, s.answer(device.FirstTxn, 0)) },
		},
		{
			// A device sends notifications unasked while a write commits.
			// One that will not decode, and one carrying another
			// transaction, are both somebody else's business.
			name: "one talking about something else at the same time",
			device: func() *deviceDouble {
				return answers(s.ctrl,
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
			device: func() *deviceDouble { return answers(s.ctrl, s.answer(device.FirstTxn, 255)) },
			is:     wire.ErrRefused,
		},
		{
			// Saying nothing at all is a different thing from answering and
			// getting on with the erase.
			name:   "one that never answers",
			device: func() *deviceDouble { return answers(s.ctrl) },
			says:   "no reply",
		},
		{
			name:   "a bus that cannot be written to",
			device: func() *deviceDouble { return writeFails(s.ctrl, errors.New("boom")) },
			says:   "boom",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			d := tt.device()

			if tt.cancelled {
				cancel()
			}

			if tt.cancelsOnWrite {
				d.onWrite = cancel
			}

			b := device.ShortBudgets()

			if tt.commitBudget > 0 {
				b.Commit = tt.commitBudget
			}

			if tt.replyBudget > 0 {
				b.Reply = tt.replyBudget
			}

			document := tt.document
			if document == nil {
				document = []byte{0x01}
			}

			session := device.NewTestSessionWith(s.T(), d.out, d.in, b)
			session.OpenChannels()

			if tt.dead {
				ended(s.T(), session)
			}

			err := session.WritePreset(ctx, 0, 3, document)

			if tt.nothingSent {
				s.Require().Empty(d.frames(), "nothing reaches the device")
			}

			if tt.whole {
				s.Require().Zero(d.pending(), "the answer was read")
				s.Require().Contains(string(s.stream(d)), string(document),
					"every chunk of the message went out")
			}

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
	sent := d.frames()

	s.Require().GreaterOrEqual(len(sent), 2000/device.StreamChunk,
		"one frame would not have fitted")

	for i, frame := range sent {
		s.Require().LessOrEqual(len(frame), device.StreamChunk+wire.FrameSize+wire.ChannelSize,
			"frame %d is larger than the device takes", i)
	}
}

// TestAChunkTheDeviceNeverAcksStopsTheMessage reproduces what stalled a
// pedal's USB endpoint on hardware: a chunk released by a frame on another
// channel rather than its own acknowledgement, and nothing acking after it.
// docs/protocol.md cites the trace.
//
// A timeout here does not merely fail the write. The data channel would be
// left holding half a message, and a later call feeding it a fresh request
// is the held-session stall all over again, so the timeout ends the session:
// the error matches ErrBus, the next call sends nothing, and Close still
// attempts the closing hellos.
func (s *WritePublicTestSuite) TestAChunkTheDeviceNeverAcksStopsTheMessage() {
	d := answers(s.ctrl)
	d.stopAckingAfter(device.DataChannel, 4)
	d.insteadOfAck(device.DataChannel,
		device.FrameFor(device.ControlChannel, wire.MsgData, []byte("something else")))

	b := device.ShortBudgets()
	b.Pace = 20 * time.Millisecond

	session := device.NewTestSessionWith(s.T(), d.out, d.in, b)
	session.OpenChannels()

	err := session.WritePreset(
		context.Background(), 0, 3, bytes.Repeat([]byte{0x2a}, 2000))

	s.Require().Error(err)
	s.Require().ErrorIs(err, device.ErrBus)
	s.Require().ErrorIs(err, device.ErrUnacked)
	s.Require().ErrorIs(err, context.DeadlineExceeded)
	s.Require().ErrorContains(err, "stopped taking the message")

	chunks := 0

	for _, raw := range d.frames() {
		if kind, _ := device.MessageKind(raw); kind == wire.MsgData {
			chunks++
		}
	}

	s.Require().LessOrEqual(chunks, 5, "nothing went out after the unacked chunk")

	before := len(d.frames())

	_, err = session.Call(context.Background(), device.ControlChannel, 1, nil)
	s.Require().ErrorIs(err, device.ErrBus)
	s.Require().Len(d.frames(), before, "nothing is asked of a session the timeout ended")

	s.Require().ErrorIs(session.Close(), device.ErrBus)

	hellos := 0

	for _, raw := range d.frames()[before:] {
		if kind, _ := device.MessageKind(raw); kind == wire.MsgHello {
			hellos++
		}
	}

	s.Require().Equal(len(device.ChannelNames()), hellos, "the farewell was attempted")
}

// TestAnUnrelatedFrameDoesNotReleaseTheNextChunk covers a notification that
// arrives on another channel while a message paces: pace waits for its own
// channel's acknowledgement, not any transfer.
//
// A count of chunks that all eventually went out would pass on the old logic
// too, which released a chunk on any transfer, the unrelated frame included.
// What proves the fix is the order: chunk N+1 is never written until chunk
// N's own acknowledgement has been served, never merely a frame on some other
// channel.
func (s *WritePublicTestSuite) TestAnUnrelatedFrameDoesNotReleaseTheNextChunk() {
	d := s.completes()

	var once sync.Once

	d.onWrite = func() {
		once.Do(func() {
			d.tell(device.FrameFor(device.ControlChannel, wire.MsgData, []byte("unasked")))
		})
	}

	err := s.session(d).WritePreset(
		context.Background(), 0, 3, bytes.Repeat([]byte{0x2a}, 2000))

	s.Require().NoError(err)

	events := d.eventLog()

	s.Require().GreaterOrEqual(len(events), 2, "more than one chunk went out")

	want := "write"

	for i, e := range events {
		s.Require().Equal(want, e, "event %d out of order: %v", i, events)

		if want == "write" {
			want = "ack"
		} else {
			want = "write"
		}
	}
}

// TestAWriteIsPacedForTheFlash covers the wait that is real.
//
// Nothing on the wire says when the erase and program finish, so a second
// write landing on the first stacks its commit. The pause is the only thing
// keeping them apart.
func (s *WritePublicTestSuite) TestAWriteIsPacedForTheFlash() {
	b := device.ShortBudgets()
	b.Flash = 40 * time.Millisecond

	d := answers(s.ctrl, s.answer(device.FirstTxn, 0))

	session := device.NewTestSessionWith(s.T(), d.out, d.in, b)
	session.OpenChannels()

	started := time.Now()

	s.Require().NoError(session.WritePreset(context.Background(), 0, 3, []byte{0x01}))

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

	s.Require().Contains(string(bytes.Join(d.frames(), nil)), "Mike Dirnt\x00",
		"a device reads an unterminated name as running into what follows")
}

// TestAWriteOnAChannelNobodyOpened covers a session that never handshook.
func (s *WritePublicTestSuite) TestAWriteOnAChannelNobodyOpened() {
	d := s.completes()

	err := device.NewTestSession(s.T(), d.out, d.in).Write(context.Background(), 5, nil)

	s.Require().ErrorContains(err, "no data channel")
}

func TestWriteTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(WritePublicTestSuite))
}
