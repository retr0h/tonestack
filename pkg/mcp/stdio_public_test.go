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
package mcp_test

import (
	"bytes"
	"context"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/mcp"
	"github.com/retr0h/tonestack/pkg/sdk"
)

type StdioPublicTestSuite struct {
	suite.Suite
}

// initialize is the first thing any client sends.
const initialize = `{"jsonrpc":"2.0","id":1,"method":"initialize",` +
	`"params":{"protocolVersion":"2025-06-18","capabilities":{},` +
	`"clientInfo":{"name":"test","version":"0"}}}` + "\n"

// slow stands in for stdout and takes its time with each reply, so a client
// that hangs up at once does so while the reply is still being written.
type slow struct {
	mu  sync.Mutex
	out bytes.Buffer
}

// Write waits, then keeps what was written.
func (w *slow) Write(
	p []byte,
) (int, error) {
	time.Sleep(50 * time.Millisecond)

	w.mu.Lock()
	defer w.mu.Unlock()

	return w.out.Write(p)
}

// TestRunOver covers how a session over a client's streams ends.
func (s *StdioPublicTestSuite) TestRunOver() {
	tests := []struct {
		name   string
		in     func() io.ReadCloser
		out    io.Writer
		cancel bool
		err    error
	}{
		{
			// A client that asks one thing and closes its end straight away,
			// the way piping a request into `tonestack mcp start` does. The
			// client is gone, so the session is over rather than broken.
			name: "a client that hangs up while its reply is written",
			in:   func() io.ReadCloser { return io.NopCloser(strings.NewReader(initialize)) },
			out:  &slow{},
		},
		{
			// Ctrl-C with a client still connected is a stop, reported as the
			// context ending so the command can tell it from a failure.
			name: "a session stopped by its context",
			in: func() io.ReadCloser {
				r, _ := io.Pipe()

				return r
			},
			out:    io.Discard,
			cancel: true,
			err:    context.Canceled,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			if tt.cancel {
				cancel()
			}

			done := make(chan error, 1)
			go func() {
				done <- mcp.New(sdk.New(), mcp.Options{}).RunOver(ctx, tt.in(), tt.out)
			}()

			select {
			case err := <-done:
				if tt.err == nil {
					s.Require().NoError(err)

					return
				}

				s.Require().ErrorIs(err, tt.err)
			case <-time.After(5 * time.Second):
				s.Fail("the session did not end")
			}
		})
	}
}

func TestStdioPublicTestSuite(t *testing.T) {
	suite.Run(t, new(StdioPublicTestSuite))
}
