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

// LoopPublicTestSuite covers the read loop: the one goroutine that reads, and
// what it keeps true while nothing else is reading.
//
// A device sends notifications unasked, and with nothing draining its
// endpoint its queue fills and it stops taking writes. Rule 2 in
// docs/protocol.md.
type LoopPublicTestSuite struct {
	suite.Suite

	ctrl *gomock.Controller
}

func (s *LoopPublicTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
}

// status is a device answering one transaction with a status and nothing else.
func (s *LoopPublicTestSuite) status(
	channel string,
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

	return device.Reply(channel, buf.Bytes())
}

// playing is a device saying which preset it has loaded.
func (s *LoopPublicTestSuite) playing(
	txn uint64,
	slot int,
) []byte {
	var buf bytes.Buffer

	enc := msgpack.NewEncoder(&buf)
	s.Require().NoError(enc.EncodeMapLen(3))
	s.Require().NoError(enc.EncodeInt(102))
	s.Require().NoError(enc.EncodeUint(txn))
	s.Require().NoError(enc.EncodeInt(103))
	s.Require().NoError(enc.EncodeInt(0))
	s.Require().NoError(enc.EncodeInt(104))
	s.Require().NoError(enc.EncodeMapLen(3))
	s.Require().NoError(enc.EncodeInt(107))
	s.Require().NoError(enc.EncodeInt(0))
	s.Require().NoError(enc.EncodeInt(108))
	s.Require().NoError(enc.EncodeInt(int64(slot)))
	s.Require().NoError(enc.EncodeInt(109))
	s.Require().NoError(enc.EncodeString("Chunky Monkey"))

	return device.Reply(device.DataChannel, buf.Bytes())
}

// unaskedLater makes the device say something on the control channel a
// little after it is first written to, while whatever that write started is
// still waiting.
func unaskedLater(
	d *deviceDouble,
	after time.Duration,
) {
	var once sync.Once

	d.onWrite = func() {
		once.Do(func() {
			time.AfterFunc(after, func() {
				d.tell(device.FrameFor(device.ControlChannel, wire.MsgData, []byte("unasked")))
			})
		})
	}
}

