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

// clock is a pause the test lets go of. Every wait on it blocks until
// release, and entered signals once one has started.
type clock struct {
	gate    chan time.Time
	entered chan struct{}
	once    sync.Once
}

// newClock is a clock that holds every pause, released when the test ends if
// the test has not released it already.
func (s *LoopPublicTestSuite) newClock() *clock {
	c := &clock{gate: make(chan time.Time), entered: make(chan struct{}, 1)}
	s.T().Cleanup(c.release)

	return c
}

// after is the session's clock.
func (c *clock) after(
	time.Duration,
) <-chan time.Time {
	select {
	case c.entered <- struct{}{}:
	default:
	}

	return c.gate
}

// release lets every pause go, now and from now on.
func (c *clock) release() {
	c.once.Do(func() { close(c.gate) })
}

// looked waits until the acknowledger has been round several more times, so
// what it would have sent by now has been sent.
func (s *LoopPublicTestSuite) looked(
	session *device.Session,
) {
	from := session.IdlePasses()

	s.Require().Eventually(func() bool {
		return session.IdlePasses() > from+3
	}, 5*time.Second, time.Millisecond, "the acknowledger kept looking")
}

// unasked is a notification on the control channel.
func unaskedFrame() []byte {
	return device.FrameFor(device.ControlChannel, wire.MsgData, []byte("unasked"))
}

