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

	"github.com/retr0h/tonestack/pkg/cli"
)

// pedal is whether this run has gone to the device.
//
// Set by a command as it takes the device path, and read from the goroutine
// answering interrupts, so it is atomic.
var pedal holding

// holding is a cli.Holder a command sets once it goes to the device, or
// points at whatever holds the device on the command's behalf.
type holding struct {
	held atomic.Bool
	// by is what holds the device for the command, if something does.
	by atomic.Pointer[heldBy]
}

// heldBy is a cli.Holder behind a pointer, so it can be stored atomically.
type heldBy struct {
	cli.Holder
}

// claim records that the command is going to the device.
func (h *holding) claim() { h.held.Store(true) }

// follow answers Held from other as well, for a command such as mcp start
// whose server decides call by call whether it holds the device.
func (h *holding) follow(
	other cli.Holder,
) {
	h.by.Store(&heldBy{Holder: other})
}

// Held reports whether claim was called, or whether what the command follows
// holds the device now.
func (h *holding) Held() bool {
	if h.held.Load() {
		return true
	}

	by := h.by.Load()

	return by != nil && by.Held()
}

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
