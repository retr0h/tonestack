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

package slots

import (
	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/chain"
	"github.com/retr0h/tonestack/pkg/sdk/compile"
	"github.com/retr0h/tonestack/pkg/sdk/device/wire"
	"github.com/retr0h/tonestack/pkg/sdk/editor"
	"github.com/retr0h/tonestack/pkg/sdk/preset"
	riggen "github.com/retr0h/tonestack/pkg/sdk/rig/gen"
)

// Catalogs opens the catalog a command reads model names out of.
type Catalogs interface {
	// Open reads the catalog at path, or the built-in one when path is empty.
	Open(path string) (*catalog.Catalog, error)
}

// Compiler moves between a preset and a rig.
//
// Two of the four methods pkg/compile carries, because reading and writing a
// slot is all this package does with it.
type Compiler interface {
	// Lift reads a preset into a rig.
	Lift(doc *preset.Document, cat *catalog.Catalog) (riggen.RigSpec, error)
	// Lower writes a rig back into a preset.
	Lower(doc *preset.Document, spec riggen.RigSpec, cat *catalog.Catalog) error
}

// Translator moves between what a device says and what a preset holds.
type Translator interface {
	// Chain reads what a device laid out as a chain.
	Chain(name string, got wire.DevicePreset, cat *catalog.Catalog) (chain.Chain, error)
	// Controllers carries what an expression pedal or a footswitch moves.
	Controllers(got wire.DevicePreset, cat *catalog.Catalog) *[]riggen.Controller
	// DeviceState carries the routing a device wraps a chain in.
	DeviceState(got wire.DevicePreset, cat *catalog.Catalog) *riggen.DeviceState
	// Document builds the preset a device's answer describes.
	Document(
		got wire.DevicePreset, cat *catalog.Catalog, name string,
	) (*preset.Document, bool, error)
	// Footswitches carries what the pedal prints under each switch.
	Footswitches(got wire.DevicePreset, cat *catalog.Catalog) *[]riggen.Footswitch
	// Placements turns a preset into what a device lays out.
	Placements(doc *preset.Document, cat *catalog.Catalog) ([]wire.Placement, error)
	// Snapshots carries what the device recalls on a footswitch.
	Snapshots(got wire.DevicePreset) *[]riggen.Snapshot
}

// Deps are the collaborators a command works through.
//
// Embedded in every options struct, and every field optional: a zero value
// reaches the real thing, so a caller names only what it wants to stand
// something else in for. This is the shape net/http gives a Client, whose
// nil Transport means the default one.
//
// It is what lets a command be tested without a catalog on disk, which is the
// same argument the sdk.Editor parameter on ShowWith and its siblings already
// makes for the device.
type Deps struct {
	// Catalogs opens catalogs. Nil reads them from disk.
	Catalogs Catalogs
	// Compiler moves between a preset and a rig. Nil uses pkg/compile.
	Compiler Compiler
	// Translator reads what a device says. Nil uses pkg/editor.
	Translator Translator
}

func (d Deps) catalogs() Catalogs {
	if d.Catalogs != nil {
		return d.Catalogs
	}

	return catalog.Files{}
}

func (d Deps) compiler() Compiler {
	if d.Compiler != nil {
		return d.Compiler
	}

	return compile.New()
}

func (d Deps) translator() Translator {
	if d.Translator != nil {
		return d.Translator
	}

	return editor.New()
}