// TestLoop keeps a read posted from start to Close, and ends the session
// when the bus does.
func (s *LoopPublicTestSuite) TestLoop() {
	broken := errors.New("the bus went away")

	tests := []struct {
		name    string
		device  func() *deviceDouble
		budgets func(b *device.Budgets)
		// clocked holds the flash pause, pacing and the poll until the row
		// releases them.
		clocked bool
		run     func(session *device.Session, d *deviceDouble, clk *clock)
	}{
		{
			// Nothing is running, and the device is still read. Before the
			// loop, this sat on the endpoint until somebody asked something.
			name: "a notification between two calls",
			device: func() *deviceDouble {
				return answers(s.ctrl, s.status(device.ControlChannel, device.FirstTxn, 0))
			},
			run: func(session *device.Session, d *deviceDouble, _ *clock) {
				_, err := session.Call(context.Background(), device.ControlChannel, 1, nil)
				s.Require().NoError(err)

				before := session.Received(device.ControlChannel)
				d.tell(unaskedFrame())

				s.Require().Eventually(func() bool {
					return session.Received(device.ControlChannel) > before
				}, 5*time.Second, time.Millisecond, "read with no operation running")
			},
		},
		{
			// The flash pause waits on a clock, not by sleeping the only
			// reader.
			name: "a notification during the flash pause",
			device: func() *deviceDouble {
				return answers(s.ctrl, s.status(device.DataChannel, device.FirstTxn, 0))
			},
			clocked: true,
			run: func(session *device.Session, d *deviceDouble, clk *clock) {
				done := make(chan error, 1)

				go func() { done <- session.WritePreset(context.Background(), 0, 3, []byte{0x01}) }()

				// One chunk, so the first pause is the flash pause.
				<-clk.entered
				d.tell(unaskedFrame())

				s.Require().Eventually(func() bool {
					return session.Received(device.ControlChannel) > 0
				}, 5*time.Second, time.Millisecond, "read while the write settled")

				clk.release()
				s.Require().NoError(<-done)
			},
		},
		{
			name: "a notification while a switch is polled",
			device: func() *deviceDouble {
				return answers(s.ctrl,
					s.status(device.DataChannel, device.FirstTxn, 1),
					s.playing(device.FirstTxn+1, 5),
					s.playing(device.FirstTxn+2, 99),
				)
			},
			budgets: func(b *device.Budgets) { b.Selecting = time.Hour },
			clocked: true,
			run: func(session *device.Session, d *deviceDouble, clk *clock) {
				done := make(chan error, 1)

				go func() { done <- session.SelectPreset(context.Background(), 0, 99) }()

				// The device answered with the old preset, so it is asked again
				// after a poll.
				<-clk.entered
				d.tell(unaskedFrame())

				s.Require().Eventually(func() bool {
					return session.Received(device.ControlChannel) > 0
				}, 5*time.Second, time.Millisecond, "read between the questions")

				clk.release()
				s.Require().NoError(<-done)
			},
		},
		{
			// A timeout is the device with nothing to say, however often it
			// happens.
			name:   "reads that time out",
			device: func() *deviceDouble { return readFails(s.ctrl, context.DeadlineExceeded) },
			run: func(session *device.Session, _ *deviceDouble, _ *clock) {
				start := session.Windows()

				s.Require().Eventually(func() bool {
					return session.Windows() > start+5
				}, 5*time.Second, time.Millisecond)
				s.Require().NoError(session.Ended())
			},
		},
		{
			// A read can end on its timeout with bytes already in hand. They
			// are an answer, and the read was not a failure.
			name: "bytes that came back with a timeout",
			device: func() *deviceDouble {
				d := unasked(s.ctrl, unaskedFrame())
				d.partial = context.DeadlineExceeded

				return d
			},
			run: func(session *device.Session, _ *deviceDouble, _ *clock) {
				s.Require().Eventually(func() bool {
					return session.Received(device.ControlChannel) == uint32(len("unasked"))
				}, 5*time.Second, time.Millisecond, "the bytes were routed")

				start := session.Windows()

				s.Require().Eventually(func() bool {
					return session.Windows() > start+3
				}, 5*time.Second, time.Millisecond)
				s.Require().NoError(session.Ended())
			},
		},
		{
			// The waiter hears about it at once rather than at the end of its
			// budget, and nothing more is asked of a device nobody is
			// reading.
			name:    "a bus that fails while a call waits",
			device:  func() *deviceDouble { return readFailsAfter(s.ctrl, broken, 1) },
			budgets: func(b *device.Budgets) { b.Reply = time.Hour },
			run: func(session *device.Session, d *deviceDouble, _ *clock) {
				_, err := session.Call(context.Background(), device.ControlChannel, 1, nil)
				s.Require().ErrorIs(err, broken)
				s.Require().ErrorIs(err, device.ErrBus)
				s.Require().ErrorContains(err, "reading from the device")

				sent := len(d.frames())

				_, err = session.Call(context.Background(), device.ControlChannel, 1, nil)
				s.Require().ErrorIs(err, broken)
				s.Require().Len(d.frames(), sent, "nothing is asked of a bus that failed")
			},
		},
		{
			// No chunk goes out without the device's acknowledgement of the
			// one before it, so a read that fails ends the loop the ordinary
			// way and the write stops there rather than sending the rest
			// blind. Close still tells the device the editor has gone.
			name:   "a bus that fails while a message goes out",
			device: func() *deviceDouble { return readFailsAfter(s.ctrl, broken, 1) },
			run: func(session *device.Session, d *deviceDouble, _ *clock) {
				err := session.WritePreset(
					context.Background(),
					0,
					3,
					bytes.Repeat([]byte{0x2a}, 2000),
				)
				s.Require().ErrorIs(err, broken)
				s.Require().ErrorIs(err, device.ErrBus)

				before := len(d.frames())

				s.Require().ErrorIs(session.Close(), broken)

				hellos := 0

				for _, raw := range d.frames()[before:] {
					if kind, _ := device.MessageKind(raw); kind == wire.MsgHello {
						hellos++
					}
				}

				s.Require().Equal(len(device.ChannelNames()), hellos, "the farewell was attempted")
			},
		},
		{
			// Recovered, and ended the way a bus error ends it, so the caller
			// gets an error rather than a process that died.
			name: "a panic in routing while a call waits",
			device: func() *deviceDouble {
				return answers(s.ctrl, s.status(device.ControlChannel, device.FirstTxn, 0))
			},
			budgets: func(b *device.Budgets) { b.Reply = time.Hour },
			run: func(session *device.Session, _ *deviceDouble, _ *clock) {
				session.Trace(panicking{})

				_, err := session.Call(context.Background(), device.ControlChannel, 1, nil)
				s.Require().ErrorIs(err, device.ErrBus)
				s.Require().ErrorContains(err, "the read loop panicked")
			},
		},
		{
			// The acknowledger is a goroutine nobody joins too.
			name: "a panic in the acknowledger",
			device: func() *deviceDouble {
				d := unasked(s.ctrl, unaskedFrame())

				var once sync.Once

				d.onWrite = func() {
					fire := false
					once.Do(func() { fire = true })

					if fire {
						panic("a bug inside an acknowledgement")
					}
				}

				return d
			},
			budgets: func(b *device.Budgets) { b.Idle = time.Millisecond },
			run: func(session *device.Session, _ *deviceDouble, _ *clock) {
				ended(s.T(), session)

				s.Require().ErrorIs(session.Ended(), device.ErrBus)
				s.Require().ErrorContains(session.Ended(), "the acknowledger panicked")
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			d := tt.device()
			clk := s.newClock()

			b := device.ShortBudgets()
			if tt.budgets != nil {
				tt.budgets(&b)
			}

			if tt.clocked {
				b.After = clk.after
			}

			session := device.NewOpenTestSession(s.T(), d.out, d.in, b)
			tt.run(session, d, clk)
		})
	}
}

