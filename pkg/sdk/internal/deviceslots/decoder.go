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

package deviceslots

import (
	"context"

	"github.com/retr0h/tonestack/pkg/sdk/preset"
	"github.com/retr0h/tonestack/pkg/sdk/result"
	"github.com/retr0h/tonestack/pkg/sdk/slot"
)

// Decoder reads a device's answer for one slot as the preset a backup keeps.
//
// It is what backup.New takes. It reads through the catalog and translator of
// the flows it was made from, so the backup package needs neither the device
// nor its wire.
type Decoder struct {
	flows *Flows
}

// NewDecoder is a Decoder reading through f.
func NewDecoder(
	f *Flows,
) Decoder {
	return Decoder{flows: f}
}

// Document returns the preset body describes, or nil when the slot holds no
// blocks.
func (d Decoder) Document(
	ctx context.Context,
	body []byte,
	at slot.Address,
	name string,
) (*preset.Document, error) {
	read, err := d.flows.deviceReading(ctx, body, at.Slot, name, result.FormatPreset)
	if err != nil {
		return nil, err
	}

	return read.Doc, nil
}
