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
	"errors"
	"io"
	"sync"
	"sync/atomic"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// run serves over a client's streams until ctx ends or the client hangs up.
//
// go-sdk v1.7.0 reports a client closing its input as a failure when the close
// lands while a reply is still being written: "server is closing: EOF". The
// client is gone either way, so a session whose input has ended is over rather
// than broken. The library keeps that error unexported, so this watches the
// input for its end instead of matching the error's text.
func (s *Server) run(
	ctx context.Context,
	r io.ReadCloser,
	w io.Writer,
) error {
	in := &hangup{ReadCloser: r}
	out := &output{Writer: w}

	err := s.Serve(ctx, &gomcp.IOTransport{Reader: in, Writer: out})

	// A context that has ended is why the session stopped, even when the input
	// ran out in the same moment and the library reported that instead. Which
	// one it saw first is a race, and the command tells a stop from a failure
	// by the context's error.
	if ctxErr := ctx.Err(); ctxErr != nil {
		return errors.Join(ctxErr, err)
	}

	// A reply that could not be written is why, too. The library closes the
	// input when a write fails, and whether it then reports the write or the
	// read that closing ended is another race.
	if werr := out.failed(); werr != nil {
		return errors.Join(werr, err)
	}

	if in.ended.Load() {
		return nil
	}

	return err
}

// hangup is what the client sends, and remembers whether it has ended.
type hangup struct {
	io.ReadCloser
	ended atomic.Bool
}

// Read reads what the client sent and notes when it has nothing more.
func (h *hangup) Read(
	p []byte,
) (int, error) {
	n, err := h.ReadCloser.Read(p)
	if errors.Is(err, io.EOF) {
		h.ended.Store(true)
	}

	return n, err
}

// output is where replies go. It remembers the first write that failed, and
// leaves stdout open when a session closes, since the process may still print
// to it on the way out.
type output struct {
	io.Writer
	mu  sync.Mutex
	err error
}

// Write writes a reply and notes the first failure.
func (o *output) Write(
	p []byte,
) (int, error) {
	n, err := o.Writer.Write(p)
	if err != nil {
		o.mu.Lock()
		if o.err == nil {
			o.err = err
		}
		o.mu.Unlock()
	}

	return n, err
}

// Close does nothing.
func (*output) Close() error { return nil }

// failed is the first write that failed, if one did.
func (o *output) failed() error {
	o.mu.Lock()
	defer o.mu.Unlock()

	return o.err
}