// TestPaceEndsWhenTheBusDiesWhileWaiting covers a chunk's pace already
// waiting, not just the check before it starts: a loop that ends while pace
// is blocked in its own select wakes it directly, on the same signal a bus
// failure discovered between two reads uses.
func (s *LoopPublicTestSuite) TestPaceEndsWhenTheBusDiesWhileWaiting() {
	broken := errors.New("the bus went away")

	d := answers(s.ctrl)
	// The chunk goes out and earns nothing: pace has only the gated budget
	// and the bus itself to wake it.
	d.stopAckingAfter(device.DataChannel, 0)

	clk := s.newClock()

	b := device.ShortBudgets()
	b.After = clk.after

	session := device.NewOpenTestSession(s.T(), d.out, d.in, b)

	done := make(chan error, 1)

	go func() {
		done <- session.WritePreset(
			context.Background(), 0, 3, bytes.Repeat([]byte{0x2a}, 2000))
	}()

	// Pace is now waiting on the gated budget for the first chunk's
	// acknowledgement, which never comes.
	<-clk.entered

	d.mu.Lock()
	d.readErr, d.failAfter = broken, 0
	d.mu.Unlock()
	d.signal()

	err := <-done
	s.Require().ErrorIs(err, broken)
	s.Require().ErrorIs(err, device.ErrBus)
}

// ackValues is what each acknowledgement a session sent on a channel carried,
// in the order they went out.
func ackValues(
	d *deviceDouble,
	channel string,
) []uint32 {
	var values []uint32

	for _, raw := range d.frames() {
		f, _, err := wire.DecodeFrame(raw)
		if err != nil || f.Type != wire.MsgAck || device.ChannelOf(f) != channel {
			continue
		}

		values = append(values, f.Ack)
	}

	return values
}

// acks counts the acknowledgements a session sent on a channel, and says what
// the last of them carried.
func acks(
	d *deviceDouble,
	channel string,
) (int, uint32) {
	values := ackValues(d, channel)
	if len(values) == 0 {
		return 0, 0
	}

	return len(values), values[len(values)-1]
}

// allAcks counts the acknowledgements a session sent on every channel.
func allAcks(
	d *deviceDouble,
) int {
	total := 0

	for _, name := range device.ChannelNames() {
		count, _ := acks(d, name)
		total += count
	}

	return total
}

