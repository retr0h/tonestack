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
package sdk

import (
	"errors"

	"github.com/retr0h/tonestack/pkg/sdk/internal/catalogview"
	"github.com/retr0h/tonestack/pkg/sdk/internal/device"
	"github.com/retr0h/tonestack/pkg/sdk/internal/recipes"
)

// Errors a caller matches with errors.Is.
//
// Each says what went wrong and nothing about what to do next. A terminal
// runs a command and an agent calls a tool, so the wrapper that knows which
// it is adds the next step.
var (
	// ErrNoSuchBlock reports a model the catalog does not carry.
	ErrNoSuchBlock = catalogview.ErrNotFound
	// ErrNoSuchRecipe reports a rig nobody has written.
	ErrNoSuchRecipe = recipes.ErrNotFound

	// ErrSelectNeedsDevice reports a Select whose Read named a file. A slot
	// is only ever selected on the device that plays it; naming a file and
	// quietly going to the pedal anyway would be wrong.
	ErrSelectNeedsDevice = errors.New("a slot is selected on a device, not in a file")
	// ErrEditSetlist reports a Copy or Swap whose Where.Setlist was set.
	// Edit already has FromSetlist and ToSetlist for that, one per side of
	// the move; Where.Setlist has no side to belong to and is never read.
	ErrEditSetlist = errors.New("name the setlists with FromSetlist and ToSetlist")

	// ErrClosed reports a Session method called after Close.
	ErrClosed = errors.New("the session is closed")
	// ErrBus reports a Session the bus ended: a read or a write the bus
	// refused, or the Session's read loop stopping on its own. That Session
	// is finished and nothing reconnects it; a caller who wants the pedal
	// again closes it and opens another.
	ErrBus = device.ErrBus
)
