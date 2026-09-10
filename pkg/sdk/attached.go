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

// Attached is what is on the bus that this recognises.
//
// Recognised rather than everything: a bus holds keyboards and webcams, and a
// list of those is not an answer to "what can I write a preset to".
type Attached struct {
	// Devices are what was found, in the order the bus reported them.
	Devices []Attachment
}

// Attachment is one device on the bus.
type Attachment struct {
	// Model is the device as Line 6 markets it.
	Model string
	// DeviceID is what a preset for this device carries in data.device,
	// which is how a preset says which hardware it was made for.
	DeviceID int
	// Vendor and Product are how the bus identifies it.
	Vendor  uint16
	Product uint16
	// Bus and Address are where it is plugged in. They change between
	// unpluggings, so they identify a device now and not later.
	Bus     int
	Address int
}
