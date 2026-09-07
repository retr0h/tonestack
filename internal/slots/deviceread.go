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
	"fmt"
	"io"

	"github.com/retr0h/tonestack/internal/catalogview"
	"github.com/retr0h/tonestack/internal/lift"
	"github.com/retr0h/tonestack/pkg/catalog"
	"github.com/retr0h/tonestack/pkg/preset"
	riggen "github.com/retr0h/tonestack/pkg/rig/gen"
	"github.com/retr0h/tonestack/pkg/sdk/wire"
	slotpkg "github.com/retr0h/tonestack/pkg/slot"
)

// writeDeviceRig turns a device's answer into a rig and writes it.
//
// The same rig a backup would produce, because the device and a file describe
// the same preset. What arrives here names nothing — a model is a number and
// parameters are a bare array — so the catalog's model table is what makes it
// readable.
func writeDeviceRig(w io.Writer, body []byte, opts DeviceOptions) error {
	got, err := wire.DecodePreset(body)
	if err != nil {
		return fmt.Errorf("reading slot %s: %w", slotpkg.Label(opts.Slot), err)
	}

	cat, err := catalogview.Open(opts.CatalogPath)
	if err != nil {
		return err
	}

	name := opts.Name
	if name == "" {
		name = "slot " + slotpkg.Label(opts.Slot)
	}

	c, err := chainOf(name, got, cat)
	if err != nil {
		return err
	}

	if len(c.Blocks) == 0 {
		_, err := fmt.Fprintf(w, "# %s is empty\n", name)

		return err
	}

	// An untouched preset the device itself wrote, which cannot fail to
	// decode: it is embedded in this binary and a test reads it. The chain
	// goes into it, and what comes out is a rig.
	doc, _ := preset.Blank()
	doc.Data.Device = cat.DeviceID
	doc.Data.Meta.Name = name

	_ = doc.SetSpec(c)

	spec, err := lift.Lift(doc, cat)
	if err != nil {
		return fmt.Errorf("reading slot %s: %w", slotpkg.Label(opts.Slot), err)
	}

	// The chain is the device's; the rest of that section would be the
	// untouched preset it was assembled into. Routing and controller
	// assignments arrive over USB in a numbering nobody has decoded yet, and
	// writing the template's in their place would put a stranger's settings
	// in a file that claims to describe this slot.
	//
	// Absent means absent: compiling this rig builds it into an untouched
	// preset, the same as one somebody typed. Reading the same slot out of a
	// backup carries the real thing — see docs/protocol.md.
	spec.Device = nil
	spec.Snapshots = snapshotsOf(got)
	spec.Footswitches = footswitchesOf(got, cat)

	return writeRigTo(w, spec)
}

// snapshotsOf carries what the device recalls on a footswitch.
func snapshotsOf(got wire.DevicePreset) *[]riggen.Snapshot {
	if len(got.Snapshots) == 0 {
		return nil
	}

	out := make([]riggen.Snapshot, 0, len(got.Snapshots))

	for _, s := range got.Snapshots {
		snap := riggen.Snapshot{}

		if s.Name != "" {
			name := s.Name
			snap.Name = &name
		}

		if s.Tempo > 0 {
			tempo := s.Tempo
			snap.Tempo = &tempo
		}

		led, valid := s.LED, s.Valid
		snap.Led, snap.Valid = &led, &valid

		out = append(out, snap)
	}

	return &out
}

// footswitchesOf carries what the pedal prints under each switch.
func footswitchesOf(got wire.DevicePreset, cat *catalog.Catalog) *[]riggen.Footswitch {
	if len(got.Footswitches) == 0 {
		return nil
	}

	out := make([]riggen.Footswitch, 0, len(got.Footswitches))

	for _, f := range got.Footswitches {
		label, gear, at, block := f.Label, f.Gear, f.Switch, f.Block
		fs := riggen.Footswitch{
			Switch: &at, Label: &label, Gear: &gear, Block: &block,
		}

		// A colour a device knows and this catalog does not gets no name
		// rather than a wrong one.
		if name, ok := cat.LEDColour(f.LED); ok {
			fs.Led = &name
		}

		out = append(out, fs)
	}

	return &out
}
