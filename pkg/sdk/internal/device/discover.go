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
	"fmt"
)

// modelFor returns the model with the given USB product identifier.
func modelFor(product uint16) (Model, bool) {
	for _, m := range models {
		if m.ProductID == product {
			return m, true
		}
	}

	return Model{}, false
}

// Devices returns every attached device this package recognises, in the order
// the lister reported them.
//
// Devices from other vendors are ignored. A Line 6 device with an unrecognised
// product identifier is also ignored rather than reported as an error — a bus
// may legitimately hold hardware this package does not know.
func Devices(ctx context.Context, l Lister) ([]Device, error) {
	descs, err := l.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing usb devices: %w", err)
	}

	found := make([]Device, 0, len(descs))

	for _, d := range descs {
		if d.Vendor != vendorID {
			continue
		}

		m, ok := modelFor(d.Product)
		if !ok {
			continue
		}

		found = append(found, Device{Model: m.Name, DeviceID: m.DeviceID, Descriptor: d})
	}

	return found, nil
}

// first returns the single attached device this package recognises.
//
// It reports ErrNoDevice when none is attached. When more than one is present
// it returns the first the bus reported, because there is no basis for
// preferring one over another — a caller that cares should use Devices.
func first(ctx context.Context, l Lister) (Device, error) {
	found, err := Devices(ctx, l)
	if err != nil {
		return Device{}, err
	}

	if len(found) == 0 {
		return Device{}, ErrNoDevice
	}

	return found[0], nil
}
