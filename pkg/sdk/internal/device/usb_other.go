//go:build !darwin

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
	"errors"
)

// ErrNoUSBSupport reports an operating system with no USB backend yet.
//
// Only macOS has one, in usb_darwin.go. Everything else here, describing a
// chain, validating it, writing a preset, works on any operating system, so a
// build for one without a backend is useful and says what it cannot do rather
// than failing to compile. A new backend is one file implementing the same
// seams as usb_darwin.go, with this file's build tag narrowed to exclude it.
var ErrNoUSBSupport = errors.New("device access is not supported on this operating system yet")

// USBLister is the stand-in where there is no backend. Its methods report ErrNoUSBSupport.
type USBLister struct{}

// NewUSBLister returns a Lister that cannot reach a device.
func NewUSBLister() Bus { return &USBLister{} }

// openUSB reports that this operating system has no bus backend.
func openUSB() bus { return noBus{} }

// noBus is what an operating system without a backend has instead of a bus.
type noBus struct{}

// Devices reports ErrNoUSBSupport.
func (noBus) Devices(func(vendor, product uint16) bool) ([]handle, error) {
	return nil, ErrNoUSBSupport
}

// Close does nothing.
func (noBus) Close() error { return nil }

// Close does nothing.
func (*USBLister) Close() error { return nil }

// List reports ErrNoUSBSupport.
func (*USBLister) List(_ context.Context) ([]Descriptor, error) {
	return nil, ErrNoUSBSupport
}
