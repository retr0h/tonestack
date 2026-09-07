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

// Every method here is one expression handing a call to libusb, so that
// nothing above it has to. Finding a device, choosing between two, claiming an
// interface and waiting on a busy one are all in discover_bus.go, where they
// run against a bus a test provides.
//
// This is the only file in the package a test cannot reach, and it holds no
// decisions for a test to check.
package sdk

import (
	"context"
	"fmt"

	"github.com/google/gousb"
)

// USBLister lists devices through libusb.
type USBLister struct {
	ctx *gousb.Context
}

// NewUSBLister returns a Lister backed by libusb.
//
// The caller must Close it; libusb holds an open context until then.
func NewUSBLister() Bus { return &USBLister{ctx: gousb.NewContext()} }

// Close releases the libusb context.
func (l *USBLister) Close() error {
	if l.ctx == nil {
		return nil
	}

	return l.ctx.Close()
}

// List reports every device currently on the bus.
//
// Devices are enumerated by descriptor only — none is opened, so this needs no
// special privileges and cannot disturb a device in use by other software.
func (l *USBLister) List(_ context.Context) ([]Descriptor, error) {
	var found []Descriptor

	_, err := l.ctx.OpenDevices(func(d *gousb.DeviceDesc) bool {
		found = append(found, descriptorOf(d))

		return false // the descriptor is enough; do not open it
	})
	if err != nil {
		return nil, fmt.Errorf("enumerating usb devices: %w", err)
	}

	return found, nil
}

// openUSB returns the bus libusb sees.
func openUSB() bus { return usbBus{ctx: gousb.NewContext()} }

// usbBus is a libusb context as a bus.
type usbBus struct {
	ctx *gousb.Context
}

// Devices opens every device the matcher accepts.
func (b usbBus) Devices(
	match func(vendor, product uint16) bool,
) ([]handle, error) {
	devs, err := b.ctx.OpenDevices(func(d *gousb.DeviceDesc) bool {
		return match(uint16(d.Vendor), uint16(d.Product))
	})

	out := make([]handle, 0, len(devs))
	for _, d := range devs {
		out = append(out, usbHandle{dev: d})
	}

	return out, err
}

// Close releases the libusb context.
func (b usbBus) Close() error { return b.ctx.Close() }

// usbHandle is an open device.
type usbHandle struct {
	dev *gousb.Device
}

// Descriptor is what the device says it is.
func (h usbHandle) Descriptor() Descriptor { return descriptorOf(h.dev.Desc) }

// Close releases the device.
func (h usbHandle) Close() error { return h.dev.Close() }

// Claim takes the editor interface, and returns how to give it back.
func (h usbHandle) Claim() (endpoints, func(), error) {
	intf, release, err := h.dev.DefaultInterface()
	if err != nil {
		return nil, nil, err
	}

	return usbEndpoints{intf: intf}, release, nil
}

// usbEndpoints is a claimed interface.
type usbEndpoints struct {
	intf *gousb.Interface
}

// Out is where requests go.
func (e usbEndpoints) Out() (sender, error) { return e.intf.OutEndpoint(endpointOut) }

// In is where answers come from.
func (e usbEndpoints) In() (receiver, error) { return e.intf.InEndpoint(endpointIn) }

// descriptorOf reads what a device says it is.
func descriptorOf(d *gousb.DeviceDesc) Descriptor {
	return Descriptor{
		Vendor:  uint16(d.Vendor),
		Product: uint16(d.Product),
		Bus:     d.Bus,
		Address: d.Address,
	}
}
