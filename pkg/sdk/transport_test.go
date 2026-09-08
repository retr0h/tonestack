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
	"context"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk"
	"github.com/retr0h/tonestack/pkg/sdk/wire"
)

// TransportTestSuite covers reading and writing frames.
//
// Framing, sequence numbers and acknowledgements: none of it needs a device,
// and all of it is what breaks one when it is wrong.
type TransportTestSuite struct {
	suite.Suite
}

// TestDrain reads until the device genuinely has nothing left.
func (s *TransportTestSuite) TestDrain() {
	tests := []struct {
		name   string
		device func() *device
		opened bool
	}{
		{
			name:   "a device with nothing to say",
			device: func() *device { return answers() },
		},
		{
			// A read that fails is treated as the device having nothing to
			// say, because that is what a timeout looks like and a timeout
			// is the ordinary case. The cost is that a genuine bus failure
			// surfaces later, as a call with no reply, rather than here.
			name:   "a bus that will not answer",
			device: func() *device { return &device{readErr: errors.New("boom")} },
		},
		{
			name:   "one at the end of its input",
			device: func() *device { return &device{readErr: io.EOF} },
		},
		{
			// A drain that saw traffic starts counting quiet reads again,
			// and what it consumed is not replayed into a later reply.
			name: "one with something to say",
			device: func() *device {
				return answers(sdk.FrameFor("control", wire.MsgData, []byte("noise")))
			},
			opened: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			d := tt.device()
			session := sdk.NewTestSession(d, d)

			if tt.opened {
				session.OpenChannels()
			}

			session.Drain(context.Background())

			s.Require().Empty(d.replies, "everything the device had was read")
		})
	}
}

// TestReceive takes one transfer off the bus.
func (s *TransportTestSuite) TestReceive() {
	full := sdk.FrameFor("control", wire.MsgData, []byte("noise"))

	tests := []struct {
		name   string
		frames [][]byte
		opened bool
		want   bool
	}{
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
			frames: [][]byte{sdk.FrameFor("events", wire.MsgData, []byte("noise"))},
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
			session := sdk.NewTestSession(d, d)

			if tt.opened {
				session.OpenChannels()
			}

			s.Require().Equal(tt.want, session.Receive(context.Background()))
		})
	}
}

// TestTheWireTrace is how both directions were read off a device in the
// first place, and the thing that found the tag a write goes out under.
func (s *TransportTestSuite) TestTheWireTrace() {
	defer sdk.SetDebug(true)()

	d := answers(sdk.FrameFor("control", wire.MsgData, []byte("noise")))

	session := sdk.NewTestSession(d, d)
	session.OpenChannels()
	session.Drain(context.Background())

	// Both directions: what was asked as well as what came back.
	_, _ = session.Call(context.Background(), sdk.ControlChannel, 1, nil)
}

func TestTransportTestSuite(t *testing.T) {
	suite.Run(t, new(TransportTestSuite))
}
