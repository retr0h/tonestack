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

package setlist

import (
	"encoding/json"
	"maps"

	"github.com/retr0h/tonestack/pkg/preset"
)

// Copy overwrites one slot with another.
//
// Whatever the destination held is gone. The source is untouched.
func (d *Document) Copy(from, to Address) error {
	src, err := d.Slot(from.Setlist, from.Slot)
	if err != nil {
		return err
	}

	dst, err := d.Slot(to.Setlist, to.Slot)
	if err != nil {
		return err
	}

	// Deep, because a preset.Data is mostly maps: a struct copy would leave
	// the two slots sharing their tone and their metadata, so the next edit
	// to either would rewrite both.
	//
	// The raw messages inside are shared, and that is safe — nothing in this
	// package writes into one, they are replaced whole.
	out := *src
	out.Meta.Rest = maps.Clone(src.Meta.Rest)
	out.Tone = make(map[string]preset.Tone, len(src.Tone))

	for key, entries := range src.Tone {
		out.Tone[key] = maps.Clone(entries)
	}

	*dst = out

	return nil
}

// Swap exchanges two slots.
//
// This is what moving a preset means here. Blanking the source instead would
// mean writing an empty preset, and an empty preset is not empty: it carries
// the inputs, outputs, split and join a device expects, which differ by model
// and by firmware. Swapping invents nothing and can be undone by repeating it.
func (d *Document) Swap(a, b Address) error {
	x, err := d.Slot(a.Setlist, a.Slot)
	if err != nil {
		return err
	}

	y, err := d.Slot(b.Setlist, b.Slot)
	if err != nil {
		return err
	}

	*x, *y = *y, *x

	return nil
}

// Rename sets the name a player sees for a slot.
func (d *Document) Rename(at Address, name string) error {
	s, err := d.Slot(at.Setlist, at.Slot)
	if err != nil {
		return err
	}

	s.Meta.Name = name

	return nil
}

// Name returns the setlist's name.
//
// Setlist metadata is held as raw JSON so nothing this package has no opinion
// about is dropped on write. The name is the one field worth reaching for, so
// it is read on demand rather than modelled.
func (s Setlist) Name() string {
	var meta struct {
		Name string `json:"name"`
	}

	_ = json.Unmarshal(s.Meta, &meta)

	return meta.Name
}

// Names returns the name of every slot in a setlist, in slot order.
//
// Empty slots keep whatever the device named them, so the result is as long
// as the setlist and positions line up with what the hardware shows.
func (d *Document) Names(setlist int) ([]string, error) {
	if setlist < 0 || setlist >= len(d.Setlists) {
		return nil, &NoSuchSlotError{Setlist: setlist, Have: 0}
	}

	slots := d.Setlists[setlist].Slots
	out := make([]string, 0, len(slots))

	for i := range slots {
		out = append(out, slots[i].Meta.Name)
	}

	return out, nil
}
