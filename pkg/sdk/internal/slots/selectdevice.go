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

	"github.com/retr0h/tonestack/pkg/sdk/device"
	"github.com/retr0h/tonestack/pkg/sdk/result"
	slotpkg "github.com/retr0h/tonestack/pkg/sdk/slot"
)

// SelectDevice makes one preset the active one on an attached device.
func SelectDevice(ctx context.Context, opts DeviceOptions) (result.Change, error) {
	s, err := OpenDevice(ctx)
	if err != nil {
		return result.Change{}, err
	}

	defer s.Close()

	return SelectWith(ctx, s, opts)
}

// SelectWith makes one preset the active one on the given session.
//
// The device loads it and starts making that sound. Nothing is written: a
// preset is loaded into the edit buffer and the slot it came from is
// untouched, so this is the one device operation that changes what you hear
// without changing what the device holds.
func SelectWith(
	ctx context.Context,
	s device.Editor,
	opts DeviceOptions,
) (result.Change, error) {
	sel, ok := s.(device.Selector)
	if !ok {
		return result.Change{}, fmt.Errorf("this session cannot select a preset")
	}

	// Read before selecting, so the name is the one being switched to rather
	// than whatever the device answers with mid-switch.
	found, err := s.Presets(ctx, opts.Setlist)
	if err != nil {
		return result.Change{}, fmt.Errorf(
			"listing setlist %d: %w", opts.Setlist, err)
	}

	if err := sel.SelectPreset(ctx, opts.Setlist, opts.Slot); err != nil {
		return result.Change{}, fmt.Errorf(
			"selecting slot %s: %w", slotpkg.Label(opts.Slot), err)
	}

	return result.Change{
		Action: result.Selected,
		To:     result.At{Slot: opts.Slot, Name: nameOf(found, opts.Slot)},
	}, nil
}
