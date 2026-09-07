//go:build cgo

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
	"context"
	"fmt"

	"github.com/google/gousb"
)

// USBLister lists devices through libusb. It is the only part of this package
// that needs hardware, and is excluded from the coverage gate for that reason.
//
// Its zero value is not usable; call NewUSBLister and close the result.
type USBLister struct {
	ctx *gousb.Context
}

// NewUSBLister returns a Lister backed by libusb.
//
// The caller must Close it; libusb holds an open context until then.
func NewUSBLister() Bus {
	return &USBLister{ctx: gousb.NewContext()}
}

// Close releases the libusb context.
func (l *USBLister) Close() error {
	if l.ctx == nil {
		return nil
	}

	if err := l.ctx.Close(); err != nil {
		return fmt.Errorf("closing usb context: %w", err)
	}

	return nil
}

// List reports every device currently on the bus.
//
// Devices are enumerated by descriptor only — none is opened, so this needs no
// special privileges and cannot disturb a device in use by other software.
func (l *USBLister) List(_ context.Context) ([]Descriptor, error) {
	var found []Descriptor

	_, err := l.ctx.OpenDevices(func(d *gousb.DeviceDesc) bool {
		found = append(found, Descriptor{
			Vendor:  uint16(d.Vendor),
			Product: uint16(d.Product),
			Bus:     d.Bus,
			Address: d.Address,
		})

		return false // descriptor is enough; do not open
	})
	if err != nil {
		return nil, fmt.Errorf("enumerating usb devices: %w", err)
	}

	return found, nil
}
