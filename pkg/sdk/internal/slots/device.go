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
	"errors"
	"fmt"

	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/chain"
	"github.com/retr0h/tonestack/pkg/sdk/internal/device"
	"github.com/retr0h/tonestack/pkg/sdk/internal/wire"
	"github.com/retr0h/tonestack/pkg/sdk/result"
	slotpkg "github.com/retr0h/tonestack/pkg/sdk/slot"
)

// ShowWith reads one slot off the given session, as a rig.
//
// Read-only: the device hands back the preset and goes on playing whatever it
// was. Nothing is selected, loaded or written. Taking the session makes
// reading a device testable without one attached.
func (f *Flows) ShowWith(
	ctx context.Context,
	s device.Editor,
	at slotpkg.Address,
) (result.Reading, error) {
	return f.read(ctx, s, at, result.FormatRig)
}

// ExportWith writes one slot off the given session to a file.
//
// The same rig ShowWith reads, which is the point: a slot read off the
// hardware and one read out of a backup are the same document.
func (f *Flows) ExportWith(
	ctx context.Context,
	s device.Editor,
	at slotpkg.Address,
	out string,
	as result.Format,
) (result.Written, error) {
	// Before the device is asked anything.
	if err := known(as); err != nil {
		return result.Written{}, err
	}

	read, err := f.read(ctx, s, at, as)
	if err != nil {
		return result.Written{}, err
	}

	return write(read, at.Slot, out, as)
}

// read reads one slot off the given session in the format asked for.
func (f *Flows) read(
	ctx context.Context,
	s device.Editor,
	at slotpkg.Address,
	as result.Format,
) (result.Reading, error) {
	// The name comes from the listing rather than the preset: what the device
	// hands back for one slot does not carry it. A listing that fails costs
	// only the name, and the slot is named by where it sits instead.
	name := ""
	if found, err := s.Presets(ctx, at.Setlist); err == nil {
		name = nameOf(found, at.Slot)
	}

	body, err := s.ReadPreset(ctx, at.Setlist, at.Slot)

	// A device that answered with something else is not a failure to report
	// as one: what arrived is worth keeping and reporting, because it is how
	// a protocol change becomes visible.
	var answer *device.NotAPresetError
	if errors.As(err, &answer) {
		if err := dump(f.Capture, answer.Result); err != nil {
			return result.Reading{}, err
		}

		return result.Reading{
			Name: name,
			Answer: &result.Answer{
				Model: s.Model().Name,
				Slot:  at.Slot,
				Shape: answer.Shape(),
			},
		}, nil
	}

	if err != nil {
		return result.Reading{}, fmt.Errorf(
			"reading slot %s: %w", slotpkg.Label(at.Slot), err)
	}

	if err := dump(f.Capture, body); err != nil {
		return result.Reading{}, err
	}

	// A device answers an empty slot with no document at all. That is a slot
	// holding nothing rather than a failure, and a backup has to know the
	// difference to put a pedal back the way it was found.
	if body == nil {
		return result.Reading{}, fmt.Errorf(
			"%w: %s", ErrEmptySlot, slotpkg.Label(at.Slot))
	}

	return f.deviceReading(ctx, body, at.Slot, name, as)
}

// ErrEmptySlot is returned for a slot holding no preset.
//
// Reported rather than written out: an export that answered an empty slot
// with a file would leave somebody a preset that is not one.
var ErrEmptySlot = errors.New("the slot holds no preset")

// nameOf finds what a listing calls one slot.
//
// Searched rather than indexed. A device answers with every slot in order, so
// the two are the same today, and a listing that ever skipped an empty slot
// would silently name every preset after it wrongly.
func nameOf(
	found []wire.Preset,
	slot int,
) string {
	for _, p := range found {
		if p.Slot == slot {
			return p.Name
		}
	}

	return ""
}

// ListWith answers with what one setlist on the given session holds.
//
// Read-only: it asks the device to describe a setlist and nothing more.
// Nothing is selected, loaded or written.
func (f *Flows) ListWith(
	ctx context.Context,
	s device.Editor,
	setlist int,
) (result.Listing, error) {
	presets, err := s.Presets(ctx, setlist)
	if err != nil {
		return result.Listing{}, fmt.Errorf("listing presets: %w", err)
	}

	cat, err := f.catalog(ctx)
	if err != nil {
		return result.Listing{}, err
	}

	held := make([]result.Held, 0, len(presets))

	for _, p := range presets {
		one := result.Held{Slot: p.Slot, Name: p.Name}

		// A slot is always named, so a name says nothing about whether
		// anything is in it. An untouched one keeps the name it shipped
		// with, and only reading it says which.
		if p.Name != untouched {
			blocks, err := f.chainAt(ctx, s, cat, slotpkg.Address{Setlist: setlist, Slot: p.Slot})
			if err != nil {
				return result.Listing{}, err
			}

			one.Blocks = blocks
		}

		held = append(held, one)
	}

	return result.Listing{Name: s.Model().Name, Slots: held}, nil
}

// untouched is what a device calls a slot nobody has named.
const untouched = "New Preset"

// chainAt reads one slot and returns the blocks it holds.
//
// A slot that will not decode is reported as holding nothing rather than
// failing the listing around it: one unreadable preset should not hide the
// hundred that read.
func (f *Flows) chainAt(
	ctx context.Context,
	s device.Editor,
	cat *catalog.Catalog,
	at slotpkg.Address,
) ([]chain.Block, error) {
	body, err := s.ReadPreset(ctx, at.Setlist, at.Slot)

	// An answer that is not a preset is skipped the way an undecodable one
	// is: this is a listing, and one slot nobody can read should not hide the
	// hundred that read.
	if errors.Is(err, device.ErrNotAPreset) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("reading slot %s: %w", slotpkg.Label(at.Slot), err)
	}

	if body == nil {
		return nil, nil
	}

	preset, err := wire.DecodePreset(body)
	if err != nil {
		return nil, nil
	}

	c, err := f.translator().Chain("", preset, cat)
	if err != nil {
		return nil, nil
	}

	return c.Blocks, nil
}
