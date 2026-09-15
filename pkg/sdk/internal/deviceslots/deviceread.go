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

package deviceslots

import (
	"context"
	"fmt"

	"github.com/retr0h/tonestack/pkg/sdk/internal/wire"
	"github.com/retr0h/tonestack/pkg/sdk/result"
	slotpkg "github.com/retr0h/tonestack/pkg/sdk/slot"
)

// deviceReading turns a device's answer for one slot into a reading.
//
// The same rig a backup would produce, because the device and a file describe
// the same preset. What arrives here names nothing: a model is a number and
// parameters are a bare array, so the catalog's model table is what makes it
// readable.
//
// name is what the device calls the slot, empty when nothing said. as is the
// format the reading is for: the device's own file needs no rig lifted out of
// it.
func (f *Flows) deviceReading(
	ctx context.Context,
	body []byte,
	slot int,
	name string,
	as result.Format,
) (result.Reading, error) {
	got, err := wire.DecodePreset(body)
	if err != nil {
		return result.Reading{}, fmt.Errorf(
			"reading slot %s: %w", slotpkg.Label(slot), err)
	}

	cat, err := f.catalog(ctx)
	if err != nil {
		return result.Reading{}, err
	}

	if name == "" {
		name = "slot " + slotpkg.Label(slot)
	}

	doc, empty, err := f.translator().Document(got, cat, name)
	if err != nil {
		return result.Reading{}, err
	}

	if empty {
		return result.Reading{Name: name}, nil
	}

	// Only the device's own file was asked for, so the lift is work nobody
	// wants. A rig is the default because it reads on other hardware; this is
	// the faithful copy.
	if as == result.FormatPreset {
		return result.Reading{Name: name, Doc: doc}, nil
	}

	spec, err := f.compiler().Lift(doc, cat)
	if err != nil {
		return result.Reading{}, fmt.Errorf(
			"reading slot %s: %w", slotpkg.Label(slot), err)
	}

	// Everything else in that section would be the untouched preset this was
	// assembled into rather than the slot it describes, so only what the
	// device actually said is kept.
	//
	// Controller assignments are not decoded yet and so are not carried. A
	// rig read off the device rebuilds its routing but not those.
	spec.Device = f.translator().DeviceState(got, cat)
	spec.Snapshots = f.translator().Snapshots(got)
	spec.Footswitches = f.translator().Footswitches(got, cat)
	spec.Controllers = f.translator().Controllers(got, cat)

	return result.Reading{Name: name, Doc: doc, Rig: spec}, nil
}
