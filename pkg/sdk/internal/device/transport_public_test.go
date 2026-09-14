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
	"context"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/internal/device"
	"github.com/retr0h/tonestack/pkg/sdk/internal/wire"
)

// TransportPublicTestSuite covers reading and writing frames.
//
// Framing, sequence numbers and acknowledgements: none of it needs a device,
// and all of it is what breaks one when it is wrong.
type TransportPublicTestSuite struct {
	suite.Suite
}

// TestDrain reads until the device genuinely has nothing left.
func (s *TransportPublicTestSuite) TestDrain() {
	tests := []struct {
		name   string
		device func() *scripted
		opened bool
		// somebody who stopped waiting, who is not owed a drained endpoint.
		cancelled bool
		// how many reads the drain took, when that is the point.
		reads int
	}{
		{
			// Three quiet reads in a row is a device with nothing left.
			name:   "a device with nothing to say",
			device: func() *scripted { return answers() },
			reads:  3,
		},
		{
			// A timeout is the ordinary case, and counts as a quiet read.
			name: "one whose reads time out",
			device: func() *scripted {
				return &scripted{readErr: context.DeadlineExceeded}
			},
			reads: 3,
		},
		{
			// A read that fails outright is not a quiet device. Reading on
			// would spin for the whole budget against a bus that is gone.
			name:   "a bus that will not answer",
			device: func() *scripted { return &scripted{readErr: errors.New("boom")} },
			reads:  1,
		},
		{
			name:   "one at the end of its input",
			device: func() *scripted { return &scripted{readErr: io.EOF} },
			reads:  1,
		},
		{
			// A drain that saw traffic starts counting quiet reads again,
			// and what it consumed is not replayed into a later reply.
			name: "one with something to say",
			device: func() *scripted {
				return answers(device.FrameFor("control", wire.MsgData, []byte("noise")))
			},
			opened: true,
		},
		{
			// A cancelled read returns instantly, so draining on regardless
			// would spin for the whole budget rather than stop.
			name:      "a session nobody is waiting on any more",
			device:    func() *scripted { return answers() },
			cancelled: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			d := tt.device()
			session := device.NewTestSession(d, d)

			if tt.opened {
				session.OpenChannels()
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			if tt.cancelled {
				cancel()
			}

			session.Drain(ctx)

			s.Require().Empty(d.replies, "everything the device had was read")

			if tt.reads > 0 {
				s.Require().Equal(tt.reads, d.reads)
			}
		})
	}
}

// TestReceive takes one transfer off the bus.
func (s *TransportPublicTestSuite) TestReceive() {
	full := device.FrameFor("control", wire.MsgData, []byte("noise"))
	broken := errors.New("the bus went away")

	tests := []struct {
		name    string
		frames  [][]byte
		readErr error
		opened  bool
		// somebody who stopped waiting before the read came back.
		cancelled bool
		want      bool
		err       error
		says      string
	}{
		{
			// A device is asked far more often than it answers.
			name:    "a read that timed out",
			readErr: context.DeadlineExceeded,
		},
		{
			// Not silence. Reporting it as silence turns a bus that has gone
			// into a call that waits out its budget and then says no reply.
			name:    "a read that failed outright",
			readErr: broken,
			err:     broken,
			says:    "reading from the device",
		},
		{
			// Whatever the read says, the caller stopped waiting, and that is
			// what they are told.
			name:      "a caller who stopped waiting",
			readErr:   context.DeadlineExceeded,
			cancelled: true,
			err:       context.Canceled,
		},
		{
			name:   "a frame on a channel somebody opened",
			frames: [][]byte{full},
			opened: true,
			want:   true,
		},
		{
			// The device sends notifications unasked, and one on a channel
			// this session never opened is not anybody's business.
			name:   "one on a channel nobody opened",
			frames: [][]byte{device.FrameFor("events", wire.MsgData, []byte("noise"))},
		},
		{
			// A frame cut short at the end of a transfer is the transfer
			// ending, not corruption.
			name:   "one cut short at the end of a transfer",
			frames: [][]byte{full[:6]},
			opened: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			d := answers(tt.frames...)
			d.readErr = tt.readErr

			session := device.NewTestSession(d, d)

			if tt.opened {
				session.OpenChannels()
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			if tt.cancelled {
				cancel()
			}

			got, err := session.Receive(ctx)

			s.Require().Equal(tt.want, got)

			if tt.err == nil {
				s.Require().NoError(err)

				return
			}

			s.Require().ErrorIs(err, tt.err)
			s.Require().ErrorContains(err, tt.says)
		})
	}
}

// TestTheWireTrace is how both directions were read off a device in the
// first place, and the thing that found the tag a write goes out under.
func (s *TransportPublicTestSuite) TestTheWireTrace() {
	defer device.SetDebug(true)()

	d := answers(device.FrameFor("control", wire.MsgData, []byte("noise")))

	session := device.NewTestSession(d, d)
	session.OpenChannels()
	session.Drain(context.Background())

	// Both directions: what was asked as well as what came back.
	_, _ = session.Call(context.Background(), device.ControlChannel, 1, nil)
}

func TestTransportTestSuite(t *testing.T) {
	suite.Run(t, new(TransportPublicTestSuite))
}
