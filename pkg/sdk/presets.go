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

	"github.com/retr0h/tonestack/pkg/sdk/slot"
)

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

// Presets reports what a setlist on the attached device holds, slot by slot.
//
// Each of the Client's device methods opens a Session, makes one call on it
// and closes it, so each costs a claim and a handshake. A caller making several
// opens a Session instead. For a .hls or .hlb on disk, see Setlist.
func (c *Client) Presets(
	ctx context.Context,
	setlist int,
) (Listing, error) {
	return once(ctx, c, func(s *Session) (Listing, error) {
		return s.Presets(ctx, setlist)
	})
}

// Preset reads one slot on the attached device as the rig it describes.
//
// A rig, not a rendering of one. What comes out compiles back into the preset
// it came from, unchanged.
func (c *Client) Preset(
	ctx context.Context,
	at slot.Address,
) (Reading, error) {
	return once(ctx, c, func(s *Session) (Reading, error) {
		return s.Preset(ctx, at)
	})
}

// Export writes one slot on the attached device out to a file, as a rig or as
// the device's own file.
func (c *Client) Export(
	ctx context.Context,
	at slot.Address,
	out string,
	as Format,
) (Written, error) {
	return once(ctx, c, func(s *Session) (Written, error) {
		return s.Export(ctx, at, out, as)
	})
}

// Import puts a preset file into a slot on the attached device.
//
// Whatever the slot held is gone. A device has no undo, so what was there is
// read and kept first, in the directory WithBackupDir named.
func (c *Client) Import(
	ctx context.Context,
	file string,
	at slot.Address,
) (Change, error) {
	return once(ctx, c, func(s *Session) (Change, error) {
		return s.Import(ctx, file, at)
	})
}

// Copy puts what one slot on the attached device holds into another.
//
// The preset moves exactly as it was written. Nothing is decoded and nothing
// is rebuilt, which is what makes this the safest thing to write: a device
// seeks through a preset by a table of byte offsets, and the surest way to
// keep those right is to change nothing.
func (c *Client) Copy(
	ctx context.Context,
	from, to slot.Address,
) (Change, error) {
	return once(ctx, c, func(s *Session) (Change, error) {
		return s.Copy(ctx, from, to)
	})
}

// Swap exchanges what two slots on the attached device hold.
//
// This is what moving a preset means: a slot cannot be left blank without
// writing an empty preset, and an empty preset carries routing that differs
// by device and firmware. Swapping invents nothing.
func (c *Client) Swap(
	ctx context.Context,
	a, b slot.Address,
) (Change, error) {
	return once(ctx, c, func(s *Session) (Change, error) {
		return s.Swap(ctx, a, b)
	})
}

// Select makes one preset the active one on the attached device.
//
// The device loads it and starts making that sound. Nothing is written: the
// preset goes into the edit buffer and the slot it came from is untouched, so
// this is the one device operation that changes what you hear without
// changing what the device holds.
func (c *Client) Select(
	ctx context.Context,
	at slot.Address,
) (Change, error) {
	return once(ctx, c, func(s *Session) (Change, error) {
		return s.Select(ctx, at)
	})
}

// PresetFile reads a standalone .hlx as the rig it describes.
//
// The same rig a slot on a device or in a setlist reads as, which is the
// point: what the device holds and what this tool generates are the same kind
// of thing.
func (c *Client) PresetFile(
	ctx context.Context,
	path string,
) (Reading, error) {
	return c.operations().ShowFile(ctx, path)
}

// Compile says what rig to build, what to build it into, and where the preset
// goes.
//
// A struct rather than three arguments, because three paths of one type are
// too easy to pass in the wrong order.
type Compile struct {
	// Rig is the rig file to build.
	Rig string
	// Template is a preset to write the chain into. Empty uses an untouched
	// one the device itself wrote.
	Template string
	// Out is where the preset is written.
	Out string
}

// Compile builds a preset from a rig on disk.
func (c *Client) Compile(
	ctx context.Context,
	in Compile,
) (Built, error) {
	return c.operations().Compile(ctx, in.Rig, in.Template, in.Out)
}
