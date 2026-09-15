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

// Setlist is a .hls setlist or .hlb backup on disk, which HX Edit writes.
//
// It is addressed the way a Session addresses the device, and needs no
// hardware. An edit never changes the file it was read from: it always writes
// a new one, because a backup is often the only copy of what a pedal holds.
// That is why a Setlist's edits take an out path and a Session's do not.
//
// A Setlist is a name for a file and nothing more. Nothing is read until a
// method is called, and each call reads the file again.
type Setlist struct {
	flows *slots.Flows
	path  string
}

// Setlist addresses a .hls setlist or .hlb backup. Nothing is read yet.
func (c *Client) Setlist(
	path string,
) *Setlist {
	return &Setlist{flows: c.flows, path: path}
}

// Presets reports what one setlist in the file holds, slot by slot. A .hls
// holds one setlist; a .hlb holds several, counted from zero.
func (f *Setlist) Presets(
	ctx context.Context,
	setlist int,
) (Listing, error) {
	return f.flows.List(ctx, f.path, setlist)
}

// Preset reads one slot as the rig it describes.
func (f *Setlist) Preset(
	ctx context.Context,
	at slot.Address,
) (Reading, error) {
	return f.flows.Show(ctx, f.path, at)
}

// Export writes one slot out to a file of its own, as a rig or as the
// device's own file.
func (f *Setlist) Export(
	ctx context.Context,
	at slot.Address,
	out string,
	as Format,
) (Written, error) {
	return f.flows.Export(ctx, f.path, at, out, as)
}

// Import puts a preset file into a slot, and writes the edited setlist to out.
func (f *Setlist) Import(
	ctx context.Context,
	file string,
	at slot.Address,
	out string,
) (Change, error) {
	return f.flows.Import(ctx, f.path, file, at, out)
}

// Copy puts what one slot holds into another, and writes the edited setlist to
// out. Each address may name its own setlist within a bundle.
func (f *Setlist) Copy(
	ctx context.Context,
	from, to slot.Address,
	out string,
) (Change, error) {
	return f.flows.Copy(ctx, f.path, from, to, out)
}

// Swap exchanges what two slots hold, and writes the edited setlist to out.
func (f *Setlist) Swap(
	ctx context.Context,
	a, b slot.Address,
	out string,
) (Change, error) {
	return f.flows.Swap(ctx, f.path, a, b, out)
}
