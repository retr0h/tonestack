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

package tools

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

// PedalTestSuite covers giving up on the pedal, which the tools cannot reach
// on cue: a call's context ends inside the MCP library.
type PedalTestSuite struct {
	suite.Suite
}

// waiting is a context whose caller has already stopped waiting, and a pedal
// somebody else is holding.
func waiting() (context.Context, *pedal) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	p := newPedal(nil, time.Hour)
	p.lock <- struct{}{}

	return ctx, p
}

// TestTake covers waiting for the pedal.
func (s *PedalTestSuite) TestTake() {
	tests := []struct {
		name   string
		held   bool
		closed bool
	}{
		{name: "a free pedal"},
		{
			// An agent that gives up on a call should not stay queued behind
			// another one that holds the pedal.
			name: "a held one, and a call that gave up",
			held: true,
		},
		{
			// A call that arrives after the server let go would otherwise
			// claim the pedal again.
			name:   "one the server has let go",
			closed: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			if tt.held {
				ctx, p := waiting()
				s.Require().ErrorIs(p.take(ctx), context.Canceled)

				return
			}

			if tt.closed {
				p := newPedal(nil, time.Hour)
				s.Require().NoError(p.Close())
				s.Require().ErrorIs(p.take(context.Background()), errStopped)
				s.Empty(p.lock, "the lock is given back")

				return
			}

			p := newPedal(nil, time.Hour)
			s.Require().NoError(p.take(context.Background()))
			p.give()
			s.Empty(p.lock)
		})
	}
}

// TestLocked covers a call that holds the pedal without a Session.
func (s *PedalTestSuite) TestLocked() {
	s.Run("a call that gave up while the pedal was held", func() {
		ctx, p := waiting()
		ran := false

		_, err := locked(ctx, p, func() (int, error) {
			ran = true

			return 0, nil
		})

		s.Require().ErrorIs(err, context.Canceled)
		s.False(ran, "the call never runs")
	})
}

// TestOnPedal covers a device call that gives up before it has the pedal.
func (s *PedalTestSuite) TestOnPedal() {
	s.Run("a call that gave up while the pedal was held", func() {
		ctx, p := waiting()
		ran := false

		_, err := onPedal(ctx, p, func(Session) (int, error) {
			ran = true

			return 0, nil
		})

		s.Require().ErrorIs(err, context.Canceled)
		s.False(ran, "the call never runs, and nothing is opened")
	})
}

func TestPedalTestSuite(t *testing.T) {
	suite.Run(t, new(PedalTestSuite))
}
