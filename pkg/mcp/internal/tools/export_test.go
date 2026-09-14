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
	"io"
	"time"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Pedal holds a Session across calls, exported so a test can run a call on it
// without a server: a handler that panics takes the server's process with it.
type Pedal = pedal

// NewPedal holds nothing until the first call, and never lets go for idling.
func NewPedal(
	c Client,
) *Pedal {
	return newPedal(c, time.Hour)
}

// OnPedal runs call on the Session p holds.
func OnPedal(
	ctx context.Context,
	p *Pedal,
	call func(Session) (int, error),
) (int, error) {
	return onPedal(ctx, p, call)
}

// RegisterIdle is Register with the pedal let go after idle rather than ten
// seconds, so a test can watch it happen.
func RegisterIdle(
	s *gomcp.Server,
	c Client,
	allowWrites bool,
	idle time.Duration,
) io.Closer {
	return register(s, c, allowWrites, idle)
}
