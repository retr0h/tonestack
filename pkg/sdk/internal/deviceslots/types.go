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

// Package deviceslots reads and edits the slots an attached device holds.
//
// The same operations fileslots does on a file, and selecting a preset, go
// through a session with the device here. A slot about to be overwritten is
// read and handed to Backups first, because a device has no undo, and a write
// whose backup failed does not happen.
//
// Every operation is a method on Flows, which holds what the caller was
// configured with. Each takes a session and only the addresses and paths that
// one call needs.
package deviceslots

import (
	"context"
	"io"

	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/chain"
	"github.com/retr0h/tonestack/pkg/sdk/internal/backup"
	"github.com/retr0h/tonestack/pkg/sdk/internal/compile"
	"github.com/retr0h/tonestack/pkg/sdk/internal/editor"
	"github.com/retr0h/tonestack/pkg/sdk/internal/wire"
	"github.com/retr0h/tonestack/pkg/sdk/preset"
	"github.com/retr0h/tonestack/pkg/sdk/rig"
)

// Catalogs hands over the catalog model names are read out of. The sdk Client
// satisfies it, and keeps the catalog it opened.
type Catalogs interface {
	// Catalog returns the catalog, opening it on first use.
	Catalog(ctx context.Context) (*catalog.Catalog, error)
}

// Compiler reads a preset into a rig.
//
// One of the four methods pkg/compile carries, because lifting what a device
// answered is all this package does with it.
type Compiler interface {
	// Lift reads a preset into a rig.
	Lift(doc *preset.Document, cat *catalog.Catalog) (rig.Spec, error)
}

// Translator moves between what a device says and what a preset holds.
type Translator interface {
	// Chain reads what a device laid out as a chain.
	Chain(name string, got wire.DevicePreset, cat *catalog.Catalog) (chain.Chain, error)
	// Controllers carries what an expression pedal or a footswitch moves.
	Controllers(got wire.DevicePreset, cat *catalog.Catalog) *[]rig.Controller
	// DeviceState carries the routing a device wraps a chain in.
	DeviceState(got wire.DevicePreset, cat *catalog.Catalog) *rig.DeviceState
	// Document builds the preset a device's answer describes.
	Document(
		got wire.DevicePreset, cat *catalog.Catalog, name string,
	) (*preset.Document, bool, error)
	// Footswitches carries what the pedal prints under each switch.
	Footswitches(got wire.DevicePreset, cat *catalog.Catalog) *[]rig.Footswitch
	// Placements turns a preset into what a device lays out.
	Placements(doc *preset.Document, cat *catalog.Catalog) ([]wire.Placement, error)
	// Snapshots carries what the device recalls on a footswitch.
	Snapshots(got wire.DevicePreset) *[]rig.Snapshot
	// SnapshotStates reads what a preset's own snapshots recall, for writing
	// them to a device.
	SnapshotStates(doc *preset.Document) []wire.Snapshot
	// RoutingStates reads what a preset wraps its chain in, for writing it to
	// a device. held is what the preset being written into already routes
	// like, which says how many values each entry takes.
	RoutingStates(
		doc *preset.Document, cat *catalog.Catalog, held []wire.DeviceRouting,
	) []wire.Routing
}

// Backups keeps what slots held before a write replaces them.
type Backups interface {
	// Keep writes each slot it is handed and returns where each one went.
	Keep(ctx context.Context, held ...backup.Held) ([]string, error)
}

// Flows are the operations on a device's slots, and what they were configured
// with.
//
// Built once by whoever owns the configuration, which is the sdk Client. Each
// flow is a method taking only what differs between two calls: a session, an
// address and, where there is one, a path. Nothing a call could set is left
// for a flow to ignore, because a flow that does not read a setting is not
// handed it.
//
// Every collaborator is optional: a nil one reaches the real thing, so a test
// names only what it stands something else in for. This is the shape net/http
// gives a Client, whose nil Transport means the default one.
type Flows struct {
	// Catalogs hands over the catalog. Nil reads the one built into this
	// binary.
	Catalogs Catalogs
	// Compiler moves a preset into a rig. Nil uses pkg/compile.
	Compiler Compiler
	// Translator reads what a device says. Nil uses pkg/editor.
	Translator Translator
	// Backups keeps a slot's old contents before a write. Nil keeps them in
	// the state directory, read through these flows.
	Backups Backups
	// Capture receives each device answer a read gets, verbatim. Nil keeps
	// nothing.
	Capture io.Writer
}

// catalog is the catalog these flows name gear against.
func (f *Flows) catalog(
	ctx context.Context,
) (*catalog.Catalog, error) {
	if f.Catalogs != nil {
		return f.Catalogs.Catalog(ctx)
	}

	return catalog.BuiltIn()
}

func (f *Flows) compiler() Compiler {
	if f.Compiler != nil {
		return f.Compiler
	}

	return compile.New()
}

func (f *Flows) translator() Translator {
	if f.Translator != nil {
		return f.Translator
	}

	return editor.New()
}

func (f *Flows) backups() Backups {
	if f.Backups != nil {
		return f.Backups
	}

	return backup.New("", NewDecoder(f))
}
