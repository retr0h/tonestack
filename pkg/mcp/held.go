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
package mcp

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/retr0h/tonestack/pkg/mcp/internal/tools"
)

// Held reports whether the server is holding the pedal: a Session is open,
// or being opened, and has not been closed.
//
// The server opens one on the first device call and lets it go once the agent
// has been quiet for a while, when the bus fails, or when the server stops. A
// program that stops the server while this is true waits on the pedal being
// let go, and can say so.
func (s *Server) Held() bool {
	return s.held.open.Load() > 0
}

// holding is the Client the tools call, counting the Sessions it has open.
//
// A Session being opened counts as soon as the claim starts, because stopping
// the server then waits on the claim finishing and the Session closing.
type holding struct {
	tools.Client
	open atomic.Int64
}

// Open claims the pedal, and counts the Session until it is closed.
func (h *holding) Open(
	ctx context.Context,
) (tools.Session, error) {
	h.open.Add(1)

	s, err := h.Client.Open(ctx)
	if err != nil {
		h.open.Add(-1)

		return nil, err
	}

	return &heldSession{Session: s, holder: h}, nil
}

// heldSession is a Session that stops counting once it is closed.
type heldSession struct {
	tools.Session
	holder *holding
	once   sync.Once
}

// Close lets the pedal go, and stops counting the Session however that ends.
func (s *heldSession) Close() error {
	defer s.once.Do(func() { s.holder.open.Add(-1) })

	return s.Session.Close()
}
