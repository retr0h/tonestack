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

package device

import (
	"context"
	"io"
)

// usbOpener is the Opener that reaches hardware.
type usbOpener struct {
	// trace receives every frame in and out. Nil traces nothing.
	trace io.Writer
	// buses is where a bus comes from.
	buses buses
	// budgets are how long each session it opens waits on the device.
	budgets budgets
}

// NewUSB returns the Opener that reaches hardware over USB.
//
// Nothing is opened until a method is called. trace receives every frame a
// session sends and reads, which is how both directions were read off a device
// in the first place; nil traces nothing.
func NewUSB(
	trace io.Writer,
) Opener {
	return usbOpener{trace: trace, buses: usbBuses{}, budgets: defaultBudgets()}
}

// List reports every device on the bus.
func (usbOpener) List(
	ctx context.Context,
) ([]Descriptor, error) {
	l := NewUSBLister()
	// The listing is already made; a lister that will not close takes nothing from it.
	defer func() { _ = l.Close() }()

	return l.List(ctx)
}

// Open starts a session with the first attached device.
//
// HX Edit must be quit first: it claims the editor interface exclusively.
//
// Returns the interface rather than the type behind it, so that everything
// above this package can be given a session instead of finding one, which is
// what lets reading a device be tested without one attached.
func (u usbOpener) Open(
	ctx context.Context,
) (Editor, error) {
	return open(ctx, u.buses.Bus(), u.trace, u.budgets)
}
