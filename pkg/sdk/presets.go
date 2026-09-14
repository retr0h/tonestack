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
	"github.com/retr0h/tonestack/pkg/sdk/slot"
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
}

// OnDevice says whether this addresses the hardware rather than a file.
func (w Where) OnDevice() bool { return w.Path == "" }

// deps are the collaborators every flow is handed: the Client's own catalog,
// and where a device's answers are captured.
func (c *Client) deps(
	ctx context.Context,
) slots.Deps {
	return slots.Deps{
		Catalogs: catalogs{client: c, ctx: ctx},
		Capture:  c.opts.capture,
	}
}

// once opens a Session, makes one call on it and closes it.
//
// What the one-shot device methods are: the CLI runs one operation per
// process, and a caller who wants several opens a Session instead.
func once[T any](
	ctx context.Context,
	c *Client,
	call func(*Session) (T, error),
) (T, error) {
	s, err := c.Open(ctx)
	if err != nil {
		var zero T

		return zero, err
	}

	// Deferred, so a flow that panics still lets the pedal go on its way up
	// the stack. The call's own answer is what the caller needs: a read loop
	// that ended during the call already failed it with that error, and one
	// that ended afterwards changed nothing the call reported.
	defer func() { _ = s.Close() }()

	return call(s)
}

// Presets reports what a setlist holds, slot by slot.
func (c *Client) Presets(
	ctx context.Context,
	in Where,
) (Listing, error) {
	if in.OnDevice() {
		return once(ctx, c, func(s *Session) (Listing, error) {
			return s.Presets(ctx, in.Setlist)
		})
	}

	return slots.List(slots.ListOptions{
		Deps:        c.deps(ctx),
		Path:        in.Path,
		Setlist:     in.Setlist,
		CatalogPath: c.opts.catalog,
	})
}

// Preset reads one slot as the rig it describes.
//
// A rig, not a rendering of one. What comes out compiles back into the preset
// it came from, unchanged.
func (c *Client) Preset(
	ctx context.Context,
	in Read,
) (Reading, error) {
	// A standalone .hlx is neither a device nor a setlist, so it is asked
	// for by name and answered before either.
	if in.File != "" || !in.OnDevice() {
		return slots.Show(slots.ShowOptions{
			Deps:        c.deps(ctx),
			Path:        in.Path,
			File:        in.File,
			Setlist:     in.Setlist,
			Slot:        in.Slot,
			CatalogPath: c.opts.catalog,
		})
	}

	return once(ctx, c, func(s *Session) (Reading, error) {
		return s.Preset(ctx, in.address())
	})
}

