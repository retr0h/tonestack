//go:build !cgo

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
	"errors"
)

// ErrNoUSBSupport reports a binary built without cgo, which cannot reach a
// device.
//
// Talking to hardware needs libusb through cgo. Everything else — describing a
// chain, validating it, writing a preset — is pure Go, so a build without cgo
// is useful and should say what it cannot do rather than fail to compile.
var ErrNoUSBSupport = errors.New("built without usb support: rebuild with cgo enabled")

// USBLister is the no-cgo stand-in. Its methods report ErrNoUSBSupport.
type USBLister struct{}

// NewUSBLister returns a Lister that cannot reach a device.
func NewUSBLister() Bus { return &USBLister{} }

// openUSB reports that this build cannot reach a bus.
func openUSB() bus { return noBus{} }

// noBus is what a build without cgo has instead of a bus.
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