// TestLoop keeps a read posted from start to Close, and ends the session
// when the bus does.
func (s *LoopPublicTestSuite) TestLoop() {
	broken := errors.New("the bus went away")

	tests := []struct {
		name    string
		device  func() *deviceDouble
		budgets func(b *device.Budgets)
		run     func(session *device.Session, d *deviceDouble)
	}{
		{
			// Nothing is running, and the device is still read. Before the
			// loop, this sat on the endpoint until somebody asked something.
			name: "a notification between two calls",
			device: func() *deviceDouble {
				return answers(s.ctrl, s.status(device.ControlChannel, device.FirstTxn, 0))
			},
			run: func(session *device.Session, d *deviceDouble) {
				_, err := session.Call(context.Background(), device.ControlChannel, 1, nil)
				s.Require().NoError(err)

				before := session.Received(device.ControlChannel)
				d.tell(device.FrameFor(device.ControlChannel, wire.MsgData, []byte("unasked")))

				s.Require().Eventually(func() bool {
					return session.Received(device.ControlChannel) > before
				}, time.Second, time.Millisecond, "read with no operation running")
			},
		},
		{
			// The flash pause waits on a timer, not by sleeping the only
			// reader.
			name: "a notification during the flash pause",
			device: func() *deviceDouble {
				d := answers(s.ctrl, s.status(device.DataChannel, device.FirstTxn, 0))
				unaskedLater(d, 40*time.Millisecond)

				return d
			},
			budgets: func(b *device.Budgets) { b.Flash = 200 * time.Millisecond },
			run: func(session *device.Session, _ *deviceDouble) {
				s.Require().NoError(session.WritePreset(context.Background(), 0, 3, []byte{0x01}))
				s.Require().NotZero(session.Received(device.ControlChannel),
					"read while the write settled")
			},
		},
		{
			name: "a notification while a switch is polled",
			device: func() *deviceDouble {
				d := answers(s.ctrl,
					s.status(device.DataChannel, device.FirstTxn, 1),
					s.playing(device.FirstTxn+1, 5),
					s.playing(device.FirstTxn+2, 99),
				)
				unaskedLater(d, 40*time.Millisecond)

				return d
			},
			budgets: func(b *device.Budgets) {
				b.Poll, b.Selecting = 200*time.Millisecond, 2*time.Second
			},
			run: func(session *device.Session, _ *deviceDouble) {
				s.Require().NoError(session.SelectPreset(context.Background(), 0, 99))
				s.Require().NotZero(session.Received(device.ControlChannel),
					"read between the questions")
			},
		},
		{
			// A timeout is the device with nothing to say, however often it
			// happens.
			name:   "reads that time out",
			device: func() *deviceDouble { return readFails(s.ctrl, context.DeadlineExceeded) },
			run: func(session *device.Session, _ *deviceDouble) {
				start := session.Windows()

				s.Require().Eventually(func() bool {
					return session.Windows() > start+5
				}, time.Second, time.Millisecond)
				s.Require().NoError(session.Ended())
			},
		},
		{
			// The waiter hears about it at once rather than at the end of its
			// budget, and nothing more is asked of a device nobody is
			// reading.
			name:    "a bus that fails while a call waits",
			device:  func() *deviceDouble { return readFailsAfter(s.ctrl, broken, 1) },
			budgets: func(b *device.Budgets) { b.Reply = 5 * time.Second },
			run: func(session *device.Session, d *deviceDouble) {
				started := time.Now()

				_, err := session.Call(context.Background(), device.ControlChannel, 1, nil)
				s.Require().ErrorIs(err, broken)
				s.Require().ErrorIs(err, device.ErrBus)
				s.Require().ErrorContains(err, "reading from the device")
				s.Require().Less(time.Since(started), time.Second)

				sent := len(d.frames())

				_, err = session.Call(context.Background(), device.ControlChannel, 1, nil)
				s.Require().ErrorIs(err, broken)
				s.Require().Len(d.frames(), sent, "nothing is asked of a bus that failed")
			},
		},
		{
			// Recovered, and ended the way a bus error ends it, so the caller
			// gets an error rather than a process that died.
			name: "a panic in routing while a call waits",
			device: func() *deviceDouble {
				return answers(s.ctrl, s.status(device.ControlChannel, device.FirstTxn, 0))
			},
			budgets: func(b *device.Budgets) { b.Reply = 5 * time.Second },
			run: func(session *device.Session, _ *deviceDouble) {
				session.Trace(panicking{})

				_, err := session.Call(context.Background(), device.ControlChannel, 1, nil)
				s.Require().ErrorIs(err, device.ErrBus)
				s.Require().ErrorContains(err, "the read loop panicked")
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			d := tt.device()

			b := device.ShortBudgets()
			if tt.budgets != nil {
				tt.budgets(&b)
			}

			session := device.NewTestSessionWith(s.T(), d.out, d.in, b)
			session.OpenChannels()

			tt.run(session, d)
		})
	}
}

// acks counts the acknowledgements a session sent on a channel, and says what
// the last of them carried.
func acks(
	d *deviceDouble,
	channel string,
) (int, uint32) {
	var (
		count int
		last  uint32
	)

	for _, raw := range d.frames() {
		f, _, err := wire.DecodeFrame(raw)
		if err != nil || f.Type != wire.MsgAck || device.ChannelOf(f) != channel {
			continue
		}

		count++
		last = f.Ack
	}

	return count, last
}

