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
	"context"
	"fmt"
	"io"

	"github.com/retr0h/tonestack/internal/cli"
	"github.com/retr0h/tonestack/pkg/sdk"
	slotpkg "github.com/retr0h/tonestack/pkg/slot"
)

// DeviceOptions says which setlist to read off an attached device.
type DeviceOptions struct {
	// Setlist selects one of the device's setlists, from zero.
	Setlist int
	// All includes slots holding nothing.
	All bool
	// Slot selects a position within the setlist, for reading one preset.
	Slot int
	// Name is what the device calls the preset, when it is already known.
	Name string
	// CatalogPath is the generated catalog for the target device.
	CatalogPath string
}

// ShowDevice reads one slot off an attached device.
//
// Read-only: the device hands back the preset and goes on playing whatever it
// was. Nothing is selected, loaded or written.
func ShowDevice(ctx context.Context, w io.Writer, opts DeviceOptions) error {
	s, err := sdk.Open(ctx)
	if err != nil {
		return err
	}

	defer s.Close()

	// The name comes from the listing rather than the preset: what the device
	// hands back for one slot does not carry it.
	if opts.Name == "" {
		if found, err := s.Presets(ctx, opts.Setlist); err == nil {
			if opts.Slot >= 0 && opts.Slot < len(found) {
				opts.Name = found[opts.Slot].Name
			}
		}
	}

	got, err := s.ReadPreset(ctx, opts.Setlist, opts.Slot)
	if err != nil {
		return fmt.Errorf("reading slot %s: %w", slotpkg.Label(opts.Slot), err)
	}

	if err := dump(got); err != nil {
		return err
	}

	body, ok := got.(string)
	if !ok {
		return describe(w, s.Model().Name, opts.Slot, got)
	}

	return writeDeviceRig(w, []byte(body), opts)
}

// ListDevice prints what an attached device holds.
//
// Read-only: it asks the device to describe a setlist and nothing more.
// Nothing is selected, loaded or written.
func ListDevice(ctx context.Context, w io.Writer, opts DeviceOptions) error {
	s, err := sdk.Open(ctx)
	if err != nil {
		return err
	}

	defer s.Close()

	presets, err := s.Presets(ctx, opts.Setlist)
	if err != nil {
		return fmt.Errorf("listing presets: %w", err)
	}

	rows := make([][]string, 0, len(presets))

	var used int

	for _, p := range presets {
		// A device names an untouched slot rather than leaving it blank, so
		// what counts as empty is the name it was shipped with.
		if p.Name == "New Preset" {
			if !opts.All {
				continue
			}

			rows = append(rows, []string{
				cli.Mute(w, p.Label()), cli.Mute(w, p.Name),
			})

			continue
		}

		used++

		rows = append(rows, []string{cli.Accent(w, p.Label()), p.Name})
	}

	return cli.Section{
		Title:  s.Model().Name,
		Detail: fmt.Sprintf("%s · %d in use", plural(len(presets), "slot"), used),
		// One address, the one printed on the pedal. What the device counts
		// underneath is its business, and --slot takes what is shown here.
		Headers: []string{"slot", "name"},
		Rows:    rows,
		Empty:   "no presets",
	}.Render(w)
}
