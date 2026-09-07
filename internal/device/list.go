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
	"io"

	"github.com/retr0h/tonestack/internal/cli"
	"github.com/retr0h/tonestack/pkg/sdk"
)

// Lister reports the devices currently attached. sdk.Lister satisfies it.
type Lister interface {
	List(ctx context.Context) ([]sdk.Descriptor, error)
}

// ListWith writes every device the lister reports and this package
// recognises. Taking the lister makes the reporting testable without hardware.
func ListWith(ctx context.Context, w io.Writer, l Lister) error {
	found, err := sdk.Devices(ctx, l)
	if err != nil {
		return fmt.Errorf("finding devices: %w", err)
	}

	rows := make([][]string, 0, len(found))

	for _, d := range found {
		rows = append(rows, []string{
			cli.Accent(w, d.Model),
			cli.Mute(w, fmt.Sprintf("%04x:%04x",
				d.Descriptor.Vendor, d.Descriptor.Product)),
			cli.Mute(w, fmt.Sprintf("%d.%d",
				d.Descriptor.Bus, d.Descriptor.Address)),
			fmt.Sprintf("%d", d.DeviceID),
		})
	}

	if err := (cli.Section{
		Title:   "Attached",
		Headers: []string{"device", "usb", "bus", "preset device id"},
		Rows:    rows,
		Empty:   "no Helix devices attached",
	}).Render(w); err != nil {
		return fmt.Errorf("reporting: %w", err)
	}

	return nil
}
