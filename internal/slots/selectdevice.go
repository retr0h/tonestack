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

// SelectDevice makes one preset the active one on an attached device.
func SelectDevice(ctx context.Context, w io.Writer, opts DeviceOptions) error {
	s, err := openDevice(ctx)
	if err != nil {
		return err
	}

	defer s.Close()

	return SelectWith(ctx, w, s, opts)
}

// SelectWith makes one preset the active one on the given session.
//
// The device loads it and starts making that sound. Nothing is written: a
// preset is loaded into the edit buffer and the slot it came from is
// untouched, so this is the one device operation that changes what you hear
// without changing what the device holds.
func SelectWith(
	ctx context.Context,
	w io.Writer,
	s sdk.Editor,
	opts DeviceOptions,
) error {
	sel, ok := s.(sdk.Selector)
	if !ok {
		return fmt.Errorf("this session cannot select a preset")
	}

	// Read before selecting, so the name is the one being switched to rather
	// than whatever the device answers with mid-switch.
	found, err := s.Presets(ctx, opts.Setlist)
	if err != nil {
		return fmt.Errorf("listing setlist %d: %w", opts.Setlist, err)
	}

	if err := sel.SelectPreset(ctx, opts.Setlist, opts.Slot); err != nil {
		return fmt.Errorf("selecting slot %s: %w", slotpkg.Label(opts.Slot), err)
	}

	_, err = fmt.Fprintf(w, "\n%s%s %s\n\n%s%s\n\n",
		cli.Indent,
		cli.Accent(w, slotpkg.Label(opts.Slot)),
		nameOf(found, opts.Slot),
		cli.Indent, cli.Success(w, "loaded"))

	return err
}
