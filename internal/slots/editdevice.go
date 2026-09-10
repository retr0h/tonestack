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

// CopyDevice puts what one slot holds into another, on an attached device.
//
// The preset is moved exactly as the device wrote it. Nothing is decoded and
// nothing is rebuilt, which is what makes this the safest thing to write: a
// device seeks through a preset by a table of byte offsets, and the surest way
// to keep those right is to change nothing.
//
// The destination is overwritten. There is no undo on a device.
func CopyDevice(ctx context.Context, w io.Writer, opts EditOptions) error {
	return editDevice(ctx, w, opts, "copied", copyOne)
}

// SwapDevice exchanges what two slots hold.
func SwapDevice(ctx context.Context, w io.Writer, opts EditOptions) error {
	return editDevice(ctx, w, opts, "swapped", swapTwo)
}

// editDevice opens a session and hands it to one of the two above.
func editDevice(
	ctx context.Context,
	w io.Writer,
	opts EditOptions,
	verb string,
	apply func(context.Context, sdk.Editor, EditOptions) (string, string, []string, error),
) error {
	s, err := openDevice(ctx)
	if err != nil {
		return err
	}

	defer s.Close()

	return editWith(ctx, w, s, opts, verb, apply)
}

// CopyWith puts what one slot holds into another, on the given session.
func CopyWith(ctx context.Context, w io.Writer, s sdk.Editor, opts EditOptions) error {
	return editWith(ctx, w, s, opts, "copied", copyOne)
}

// SwapWith exchanges what two slots hold, on the given session.
func SwapWith(ctx context.Context, w io.Writer, s sdk.Editor, opts EditOptions) error {
	return editWith(ctx, w, s, opts, "swapped", swapTwo)
}

// editWith performs one edit against the given session.
func editWith(
	ctx context.Context,
	w io.Writer,
	s sdk.Editor,
	opts EditOptions,
	verb string,
	apply func(context.Context, sdk.Editor, EditOptions) (string, string, []string, error),
) error {
	fromName, toName, kept, err := apply(ctx, s, opts)
	if err != nil {
		return err
	}

	if err := said(w, kept...); err != nil {
		return err
	}

	_, err = fmt.Fprintf(w, "\n%s%s %s %s %s\n\n%s%s\n\n",
		cli.Indent,
		cli.Accent(w, slotpkg.Label(opts.FromSlot)), fromName,
		cli.Mute(w, "→"),
		cli.Accent(w, slotpkg.Label(opts.ToSlot)),
		cli.Indent, cli.Success(w, verb+", replacing "+toName))

	return err
}

// copyOne writes what the source holds into the destination.
func copyOne(
	ctx context.Context,
	s sdk.Editor,
	opts EditOptions,
) (string, string, []string, error) {
	from, to, err := names(ctx, s, opts)
	if err != nil {
		return "", "", nil, err
	}

	body, err := slotBytes(ctx, s, opts.FromSetlist, opts.FromSlot)
	if err != nil {
		return "", "", nil, err
	}

	// Before the backup: a session that cannot write is not going to
	// replace anything, so reading the destination to keep it would be a
	// round trip to the device for nothing.
	w, err := writerFor(s)
	if err != nil {
		return "", "", nil, err
	}

	// The destination is about to stop being what it was, and unlike the
	// source nobody has read it yet.
	replaced, err := holds(ctx, s, opts.ToSetlist, opts.ToSlot)
	if err != nil {
		return "", "", nil, err
	}

	kept, err := keep(replaced, opts, opts.ToSlot)
	if err != nil {
		return "", "", nil, err
	}

	// Named, because the destination takes the source's name along with its
	// contents. Writing without one would leave the slot called whatever it
	// was, which is not what copying a preset means.
	if err := w.WriteNamedPreset(
		ctx, opts.ToSetlist, opts.ToSlot, from, body); err != nil {
		return "", "", nil, fmt.Errorf("writing slot %s: %w",
			slotpkg.Label(opts.ToSlot), err)
	}

	return from, to, kept, nil
}

// swapTwo exchanges what two slots hold.
//
// Both are read before either is written. A device that fails halfway through
// would otherwise leave one slot holding a copy of the other and the original
// gone.
func swapTwo(
	ctx context.Context,
	s sdk.Editor,
	opts EditOptions,
) (string, string, []string, error) {
	from, to, err := names(ctx, s, opts)
	if err != nil {
		return "", "", nil, err
	}

	source, err := slotBytes(ctx, s, opts.FromSetlist, opts.FromSlot)
	if err != nil {
		return "", "", nil, err
	}

	destination, err := slotBytes(ctx, s, opts.ToSetlist, opts.ToSlot)
	if err != nil {
		return "", "", nil, err
	}

	w, err := writerFor(s)
	if err != nil {
		return "", "", nil, err
	}

	// Both of them, because a swap replaces both. No extra reads: a swap has
	// already read what it is about to move.
	kept, err := keep(destination, opts, opts.ToSlot)
	if err != nil {
		return "", "", nil, err
	}

	also, err := keep(source, opts, opts.FromSlot)
	if err != nil {
		return "", "", nil, err
	}

	kept = append(kept, also...)

	if err := w.WriteNamedPreset(
		ctx, opts.ToSetlist, opts.ToSlot, from, source); err != nil {
		return "", "", nil, fmt.Errorf("writing slot %s: %w",
			slotpkg.Label(opts.ToSlot), err)
	}

	if err := w.WriteNamedPreset(
		ctx, opts.FromSetlist, opts.FromSlot, to, destination); err != nil {
		return "", "", nil, fmt.Errorf("writing slot %s: %w",
			slotpkg.Label(opts.FromSlot), err)
	}

	return from, to, kept, nil
}

// slotBytes reads one slot as the bytes the device holds.
func slotBytes(
	ctx context.Context,
	s sdk.Editor,
	setlist, slot int,
) ([]byte, error) {
	body, err := s.ReadPreset(ctx, setlist, slot)
	if err != nil {
		return nil, fmt.Errorf("reading slot %s: %w", slotpkg.Label(slot), err)
	}

	if body == nil {
		return nil, fmt.Errorf("slot %s did not answer with a preset",
			slotpkg.Label(slot))
	}

	return body, nil
}

// names reads what the device calls both slots, before either is changed.
func names(ctx context.Context, s sdk.Editor, opts EditOptions) (string, string, error) {
	found, err := s.Presets(ctx, opts.FromSetlist)
	if err != nil {
		return "", "", fmt.Errorf("listing presets: %w", err)
	}

	return nameOf(found, opts.FromSlot), nameOf(found, opts.ToSlot), nil
}

// writerFor asks whether this session can write.
//
// Reading and writing are separate abilities because writing is the half that
// can destroy somebody's work. A session that only reads says so here rather
// than partway through an edit.
func writerFor(s sdk.Editor) (sdk.Writer, error) {
	w, ok := s.(sdk.Writer)
	if !ok {
		return nil, fmt.Errorf("this session cannot write to a device")
	}

	return w, nil
}
