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
	"context"
	"os"
	"sync/atomic"
)

// pedal is whether this run has gone to the device.
//
// Set by a command as it takes the device path, and read from the goroutine
// answering interrupts, so it is atomic.
var pedal holding

// holding is a cli.Holder a command sets once it goes to the device.
type holding struct {
	held atomic.Bool
}

// claim records that the command is going to the device.
func (h *holding) claim() { h.held.Store(true) }

// Held reports whether claim was called.
func (h *holding) Held() bool { return h.held.Load() }

// process is the cli.Process this program is.
type process struct {
	cancel context.CancelFunc
}

// Stop cancels the command's context.
func (p process) Stop() { p.cancel() }

// Exit ends the program with code.
func (process) Exit(
	code int,
) {
	os.Exit(code)
}
