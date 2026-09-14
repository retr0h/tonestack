//go:build darwin

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

// The macOS backend: IOKit, called from Go through purego, so a binary built
// with CGO_ENABLED=0 still reaches a device.
//
// Every backend implements the same five seams, bus, handle, endpoints,
// sender and receiver, plus Bus for listing. What a backend decides lives in
// backend.go, and finding a device, choosing between two and waiting on a busy
// one live in discover_bus.go, both where a test reaches them. What is here is
// only the translation into IOKit, which no test reaches without hardware.
package device

import (
	"context"
	"errors"
	"time"

	"github.com/go-macos/iokit/ioreturn"
	"github.com/go-macos/iokit/usb"
)

const (
	// writeTimeout bounds one write. A device whose queue has filled stops
	// accepting writes, and that is reported rather than retried: see the
	// rules in docs/protocol.md.
	writeTimeout = 5 * time.Second
	// readSlice is how long one read waits before looking at the context
	// again. IOKit reads take a timeout, not a context, and a read has to stay
	// posted for as long as a session is open.
	readSlice = 100 * time.Millisecond
	// editorInterface is where HX Edit and this package talk to the device.
	editorInterface = 0
)

// ErrNoUSBSupport reports an operating system with no USB backend yet. This
// one has a backend, so nothing here returns it.
var ErrNoUSBSupport = errors.New("device access is not supported on this operating system yet")

// USBLister lists devices through IOKit.
type USBLister struct{}

// NewUSBLister returns a Lister backed by IOKit.
func NewUSBLister() Bus { return &USBLister{} }

// Close does nothing: IOKit holds no context between calls.
func (*USBLister) Close() error { return nil }

// List reports every device currently on the bus.
//
// Devices are enumerated from the registry only. None is opened, so this needs
// no special privileges and cannot disturb a device in use by other software.
func (*USBLister) List(_ context.Context) ([]Descriptor, error) {
	devs, err := usb.Devices(usb.Filter{})

	return listed(devs, err, describeDevice, closeDevice)
}

// openUSB returns the bus IOKit sees.
func openUSB() bus { return iokitBus{} }

// iokitBus is the IOKit registry as a bus.
type iokitBus struct{}

// Devices returns every device the matcher accepts. Nothing is opened: IOKit
// claims an interface without the device being opened first.
func (iokitBus) Devices(match func(vendor, product uint16) bool) ([]handle, error) {
	devs, err := usb.Devices(usb.Filter{})

	return found(devs, err, deviceIDs, match, closeDevice, wrapDevice)
}

// Close does nothing: IOKit holds no context between calls.
func (iokitBus) Close() error { return nil }

// iokitHandle is one device in the registry.
type iokitHandle struct {
	dev *usb.Device
}

// Descriptor is what the device says it is.
func (h iokitHandle) Descriptor() Descriptor { return describeDevice(h.dev) }

// Close releases the device's registry reference.
func (h iokitHandle) Close() error { return h.dev.Close() }

// Claim takes the editor interface, and returns how to give it back.
//
// A plain open, never a seize. HX Edit holds this interface exclusively while
// it runs, and taking it would cut HX Edit off mid-conversation; the refusal
// is reported instead, as the reason to quit HX Edit.
func (h iokitHandle) Claim() (endpoints, func(), error) {
	info := h.dev.Info()

	ifaces, err := usb.Interfaces(usb.InterfaceFilter{
		VendorID:   info.VendorID,
		ProductIDs: []uint16{info.ProductID},
		LocationID: info.LocationID,
		Numbers:    []uint8{editorInterface},
	})

	intf, err := claimOne(ifaces, err, editorInterface, closeInterface, openInterface, busy)
	if err != nil {
		return nil, nil, err
	}

	return iokitEndpoints{intf: intf}, func() { closeInterface(intf) }, nil
}

// iokitEndpoints is a claimed interface.
type iokitEndpoints struct {
	intf *usb.InterfaceHandle
}

// Out is where requests go.
func (e iokitEndpoints) Out() (sender, error) {
	p, err := e.intf.Pipe(endpointOut)

	return piped(p.Ref, err, e.sender)
}

// In is where answers come from.
func (e iokitEndpoints) In() (receiver, error) {
	p, err := e.intf.Pipe(endpointIn)

	return piped(p.Ref, err, e.receiver)
}

// sender is the write half of one pipe.
func (e iokitEndpoints) sender(ref uint8) sender { return iokitSender{intf: e.intf, ref: ref} }

// receiver is the read half of one pipe.
func (e iokitEndpoints) receiver(
	ref uint8,
) receiver {
	return iokitReceiver{intf: e.intf, ref: ref}
}

// iokitSender writes to one pipe.
type iokitSender struct {
	intf *usb.InterfaceHandle
	ref  uint8
}

// Write sends p, or reports that the device did not take it in time.
func (s iokitSender) Write(p []byte) (int, error) {
	return s.intf.Write(s.ref, p, writeTimeout)
}

// iokitReceiver reads from one pipe.
type iokitReceiver struct {
	intf *usb.InterfaceHandle
	ref  uint8
}

// ReadContext waits for the device to send something, or for ctx to end.
func (r iokitReceiver) ReadContext(ctx context.Context, p []byte) (int, error) {
	return readUntil(ctx, p, readSlice, r.read, idle)
}

// read is one IOKit read on this pipe.
func (r iokitReceiver) read(p []byte, timeout time.Duration) (int, error) {
	return r.intf.Read(r.ref, p, timeout)
}

// idle reports a read that timed out because the device had nothing to say.
func idle(err error) bool { return is(err, ioreturn.USBTransactionTimeout) }

// busy reports an interface somebody else holds.
func busy(err error) bool { return is(err, ioreturn.ExclusiveAccess) }

// is reports whether err is an IOKit error carrying code.
func is(err error, code ioreturn.Code) bool {
	var e *usb.IOError

	return errors.As(err, &e) && e.Code == code
}

// deviceIDs reads a device's vendor and product.
func deviceIDs(d *usb.Device) (vendor, product uint16) {
	i := d.Info()

	return i.VendorID, i.ProductID
}

// describeDevice reads what a device says it is.
func describeDevice(d *usb.Device) Descriptor {
	i := d.Info()

	return located(i.VendorID, i.ProductID, i.LocationID)
}

// wrapDevice makes a registry device into a handle.
func wrapDevice(d *usb.Device) handle { return iokitHandle{dev: d} }

// closeDevice gives back a device reference nobody kept.
// Best effort: nothing holds the reference, so a refusal has nobody to tell.
func closeDevice(d *usb.Device) { _ = d.Close() }

// openInterface claims an interface without seizing it.
func openInterface(i *usb.InterfaceHandle) error { return i.Open() }

// closeInterface gives back an interface reference.
// Best effort: the claim is over either way, and a refusal has nobody to tell.
func closeInterface(i *usb.InterfaceHandle) { _ = i.Close() }
