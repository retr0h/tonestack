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
package attached

import (
	"context"
	"fmt"

	"github.com/retr0h/tonestack/pkg/sdk/internal/device"
	"github.com/retr0h/tonestack/pkg/sdk/result"
)

// Lister reports the devices currently attached. device.Lister satisfies it.
type Lister interface {
	List(ctx context.Context) ([]device.Descriptor, error)
}

// ListWith reports every device the lister returns and this package
// recognises. Taking the lister makes this testable without hardware.
func ListWith(ctx context.Context, l Lister) (result.Attached, error) {
	found, err := device.Devices(ctx, l)
	if err != nil {
		return result.Attached{}, fmt.Errorf("finding devices: %w", err)
	}

	out := make([]result.Attachment, 0, len(found))

	for _, d := range found {
		out = append(out, result.Attachment{
			Model:    d.Model,
			DeviceID: d.DeviceID,
			Vendor:   d.Descriptor.Vendor,
			Product:  d.Descriptor.Product,
			Bus:      d.Descriptor.Bus,
			Address:  d.Descriptor.Address,
		})
	}

	return result.Attached{Devices: out}, nil
}