// TestIdleAck settles what arrives on a channel nobody is using.
func (s *LoopPublicTestSuite) TestIdleAck() {
	answer := device.Reply(device.ControlChannel, []byte{0x80})
	whole := wire.EncodeEnvelope(wire.Envelope{
		Originator: wire.FromDevice, Service: 2, Body: bytes.Repeat([]byte{0x01}, 100),
	})
	partial := device.FrameFor(device.ControlChannel, wire.MsgData, whole[:20])
	events := device.FrameFor(device.EventsChannel, wire.MsgData, []byte("noise"))

	tests := []struct {
		name    string
		frame   []byte
		channel string
		// payload is how many stream bytes the frame carries, and buffered
		// what the channel still holds once they are acknowledged.
		payload  int
		buffered int
		refused  bool
	}{
		{
			// Nobody asked, so nobody is going to take it off the buffer.
			name:    "a whole answer nobody asked for",
			frame:   answer,
			channel: device.ControlChannel,
			payload: len(wire.EncodeEnvelope(wire.Envelope{
				Originator: wire.FromDevice, Service: 2, Body: []byte{0x80},
			})),
		},
		{
			// The framing has no marker to resynchronise on, so the start of
			// an envelope is kept for the rest of it.
			name:     "the start of one",
			frame:    partial,
			channel:  device.ControlChannel,
			payload:  20,
			buffered: 20,
		},
		{
			name:    "a notification on the events channel",
			frame:   events,
			channel: device.EventsChannel,
			payload: len("noise"),
		},
		{
			// Nobody is waiting on it, so the trace is where it shows.
			name:    "a bus that will not take it",
			frame:   events,
			channel: device.EventsChannel,
			refused: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			d := answers(s.ctrl)

			out := device.TestSender(d.out)
			if tt.refused {
				out = &device.FailAfter{Sender: d.out, Err: errors.New("refused")}
			}

			b := device.ShortBudgets()
			b.Idle = 20 * time.Millisecond

			session := device.NewTestSessionWith(s.T(), out, d.in, b)
			session.OpenChannels()

			var trace bytes.Buffer
			session.Trace(&trace)

			d.tell(tt.frame)

			if tt.refused {
				s.Require().Eventually(func() bool {
					return session.Received(tt.channel) > 0
				}, time.Second, time.Millisecond)

				time.Sleep(10 * b.Idle)
				s.Require().NoError(session.Close())
				s.Require().Contains(trace.String(), "idle ack: writing to events: refused")

				return
			}

			s.Require().Eventually(func() bool {
				count, _ := acks(d, tt.channel)

				return count > 0
			}, time.Second, time.Millisecond, "acknowledged once it went quiet")

			// Several quiet periods later, still the one: the acknowledgement
			// settled what was owed.
			time.Sleep(10 * b.Idle)

			count, ack := acks(d, tt.channel)
			s.Require().Equal(1, count, "at most once a quiet period")
			s.Require().Equal(wire.AckBase+uint32(tt.payload), ack)
			s.Require().Equal(tt.buffered, session.Buffered(tt.channel))
		})
	}

	s.Run("a channel with an exchange in flight", func() {
		// Write chunks carry the acknowledgement in their header and nothing
		// goes between them. Bytes that arrive partway through a message are
		// owed, and the idle acknowledgement still keeps out.
		d := answers(s.ctrl, s.status(device.DataChannel, device.FirstTxn, 0))

		var once sync.Once

		d.onWrite = func() {
			once.Do(func() {
				d.tell(device.FrameFor(device.DataChannel, wire.MsgData, []byte("unasked")))
			})
		}

		b := device.ShortBudgets()
		b.Idle, b.Pace = time.Millisecond, 30*time.Millisecond

		session := device.NewTestSessionWith(s.T(), d.out, d.in, b)
		session.OpenChannels()

		s.Require().NoError(session.WritePreset(
			context.Background(), 0, 3, bytes.Repeat([]byte{0x2a}, 2000)))

		first, last := -1, -1

		var kinds []uint16

		for i, raw := range d.frames() {
			f, _, err := wire.DecodeFrame(raw)
			s.Require().NoError(err)

			kinds = append(kinds, f.Type)

			if device.ChannelOf(f) != device.DataChannel || f.Type != wire.MsgData {
				continue
			}

			if first < 0 {
				first = i
			}

			last = i
		}

		s.Require().Greater(last, first, "the message went out in more than one chunk")

		for i := first; i <= last; i++ {
			s.Require().NotEqual(wire.MsgAck, kinds[i], "frame %d came between two chunks", i)
		}
	})
}

func TestLoopPublicTestSuite(t *testing.T) {
	suite.Run(t, new(LoopPublicTestSuite))
}
