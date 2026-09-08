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
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk"
	"github.com/retr0h/tonestack/pkg/sdk/wire"
)

// ConversationTestSuite covers a session's beginning and end.
type ConversationTestSuite struct {
	suite.Suite
}

// TestClose gives back what the session took, and tells the device it is
// over.
func (s *ConversationTestSuite) TestClose() {
	tests := []struct {
		name   string
		opened bool
		order  []string
		closes int
		sent   bool
	}{
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
			opened: true,
			order:  []string{"interface", "device", "library"},
			closes: len(sdk.ChannelNames()),
			sent:   true,
		},
		{
			// Nothing was taken, so there is nothing to give back.
			name:  "one that never opened any",
			order: []string{"interface", "device", "library"},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			d := answers(sdk.FrameFor("control", wire.MsgData, []byte("noise")))

			var given []string

			session := sdk.NewTestSession(d, d)

			if !tt.opened {
				session = sdk.NewTestSession(nil, nil)
			} else {
				session.OpenChannels()
			}

			session.OnDone(func() { given = append(given, "interface") })
			session.Holding(
				func() error { given = append(given, "device"); return nil },
				func() error { given = append(given, "library"); return nil },
			)

			before := len(d.sent)

			session.Close()

			// In the order they were taken.
			s.Require().Equal(tt.order, given)

			if !tt.sent {
				return
			}

			s.Require().NotEmpty(d.sent, "the acknowledgement settles the debt")

			closes := 0

			for _, frame := range d.sent[before:] {
				kind, err := sdk.MessageKind(frame)
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
func (s *ConversationTestSuite) TestRetry() {
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
			tries: sdk.ClaimAttempts,
			err:   true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tries := 0

			err := sdk.Retry(func() error {
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
func (s *ConversationTestSuite) TestModel() {
	s.Require().Equal("HX Stomp", sdk.NewTestSession(nil, nil).Model().Name)
}

func TestConversationTestSuite(t *testing.T) {
	suite.Run(t, new(ConversationTestSuite))
}
