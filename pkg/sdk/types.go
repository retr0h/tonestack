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

// Package sdk talks to Line 6 Helix-family hardware over USB.
//
// Discovery and identification are pure functions over a [Lister], so they are
// testable without a device attached. The libusb-backed Lister lives in
// usb.go and is the only part that needs hardware.
package sdk

import (
	"context"

	"github.com/retr0h/tonestack/pkg/sdk/wire"
)

// vendorID is Line 6's USB vendor identifier. Every device this package
// recognises reports it.
const vendorID uint16 = 0x0e41

// Model names one Line 6 device by its USB product identifier.
type Model struct {
	// Name is the device as Line 6 markets it.
	Name string
	// ProductID is the USB product identifier it reports.
	ProductID uint16
	// DeviceID is the integer written to a preset's data.device field.
	// Zero means the mapping is not yet established.
	DeviceID int
}

// models are the devices this package recognises.
//
// Product identifiers come from observing the bus; device identifiers come
// from the preset corpus. A device absent here is still reachable over USB but
// will not be named, and presets cannot be written for it.
var models = []Model{
	{Name: "HX Stomp", ProductID: 0x4246, DeviceID: 2162694},
	{Name: "HX Stomp XL", ProductID: 0x4253, DeviceID: 2162699},
	{Name: "Helix Floor", ProductID: 0x4248, DeviceID: 2162689},
	{Name: "Helix LT", ProductID: 0x424a, DeviceID: 2162692},
}

// Descriptor is a device as the bus reports it, before this package has
// decided whether it recognises it.
type Descriptor struct {
	Vendor  uint16
	Product uint16
	Bus     int
	Address int
}

// Device is an attached device this package recognises.
type Device struct {
	// Model is the device as Line 6 markets it.
	Model string
	// DeviceID is what a preset for this device carries in data.device.
	DeviceID int
	// Descriptor is how the bus reported it.
	Descriptor Descriptor
}

// Lister reports the devices currently attached to the bus.
//
// Implementations return every device they can see; filtering to Line 6
// hardware is this package's job, not theirs.
type Lister interface {
	List(ctx context.Context) ([]Descriptor, error)
}

// Bus is a lister holding something that needs releasing.
//
// Returned rather than the type behind it, so that a caller can be handed a
// bus instead of finding one — which is what lets the code around it be
// tested without hardware.
type Bus interface {
	Lister

	// Close releases whatever the lister holds.
	Close() error
}

// Writer is a session that can put a preset on a device.
//
// Separate from Editor because writing is the half that can destroy
// somebody's work, and a caller that only reads should not be handed the
// ability to.
type Writer interface {
	// WritePreset puts a document into a slot, leaving its name alone.
	WritePreset(ctx context.Context, setlist, slot int, document []byte) error
	// WriteNamedPreset puts a document into a slot under a name.
	WriteNamedPreset(
		ctx context.Context, setlist, slot int, name string, document []byte,
	) error
}

// Selector is a session that can change which preset a device is playing.
//
// Separate from Writer: selecting changes what comes out of the amplifier and
// changes nothing the device holds, where writing overwrites a slot. A caller
// that only wants to switch presets should not be handed the ability to
// overwrite one.
type Selector interface {
	// SelectPreset loads a preset, the way a footswitch does.
	SelectPreset(ctx context.Context, setlist, slot int) error
}

// Editor is a session with an attached device.
//
// What everything above this package needs from one: what it is, what it
// holds, and one preset at a time. *session satisfies it, and so does a mock,
// which is what lets the code that reads a device be tested without one.
type Editor interface {
	// Model is which device answered.
	Model() Model
	// Presets lists what a setlist holds.
	Presets(ctx context.Context, setlist int) ([]wire.Preset, error)
	// ReadPreset fetches one slot without loading it. No bytes and no error
	// is a slot holding no preset.
	ReadPreset(ctx context.Context, setlist, slot int) ([]byte, error)
	// Close releases the device.
	Close()
}
