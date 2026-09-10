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

package cli

import (
	"fmt"
	"io"

	"github.com/retr0h/tonestack/pkg/sdk"
)

// Attached prints what is on the bus, one device to a row.
func Attached(w io.Writer, a sdk.Attached) error {
	rows := make([][]string, 0, len(a.Devices))

	for _, d := range a.Devices {
		rows = append(rows, []string{
			Accent(w, d.Model),
			Mute(w, fmt.Sprintf("%04x:%04x", d.Vendor, d.Product)),
			Mute(w, fmt.Sprintf("%d.%d", d.Bus, d.Address)),
			fmt.Sprintf("%d", d.DeviceID),
		})
	}

	if err := (Section{
		Title:   "Attached",
		Headers: []string{"device", "usb", "bus", "preset device id"},
		Rows:    rows,
		Empty:   "no Helix devices attached",
	}).Render(w); err != nil {
		return fmt.Errorf("reporting: %w", err)
	}

	return nil
}
