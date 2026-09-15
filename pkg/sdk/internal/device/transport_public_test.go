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
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/retr0h/tonestack/pkg/sdk/internal/device"
	"github.com/retr0h/tonestack/pkg/sdk/internal/wire"
)

// TransportPublicTestSuite covers reading and writing frames.
//
// Framing, sequence numbers and acknowledgements: none of it needs a device,
// and all of it is what breaks one when it is wrong.
type TransportPublicTestSuite struct {
	suite.Suite

	ctrl *gomock.Controller
}

func (s *TransportPublicTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
}

// TestDrain waits until the device genuinely has nothing left.
func (s *TransportPublicTestSuite) TestDrain() {
	noise := device.FrameFor("control", wire.MsgData, []byte("noise"))

	tests := []struct {
		name   string
		device func() *deviceDouble
		opened bool
		// somebody who stopped waiting, who is not owed a drained endpoint.
		cancelled bool
		// quiet is a drain that waited for three quiet reads, and atBudget
		// one that could only end at its budget.
		quiet    bool
		atBudget bool
		// how many reads the session took, when the loop ended.
		reads int
	}{
		{
			// Three quiet reads in a row is a device with nothing left.
			name:   "a device with nothing to say",
			device: func() *deviceDouble { return answers(s.ctrl) },
			quiet:  true,
		},
		{
			// A timeout is the ordinary case, and counts as a quiet read,
			// however soon the endpoint says so.
			name: "one whose reads time out",
			device: func() *deviceDouble {
				return readFails(s.ctrl, context.DeadlineExceeded)
			},
			quiet: true,
		},
		{
			// A read that fails outright is not a quiet device. Waiting on
			// would spend the whole budget against a bus that is gone.
			name:   "a bus that will not answer",
			device: func() *deviceDouble { return readFails(s.ctrl, errors.New("boom")) },
			reads:  1,
		},
		{
			name:   "one at the end of its input",
			device: func() *deviceDouble { return readFails(s.ctrl, io.EOF) },
			reads:  1,
		},
		{
			// A drain that saw traffic starts counting quiet reads again,
			// and what it consumed is not replayed into a later reply.
			name:   "one with something to say",
			device: func() *deviceDouble { return unasked(s.ctrl, noise) },
			opened: true,
		},
		{
			// The device sends empty transfers when it has nothing to say.
			// Counted as traffic, they held every drain for its whole budget.
			name: "one that sends nothing but empty transfers",
			device: func() *deviceDouble {
				d := answers(s.ctrl)
				d.noisy = device.FrameFor(device.ControlChannel, wire.MsgAck, nil)

				return d
			},
			opened: true,
			quiet:  true,
		},
		{
			// A stale backlog clears in about a hundred frames. A device that
			// never stops is not waited on past the budget.
			name: "one that never goes quiet",
			device: func() *deviceDouble {
				d := answers(s.ctrl)
				d.noisy = noise

				return d
			},
			opened:   true,
			atBudget: true,
		},
		{
			name:      "a session nobody is waiting on any more",
			device:    func() *deviceDouble { return answers(s.ctrl) },
			cancelled: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			d := tt.device()

			b := device.ShortBudgets()
			b.Drain = 300 * time.Millisecond

			if !tt.atBudget {
				b.Drain = time.Minute
			}

			session := device.NewTestSessionWith(s.T(), d.out, d.in, b)

			if tt.opened {
				session.OpenChannels()
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			if tt.cancelled {
				cancel()
			}

			start := session.Windows()
			started := time.Now()

			session.Drain(ctx)

			took := time.Since(started)

			if !tt.atBudget {
				s.Require().Less(took, b.Drain,
					"a drain ends when it has an answer, not at a budget it did not need")
			}

			if tt.quiet {
				s.Require().GreaterOrEqual(session.Windows()-start, uint64(3), "three quiet reads")
			}

			if tt.atBudget {
				s.Require().GreaterOrEqual(took, b.Drain)
			}

			// A device that never stops has said more by the time anybody
			// looks, so only a drain that ended on silence is checked.
			if !tt.atBudget {
				s.Require().Zero(d.pending(), "everything the device had was read")
				s.Require().Zero(session.Buffered(device.ControlChannel),
					"what it consumed is not replayed into a later reply")
			}

			if tt.reads > 0 {
				s.Require().Equal(tt.reads, d.readCount())
			}
		})
	}
}

// TestRoute hands what a read brought to the channel it belongs to.
func (s *TransportPublicTestSuite) TestRoute() {
	noise := []byte("noise")
	full := device.FrameFor(device.ControlChannel, wire.MsgData, noise)
	events := device.FrameFor(device.EventsChannel, wire.MsgData, noise)

	tests := []struct {
		name    string
		frame   []byte
		opened  bool
		channel string
		// received is what the channel counted, and buffered what it kept.
		received uint32
		buffered int
	}{
		{
			name:     "a frame on a channel somebody opened",
			frame:    full,
			opened:   true,
			channel:  device.ControlChannel,
			received: uint32(len(noise)),
			buffered: len(noise),
		},
		{
			// The device sends notifications unasked, and one on a channel
			// this session never opened is not anybody's business.
			name:    "one on a channel nobody opened",
			frame:   events,
			channel: device.EventsChannel,
		},
		{
			// Nothing reads the events channel. What arrives there is owed an
			// acknowledgement, and kept nowhere.
			name:     "one on the events channel",
			frame:    events,
			opened:   true,
			channel:  device.EventsChannel,
			received: uint32(len(noise)),
		},
		{
			// A frame cut short at the end of a transfer is the transfer
			// ending, not corruption.
			name:    "one cut short at the end of a transfer",
			frame:   full[:6],
			opened:  true,
			channel: device.ControlChannel,
		},
		{
			// Acknowledging an empty transfer burns a sequence number, so
			// one is never counted.
			name:    "one carrying no stream bytes",
			frame:   device.FrameFor(device.ControlChannel, wire.MsgAck, nil),
			opened:  true,
			channel: device.ControlChannel,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			d := answers(s.ctrl)
			session := device.NewTestSession(s.T(), d.out, d.in)

			if tt.opened {
				session.OpenChannels()
			}

			start := session.Transfers()
			d.tell(tt.frame)

			s.Require().Eventually(func() bool {
				return session.Transfers() > start
			}, time.Second, time.Millisecond, "the loop read what the device said")

			s.Require().Equal(tt.received, session.Received(tt.channel))
			s.Require().Equal(tt.buffered, session.Buffered(tt.channel))
		})
	}
}

// TestTheWireTrace is how both directions were read off a device in the
// first place, and the thing that found the tag a write goes out under.
func (s *TransportPublicTestSuite) TestTheWireTrace() {
	d := answers(s.ctrl, device.FrameFor("control", wire.MsgData, []byte("noise")))

	var trace bytes.Buffer

	session := device.NewTestSession(s.T(), d.out, d.in)
	session.Trace(&trace)
	session.OpenChannels()
	session.Drain(context.Background())

	// Both directions: what was asked as well as what came back.
	_, _ = session.Call(context.Background(), device.ControlChannel, 1, nil)

	// Read once the loop has stopped writing to it.
	s.Require().NoError(session.Close())

	s.Require().Contains(trace.String(), "IN ")
	s.Require().Contains(trace.String(), "OUT control")
}

func TestTransportTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(TransportPublicTestSuite))
}