// address is the slot a Read names on a device.
func (r Read) address() slot.Address {
	return slot.Address{Setlist: r.Setlist, Slot: r.Slot}
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
func (c *Client) Export(
	ctx context.Context,
	in Export,
) (Written, error) {
	if in.OnDevice() {
		return once(ctx, c, func(s *Session) (Written, error) {
			return s.Export(ctx, slot.Address{Setlist: in.Setlist, Slot: in.Slot},
				in.OutputPath, in.As)
		})
	}

	return slots.Export(slots.ExportOptions{
		Deps:        c.deps(ctx),
		Path:        in.Path,
		Setlist:     in.Setlist,
		Slot:        in.Slot,
		OutputPath:  in.OutputPath,
		As:          slots.Format(in.As),
		CatalogPath: c.opts.catalog,
	})
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
}

// Import puts a preset file into a slot.
//
// Whatever the slot held is gone. A device has no undo, so what was there is
// read and kept first, in the directory WithBackupDir named; a file is left
// alone and the result goes somewhere new.
func (c *Client) Import(
	ctx context.Context,
	in Put,
) (Change, error) {
	if in.OnDevice() {
		return once(ctx, c, func(s *Session) (Change, error) {
			return s.Import(ctx, in.File, slot.Address{Setlist: in.Setlist, Slot: in.Slot})
		})
	}

	return slots.Import(slots.ImportOptions{
		BackupDir:   c.opts.backupDir,
		Deps:        c.deps(ctx),
		Path:        in.Path,
		File:        in.File,
		Setlist:     in.Setlist,
		Slot:        in.Slot,
		OutputPath:  in.OutputPath,
		CatalogPath: c.opts.catalog,
	})
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
}

// editOptions is what the flow takes for one edit.
func (c *Client) editOptions(
	ctx context.Context,
	e Edit,
) slots.EditOptions {
	return slots.EditOptions{
		Deps:        c.deps(ctx),
		Path:        e.Path,
		FromSetlist: e.FromSetlist,
		FromSlot:    e.FromSlot,
		ToSetlist:   e.ToSetlist,
		ToSlot:      e.ToSlot,
		OutputPath:  e.OutputPath,
		BackupDir:   c.opts.backupDir,
		CatalogPath: c.opts.catalog,
	}
}

// Copy puts what one slot holds into another.
//
// The preset moves exactly as it was written. Nothing is decoded and nothing
// is rebuilt, which is what makes this the safest thing to write: a device
// seeks through a preset by a table of byte offsets, and the surest way to
// keep those right is to change nothing.
//
// Edit already has FromSetlist and ToSetlist, one per side of the move.
// Where.Setlist has no side to belong to, so setting it is refused with
// ErrEditSetlist rather than silently read as neither.
func (c *Client) Copy(
	ctx context.Context,
	in Edit,
) (Change, error) {
	if in.Setlist != 0 {
		return Change{}, ErrEditSetlist
	}

	if in.OnDevice() {
		return once(ctx, c, func(s *Session) (Change, error) {
			return s.Copy(ctx, in.from(), in.to())
		})
	}

	return slots.Copy(c.editOptions(ctx, in))
}

// Swap exchanges what two slots hold.
//
// This is what moving a preset means: a slot cannot be left blank without
// writing an empty preset, and an empty preset carries routing that differs
// by device and firmware. Swapping invents nothing.
//
// Where.Setlist is refused the same way Copy refuses it; see ErrEditSetlist.
func (c *Client) Swap(
	ctx context.Context,
	in Edit,
) (Change, error) {
	if in.Setlist != 0 {
		return Change{}, ErrEditSetlist
	}

	if in.OnDevice() {
		return once(ctx, c, func(s *Session) (Change, error) {
			return s.Swap(ctx, in.from(), in.to())
		})
	}

	return slots.Swap(c.editOptions(ctx, in))
}

// from is the slot an Edit moves from, on a device.
func (e Edit) from() slot.Address {
	return slot.Address{Setlist: e.FromSetlist, Slot: e.FromSlot}
}

// to is the slot an Edit moves to, on a device.
func (e Edit) to() slot.Address {
	return slot.Address{Setlist: e.ToSetlist, Slot: e.ToSlot}
}

// Select makes one preset the active one on an attached device.
//
// The device loads it and starts making that sound. Nothing is written: the
// preset goes into the edit buffer and the slot it came from is untouched, so
// this is the one device operation that changes what you hear without
// changing what the device holds.
//
// A slot is only ever selected on the device that plays it. Where.Path or
// Read.File naming a file is refused with ErrSelectNeedsDevice rather than
// quietly going to the pedal anyway.
func (c *Client) Select(
	ctx context.Context,
	in Read,
) (Change, error) {
	if in.Path != "" || in.File != "" {
		return Change{}, ErrSelectNeedsDevice
	}

	return once(ctx, c, func(s *Session) (Change, error) {
		return s.Select(ctx, in.address())
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
}

// Compile builds a preset from a rig on disk.
func (c *Client) Compile(
	ctx context.Context,
	in Compile,
) (Built, error) {
	if err := ctx.Err(); err != nil {
		return Built{}, err
	}

	return slots.Compile(slots.CompileOptions{
		Deps:         c.deps(ctx),
		RigPath:      in.RigPath,
		OutputPath:   in.OutputPath,
		TemplatePath: in.TemplatePath,
		CatalogPath:  c.opts.catalog,
	})
}
