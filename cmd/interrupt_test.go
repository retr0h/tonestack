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
package cmd

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/retr0h/tonestack/pkg/cli"
	"github.com/retr0h/tonestack/pkg/cli/mocks"
	"github.com/retr0h/tonestack/pkg/mcp"
	"github.com/retr0h/tonestack/pkg/sdk"
)

// InterruptTestSuite covers what the first Ctrl-C says, by what holds the
// pedal at that moment.
type InterruptTestSuite struct {
	suite.Suite

	ctrl *gomock.Controller
}

func (s *InterruptTestSuite) SetupSubTest() {
	s.ctrl = gomock.NewController(s.T())
}

// TestHeld covers holding as cli.Interrupts reads it at the first signal.
func (s *InterruptTestSuite) TestHeld() {
	server := func(held bool) func(*gomock.Controller) cli.Holder {
		return func(ctrl *gomock.Controller) cli.Holder {
			h := mocks.NewMockHolder(ctrl)
			h.EXPECT().Held().Return(held).AnyTimes()

			return h
		}
	}

	tests := []struct {
		name  string
		claim bool
		// follow is what holds the pedal for the command, if anything does.
		follow func(*gomock.Controller) cli.Holder
		says   bool
	}{
		{name: "a command that went to the device", claim: true, says: true},
		{name: "a command that did not", says: false},
		{name: "a server holding the pedal", follow: server(true), says: true},
		{name: "a server between holds", follow: server(false), says: false},
		{
			// What mcp start follows before any device tool has been called.
			name: "mcp start's own server, before it reaches the pedal",
			follow: func(*gomock.Controller) cli.Holder {
				return mcp.New(sdk.New(), mcp.Options{})
			},
			says: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var h holding

			if tt.claim {
				h.claim()
			}

			if tt.follow != nil {
				h.follow(tt.follow(s.ctrl))
			}

			process := mocks.NewMockProcess(s.ctrl)
			process.EXPECT().Stop()

			signals := make(chan os.Signal, 1)
			signals <- os.Interrupt
			close(signals)

			var out bytes.Buffer

			cli.Interrupts(signals, &out, &h, process)

			if tt.says {
				s.Contains(out.String(), "finishing with the pedal and letting it go")
			} else {
				s.Empty(out.String())
			}
		})
	}
}

func TestInterruptTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(InterruptTestSuite))
}
