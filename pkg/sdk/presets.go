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

	"github.com/retr0h/tonestack/pkg/sdk/internal/slots"
)

// Where a setlist is read from.
//
// Every operation below takes one of these. An empty Path means the attached
// device, which is what somebody with one plugged in almost always wants; a
// path reads a backup instead and needs no hardware. Deciding that here rather
// than in each caller is what stops a terminal, a service and a TUI each
// inventing their own answer.
type Where struct {
	// Path is a .hls setlist or .hlb backup. Empty means the device.
	Path string
	// Setlist selects one setlist within a bundle.
	Setlist int
	// CatalogPath is a catalog to name gear against. Empty means the one
	// built into this binary.
	CatalogPath string
}

// OnDevice says whether this addresses the hardware rather than a file.
func (w Where) OnDevice() bool { return w.Path == "" }

// Presets reports what a setlist holds, slot by slot.
func (c *Client) Presets(ctx context.Context, in Where) (Listing, error) {
	if in.OnDevice() {
		return slots.ListDevice(ctx, slots.DeviceOptions{
			Setlist:     in.Setlist,
			CatalogPath: in.CatalogPath,
		})
	}

	return slots.List(slots.ListOptions{
		Path:        in.Path,
		Setlist:     in.Setlist,
		CatalogPath: in.CatalogPath,
	})
}

// Preset reads one slot as the rig it describes.
//
// A rig, not a rendering of one. What comes out compiles back into the preset
// it came from, unchanged.
func (c *Client) Preset(ctx context.Context, in Read) (Reading, error) {
	// A standalone .hlx is neither a device nor a setlist, so it is asked
	// for by name and answered before either.
	if in.File != "" || !in.OnDevice() {
		return slots.Show(slots.ShowOptions{
			Path:        in.Path,
			File:        in.File,
			Setlist:     in.Setlist,
			Slot:        in.Slot,
			CatalogPath: in.CatalogPath,
		})
	}

	return slots.ShowDevice(ctx, slots.DeviceOptions{
		Setlist:     in.Setlist,
		Slot:        in.Slot,
		CatalogPath: in.CatalogPath,
	})
}

// Read addresses one preset.
type Read struct {
	Where

	// File is a standalone .hlx to read instead of a slot.
	File string
	// Slot is the position within the setlist, from zero.
	Slot int
}

// Export writes one slot out to a file.
type Export struct {
	Where

	// Slot is the position to write out, from zero.
	Slot int
	// OutputPath is where the result goes.
	OutputPath string
	// As is the format. Empty writes a rig, which is what reads on other
	// hardware; "hlx" writes the device's own file, a faithful copy.
	As string
}

// Export writes one slot out.
func (c *Client) Export(ctx context.Context, in Export) (Written, error) {
	opts := slots.ExportOptions{
		Path:        in.Path,
		Setlist:     in.Setlist,
		Slot:        in.Slot,
		OutputPath:  in.OutputPath,
		As:          slots.Format(in.As),
		CatalogPath: in.CatalogPath,
	}

	if in.OnDevice() {
		return slots.ExportDevice(ctx, opts)
	}

	return slots.Export(opts)
}

// Put addresses a preset going into a slot.
type Put struct {
	Where

	// File is the .hlx to put there.
	File string
	// Slot is where it goes, from zero.
	Slot int
	// OutputPath is where the edited setlist is written. Ignored for a
	// device, which is written in place.
	OutputPath string
	// BackupDir is where the destination's old contents are kept. Empty
	// uses the state directory. Only a device is backed up: a file write
	// goes somewhere new and leaves the original alone.
	BackupDir string
}

// Import puts a preset file into a slot.
//
// Whatever the slot held is gone. A device has no undo, so what was there is
// read and kept first; a file is left alone and the result goes somewhere new.
func (c *Client) Import(ctx context.Context, in Put) (Change, error) {
	opts := slots.ImportOptions{
		BackupDir:   in.BackupDir,
		Path:        in.Path,
		File:        in.File,
		Setlist:     in.Setlist,
		Slot:        in.Slot,
		OutputPath:  in.OutputPath,
		CatalogPath: in.CatalogPath,
	}

	if in.OnDevice() {
		return slots.ImportDevice(ctx, opts)
	}

	return slots.Import(opts)
}

// Edit addresses two slots.
type Edit struct {
	Where

	// From and To address the source and the destination, from zero. Each
	// may name its own setlist within a bundle.
	FromSetlist int
	FromSlot    int
	ToSetlist   int
	ToSlot      int
	// OutputPath is where the edited setlist is written. Ignored for a
	// device, which is written in place.
	OutputPath string
	// BackupDir is where a device slot's old contents are kept. Empty uses
	// the state directory.
	BackupDir string
}

// options is what the flow takes.
func (e Edit) options() slots.EditOptions {
	return slots.EditOptions{
		Path:        e.Path,
		FromSetlist: e.FromSetlist,
		FromSlot:    e.FromSlot,
		ToSetlist:   e.ToSetlist,
		ToSlot:      e.ToSlot,
		OutputPath:  e.OutputPath,
		BackupDir:   e.BackupDir,
		CatalogPath: e.CatalogPath,
	}
}

// Copy puts what one slot holds into another.
//
// The preset moves exactly as it was written. Nothing is decoded and nothing
// is rebuilt, which is what makes this the safest thing to write: a device
// seeks through a preset by a table of byte offsets, and the surest way to
// keep those right is to change nothing.
func (c *Client) Copy(ctx context.Context, in Edit) (Change, error) {
	if in.OnDevice() {
		return slots.CopyDevice(ctx, in.options())
	}

	return slots.Copy(in.options())
}

// Swap exchanges what two slots hold.
//
// This is what moving a preset means: a slot cannot be left blank without
// writing an empty preset, and an empty preset carries routing that differs
// by device and firmware. Swapping invents nothing.
func (c *Client) Swap(ctx context.Context, in Edit) (Change, error) {
	if in.OnDevice() {
		return slots.SwapDevice(ctx, in.options())
	}

	return slots.Swap(in.options())
}

// Select makes one preset the active one on an attached device.
//
// The device loads it and starts making that sound. Nothing is written: the
// preset goes into the edit buffer and the slot it came from is untouched, so
// this is the one device operation that changes what you hear without
// changing what the device holds.
func (c *Client) Select(ctx context.Context, in Read) (Change, error) {
	return slots.SelectDevice(ctx, slots.DeviceOptions{
		Setlist: in.Setlist,
		Slot:    in.Slot,
	})
}

// Compile says what rig to build and where to put it.
type Compile struct {
	// RigPath is the rig to build.
	RigPath string
	// OutputPath is where the preset is written.
	OutputPath string
	// TemplatePath is a preset to write the chain into. Empty uses an
	// untouched one the device itself wrote.
	TemplatePath string
	// CatalogPath is the catalog to resolve models against. Empty means the
	// one built into this binary.
	CatalogPath string
}

// Compile builds a preset from a rig on disk.
func (c *Client) Compile(in Compile) (Built, error) {
	return slots.Compile(slots.CompileOptions{
		RigPath:      in.RigPath,
		OutputPath:   in.OutputPath,
		TemplatePath: in.TemplatePath,
		CatalogPath:  in.CatalogPath,
	})
}
