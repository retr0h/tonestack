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
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk"
)

// SessionTestSuite covers the Client the server hands the tools.
type SessionTestSuite struct {
	suite.Suite
}

// TestOpened covers handing on what the SDK opened.
func (s *SessionTestSuite) TestOpened() {
	refused := errors.New("no device found")

	tests := []struct {
		name    string
		session *sdk.Session
		err     error
	}{
		{name: "a Session", session: &sdk.Session{}},
		{
			// A nil *sdk.Session inside a non-nil interface would read as a
			// Session to a holder checking for none.
			name: "a failure, which comes with no Session",
			err:  refused,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := opened(tt.session, tt.err)

			if tt.err != nil {
				s.Require().ErrorIs(err, tt.err)
				s.Require().Nil(got)

				return
			}

			s.Require().NoError(err)
			s.Require().Equal(tt.session, got)
		})
	}
}

// TestFromSDK covers opening the pedal through the SDK's own Client.
func (s *SessionTestSuite) TestFromSDK() {
	s.Run("a caller who stopped waiting", func() {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		// Nothing reaches USB: the SDK looks at ctx before the bus.
		got, err := FromSDK(sdk.New()).Open(ctx)

		s.Require().ErrorIs(err, context.Canceled)
		s.Require().Nil(got)
	})
}

func TestSessionTestSuite(t *testing.T) {
	suite.Run(t, new(SessionTestSuite))
}
