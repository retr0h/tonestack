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
// Package device reports what hardware is attached.
package attached

import (
	"context"

	"github.com/retr0h/tonestack/pkg/sdk"

	"github.com/retr0h/tonestack/pkg/sdk/device"
)

// newLister is how a bus is obtained, so a test can stand in for it.
//
// The one thing in this package that needs hardware; everything reached
// through it takes the lister as an argument instead.
var newLister = device.NewUSBLister

// List reports every recognised device on the bus.
func List(ctx context.Context) (sdk.Attached, error) {
	l := newLister()
	defer func() { _ = l.Close() }()

	return ListWith(ctx, l)
}
