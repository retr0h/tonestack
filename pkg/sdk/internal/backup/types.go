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

package backup

import (
	"context"

	"github.com/retr0h/tonestack/pkg/sdk/preset"
	"github.com/retr0h/tonestack/pkg/sdk/slot"
)

// Held is what one slot held before a write, as the device answered it.
type Held struct {
	// At is where the slot is.
	At slot.Address
	// Name is what the device calls the slot. Empty when nothing said.
	Name string
	// Body is what the device answered, nil for nothing.
	Body []byte
}

// Decoder reads a device's answer for one slot as a preset.
//
// Declared here so that this package reaches neither the device nor its wire:
// the flows that talk to a device hand one in.
type Decoder interface {
	// Document returns the preset body describes, or nil when the slot holds
	// no blocks.
	Document(
		ctx context.Context,
		body []byte,
		at slot.Address,
		name string,
	) (*preset.Document, error)
}