// TestIdleAck settles what arrives on a channel nobody is using, and never
// inside a write.
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
			b.Idle = time.Millisecond

			session := device.NewOpenTestSession(s.T(), out, d.in, b)
			var trace bytes.Buffer
			session.Trace(&trace)

			d.tell(tt.frame)

			s.Require().Eventually(func() bool {
				return session.Received(tt.channel) > 0
			}, 5*time.Second, time.Millisecond)

			if tt.refused {
				s.looked(session)
				s.Require().NoError(session.Close())
				s.Require().Contains(trace.String(), "idle ack: writing to events: refused")

				return
			}

			s.Require().Eventually(func() bool {
				count, _ := acks(d, tt.channel)

				return count > 0
			}, 5*time.Second, time.Millisecond, "acknowledged once it went quiet")

			// Several rounds later, still the one: the acknowledgement settled
			// what was owed.
			s.looked(session)

			count, ack := acks(d, tt.channel)
			s.Require().Equal(1, count, "at most once a quiet period")
			s.Require().Equal(wire.AckBase+uint32(tt.payload), ack)
			s.Require().Equal(tt.buffered, session.Buffered(tt.channel))
		})
	}

	s.Run("channels being opened", func() {
		// Opening control takes seven frames, so the eighth is the events
		// channel's opening. That one draws a notification on control, which
		// is open, owed and quiet while events and data are still opening.
		replies := make([][]byte, 0, 8)
		for range 7 {
			replies = append(replies, device.FrameFor(device.ControlChannel, wire.MsgAck, nil))
		}

		replies = append(replies, unaskedFrame())

		d := answers(s.ctrl, replies...)

		b := device.ShortBudgets()
		b.Idle, b.Open = time.Millisecond, 100*time.Millisecond

		session := device.NewTestSessionWith(s.T(), d.out, d.in, b)
		s.Require().NoError(session.Handshake(context.Background()))

		// The opening's own acknowledgements are control's two. Anything more
		// before data's opening is done went out inside the handshake.
		control := 0

		for _, raw := range d.frames() {
			f, _, err := wire.DecodeFrame(raw)
			s.Require().NoError(err)

			if f.Type != wire.MsgAck {
				continue
			}

			if device.ChannelOf(f) == device.DataChannel {
				break
			}

			if device.ChannelOf(f) == device.ControlChannel {
				control++
			}
		}

		s.Require().Equal(2, control, "nothing is acknowledged while channels open")
	})

	s.Run("a write going out, on any channel", func() {
		// A notification arrives on the events channel as a write starts.
		// Before the gate was session-wide, an acknowledgement for it went
		// out between two data chunks.
		clk := s.newClock()
		d := answers(s.ctrl, s.status(device.DataChannel, device.FirstTxn, 0))

		var once sync.Once

		d.onWrite = func() {
			once.Do(func() {
				d.tell(device.FrameFor(device.EventsChannel, wire.MsgData, []byte("unasked")))
			})
		}

		b := device.ShortBudgets()
		b.Idle, b.After = time.Millisecond, clk.after

		session := device.NewOpenTestSession(s.T(), d.out, d.in, b)
		done := make(chan error, 1)

		go func() {
			done <- session.WritePreset(
				context.Background(), 0, 3, bytes.Repeat([]byte{0x2a}, 2000))
		}()

		s.Require().Eventually(func() bool {
			return session.Received(device.EventsChannel) > 0
		}, 5*time.Second, time.Millisecond)

		// The clock holds a pause between two chunks, so the message is still
		// going out while the acknowledger looks.
		s.looked(session)
		s.Require().Zero(allAcks(d), "nothing is acknowledged while a message goes out")

		clk.release()
		s.Require().NoError(<-done)
	})

	s.Run("the flash pause after a write", func() {
		// The write's answer is owed an acknowledgement, and it waits until
		// the flash pause is over.
		//
		// The answer is routed at the one moment that used to break that:
		// after the write looked through the buffer and found nothing, and
		// before it decided whether anything was owed. Deciding on a later
		// look than the buffer's acknowledged the whole answer inside the
		// write, which the scheduler did on its own three times in five
		// thousand loaded runs.
		clk := s.newClock()
		d := answers(s.ctrl)

		b := device.ShortBudgets()
		b.Idle, b.After = time.Millisecond, clk.after

		session := device.NewOpenTestSession(s.T(), d.out, d.in, b)
		status := s.status(device.DataChannel, device.FirstTxn, 0)

		var once sync.Once

		// On the write's goroutine, so it asserts rather than requires.
		session.OnUnanswered(func() {
			once.Do(func() {
				d.tell(status)

				s.Eventually(func() bool {
					return session.Received(device.DataChannel) > 0
				}, 5*time.Second, time.Millisecond, "the answer was routed")
			})
		})

		done := make(chan error, 1)

		go func() { done <- session.WritePreset(context.Background(), 0, 3, []byte{0x01}) }()

		// One chunk, so the first pause is the flash pause.
		<-clk.entered
		s.looked(session)
		s.Require().Zero(allAcks(d), "nothing is acknowledged while flash settles")

		clk.release()
		s.Require().NoError(<-done)

		s.Require().Eventually(func() bool {
			count, _ := acks(d, device.DataChannel)

			return count > 0
		}, 5*time.Second, time.Millisecond, "acknowledged once the pause was over")
	})
}

func TestLoopPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(LoopPublicTestSuite))
}
