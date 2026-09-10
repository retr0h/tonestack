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

	"github.com/retr0h/tonestack/pkg/preset"
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

	cat, err := opts.catalogs().Open(opts.CatalogPath)
	if err != nil {
		return err
	}

	name := opts.Name
	if name == "" {
		name = "slot " + slotpkg.Label(opts.Slot)
	}

	doc, empty, err := opts.translator().Document(got, cat, name)
	if err != nil {
		return err
	}

	if empty {
		_, err := fmt.Fprintf(w, "# %s is empty\n", name)

		return err
	}

	// The device's own file, when that is what was asked for. A rig is the
	// default because it reads on other hardware; this is the faithful copy.
	if opts.As == FormatPreset {
		return preset.Write(w, doc)
	}

	spec, err := opts.compiler().Lift(doc, cat)
	if err != nil {
		return fmt.Errorf("reading slot %s: %w", slotpkg.Label(opts.Slot), err)
	}

	// Everything else in that section would be the untouched preset this was
	// assembled into rather than the slot it describes, so only what the
	// device actually said is kept.
	//
	// Controller assignments are not decoded yet and so are not carried. A
	// rig read off the device rebuilds its routing but not those.
	spec.Device = opts.translator().DeviceState(got, cat)
	spec.Snapshots = opts.translator().Snapshots(got)
	spec.Footswitches = opts.translator().Footswitches(got, cat)
	spec.Controllers = opts.translator().Controllers(got, cat)

	return writeRigTo(w, spec)
}
