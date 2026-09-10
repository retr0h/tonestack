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

package editor

import (
	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/chain"
	"github.com/retr0h/tonestack/pkg/sdk/device/wire"
	"github.com/retr0h/tonestack/pkg/sdk/preset"
	riggen "github.com/retr0h/tonestack/pkg/sdk/rig/gen"
)

// Translator is this package's work as a value.
//
// It holds nothing, and every method is the package-level function of the
// same name. The type exists so that a caller can say what it depends on and
// stand something else in its place, which a package-level function does not
// allow.
//
// The interface a caller needs is the caller's to declare. Nothing wants all
// seven of these at once.
type Translator struct{}

// New returns a Translator.
func New() *Translator { return &Translator{} }

// Chain reads what a device laid out as a chain.
func (*Translator) Chain(
	name string,
	got wire.DevicePreset,
	cat *catalog.Catalog,
) (chain.Chain, error) {
	return Chain(name, got, cat)
}

// Controllers carries what an expression pedal or a footswitch moves.
func (*Translator) Controllers(
	got wire.DevicePreset,
	cat *catalog.Catalog,
) *[]riggen.Controller {
	return Controllers(got, cat)
}

// DeviceState carries the routing a device wraps a chain in.
func (*Translator) DeviceState(
	got wire.DevicePreset,
	cat *catalog.Catalog,
) *riggen.DeviceState {
	return DeviceState(got, cat)
}

// Document builds the preset a device's answer describes.
func (*Translator) Document(
	got wire.DevicePreset,
	cat *catalog.Catalog,
	name string,
) (*preset.Document, bool, error) {
	return Document(got, cat, name)
}

// Footswitches carries what the pedal prints under each switch.
func (*Translator) Footswitches(
	got wire.DevicePreset,
	cat *catalog.Catalog,
) *[]riggen.Footswitch {
	return Footswitches(got, cat)
}

// Placements turns a preset into what a device lays out.
func (*Translator) Placements(
	doc *preset.Document,
	cat *catalog.Catalog,
) ([]wire.Placement, error) {
	return Placements(doc, cat)
}

// Snapshots carries what the device recalls on a footswitch.
func (*Translator) Snapshots(got wire.DevicePreset) *[]riggen.Snapshot {
	return Snapshots(got)
}
