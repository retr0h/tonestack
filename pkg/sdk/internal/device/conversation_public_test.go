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
	"go.uber.org/mock/gomock"

	"github.com/retr0h/tonestack/pkg/sdk/internal/device"
	"github.com/retr0h/tonestack/pkg/sdk/internal/wire"
)

// ConversationPublicTestSuite covers a session's beginning and end.
type ConversationPublicTestSuite struct {
	suite.Suite

	ctrl *gomock.Controller
}

func (s *ConversationPublicTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
}

// panicking is a wire trace that panics on the first frame read, standing in
// for a bug in routing.
//
// Written by hand because io.Writer is the standard library's interface.
type panicking struct{}

func (panicking) Write(
	p []byte,
) (int, error) {
	if bytes.HasPrefix(p, []byte("IN ")) {
		panic("routing went wrong")
	}

	return len(p), nil
}

// TestClose gives back what the session took, tells the device it is over,
// and stops the loop.
func (s *ConversationPublicTestSuite) TestClose() {
	refused := errors.New("refused")
	broken := errors.New("the bus went away")
	noise := device.FrameFor("control", wire.MsgData, []byte("noise"))

	tests := []struct {
		name string
		// device is what the session reads; nil is a session that never
		// claimed one.
		device func() *deviceDouble
		// refuse puts a sender refusing every write in front of the device.
		refuse bool
		// budgets changes the test budgets.
		budgets func(b *device.Budgets)
		opened  bool
		traced  bool
		// before runs against the session ahead of Close.
		before func(session *device.Session, d *deviceDouble)
		// released is what the last hold says when it is given back.
		released error
		// closes is how many channel closings Close sends, and silent that it
		// sends nothing at all.
		closes int
		silent bool
		trace  []string
		// fast is a Close that must end at its own budget.
		fast bool
		err  error
		says string
	}{
		{
			// Close has nobody to report to, so what it swallows shows only
			// in the trace. Everything is still given back.
			name:     "a close the bus and a hold both refuse, with the trace on",
			device:   func() *deviceDouble { return answers(s.ctrl) },
			refuse:   true,
			opened:   true,
			traced:   true,
			released: refused,
			trace: []string{
				"ack on close: writing to control: refused",
				"hello on close: writing to data: refused",
				"release 1 on close: refused",
			},
		},
		{
			// Two drains, each bounded on its own, would hold Close for twice
			// the drain budget. The close budget bounds the whole, and the
			// device is still told the session is over.
			name: "a session whose device never goes quiet",
			device: func() *deviceDouble {
				d := answers(s.ctrl)
				d.noisy = noise

				return d
			},
			budgets: func(b *device.Budgets) {
				b.Drain, b.Close = 2*time.Second, 50*time.Millisecond
			},
			opened: true,
			closes: len(device.ChannelNames()),
			fast:   true,
		},
		{
			// What the device sent is drained and acknowledged first:
			// dropping the interface with bytes unacknowledged carries a
			// debt into later sessions, until an otherwise innocent write
			// stops the device.
			//
			// And the message that opens a channel closes one. Without it
			// the device goes on believing an editor is attached, and its
			// front panel stops refreshing footswitches as somebody browses
			// presets on the pedal itself.
			name:   "a session that opened its channels",
			device: func() *deviceDouble { return unasked(s.ctrl, noise) },
			opened: true,
			closes: len(device.ChannelNames()),
		},
		{
			// The caller stopped waiting, and the loop carried on reading.
			// Close is what stops it.
			name:   "a session whose last call was given up on",
			device: func() *deviceDouble { return answers(s.ctrl) },
			opened: true,
			before: func(session *device.Session, _ *deviceDouble) {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
				defer cancel()

				_, err := session.Call(ctx, device.ControlChannel, 1, nil)
				s.Require().ErrorIs(err, context.DeadlineExceeded)
			},
			closes: len(device.ChannelNames()),
		},
		{
			// No read is posted to catch the device's answers, so nothing
			// more is said to it. The interface is still given back, and the
			// error that ended the loop is what Close says.
			name:   "a session the bus ended",
			device: func() *deviceDouble { return readFails(s.ctrl, broken) },
			opened: true,
			before: func(session *device.Session, _ *deviceDouble) {
				<-session.Dead()
			},
			silent: true,
			err:    broken,
		},
		{
			// A panic on a goroutine nobody joins would kill the process
			// without running anybody's deferred Close, and leave the pedal
			// needing a power cycle. It ends the session instead.
			name:   "a session whose loop panicked",
			device: func() *deviceDouble { return answers(s.ctrl) },
			opened: true,
			before: func(session *device.Session, d *deviceDouble) {
				session.Trace(panicking{})
				d.tell(noise)
				<-session.Dead()
			},
			silent: true,
			says:   "the read loop panicked: routing went wrong",
		},
		{
			// Nothing was taken, so there is nothing to give back.
			name: "one that never opened any",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			b := device.ShortBudgets()
			if tt.budgets != nil {
				tt.budgets(&b)
			}

			session := device.NewTestSessionWith(s.T(), nil, nil, b)

			var d *deviceDouble

			if tt.device != nil {
				d = tt.device()

				out := device.TestSender(d.out)
				if tt.refuse {
					out = &device.FailAfter{Sender: d.out, Err: refused}
				}

				session = device.NewTestSessionWith(s.T(), out, d.in, b)
			}

			if tt.opened {
				session.OpenChannels()
			}

			var trace bytes.Buffer
			if tt.traced {
				session.Trace(&trace)
			}

			var given []string

			session.OnDone(func() { given = append(given, "interface") })
			session.Holding(
				func() error { given = append(given, "device"); return nil },
				func() error { given = append(given, "library"); return tt.released },
			)

			if tt.before != nil {
				tt.before(session, d)
			}

			before := 0
			if d != nil {
				before = len(d.frames())
			}

			started := time.Now()
			err := session.Close()

			if tt.fast {
				s.Require().Less(time.Since(started), time.Second,
					"Close ends at its budget, not after two drains")
			}

			// In the order they were taken, and once however often Close is
			// called.
			s.Require().Equal([]string{"interface", "device", "library"}, given)
			s.Require().Equal(err, session.Close(), "Close is idempotent")
			s.Require().Len(given, 3, "nothing is given back twice")

			if d != nil {
				select {
				case <-session.LoopDone():
				default:
					s.Fail("the loop is still reading after Close")
				}
			}

			for _, want := range tt.trace {
				s.Require().Contains(trace.String(), want)
			}

			switch {
			case tt.err != nil || tt.says != "":
				s.Require().ErrorIs(err, device.ErrBus)
				s.Require().ErrorContains(err, tt.says)

				if tt.err != nil {
					s.Require().ErrorIs(err, tt.err)
				}
			default:
				s.Require().NoError(err)
			}

			if tt.silent {
				s.Require().Len(d.frames(), before,
					"nothing is said to a device nobody is reading")
			}

			if tt.closes == 0 {
				return
			}

			sent := d.frames()
			s.Require().Greater(len(sent), before, "the acknowledgement settles the debt")

			closes := 0

			for _, frame := range sent[before:] {
				kind, err := device.MessageKind(frame)
				s.Require().NoError(err)

				if kind == wire.MsgHello {
					closes++
				}
			}

			s.Require().Equal(tt.closes, closes,
				"every channel is told the session is over")
		})
	}
}

// TestRetry waits on an interface a previous session has not let go of.
func (s *ConversationPublicTestSuite) TestRetry() {
	tests := []struct {
		name  string
		until int
		tries int
		err   bool
	}{
		{
			// Cleanup after a previous session races the next claim, so an
			// interface that is busy is worth waiting on rather than
			// reporting.
			name:  "one that comes free",
			until: 2,
			tries: 2,
		},
		{
			name:  "one that never does",
			until: 0,
			tries: device.ClaimAttempts,
			err:   true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tries := 0

			err := device.Retry(func() error {
				tries++
				if tt.until == 0 || tries < tt.until {
					return errors.New("busy")
				}

				return nil
			})

			if tt.err {
				s.Require().Error(err, "patience runs out")
			} else {
				s.Require().NoError(err)
			}

			s.Require().Equal(tt.tries, tries)
		})
	}
}

// TestModel is which device answered.
func (s *ConversationPublicTestSuite) TestModel() {
	s.Require().Equal("HX Stomp", device.NewTestSession(s.T(), nil, nil).Model().Name)
}

func TestConversationTestSuite(t *testing.T) {
	suite.Run(t, new(ConversationPublicTestSuite))
}
