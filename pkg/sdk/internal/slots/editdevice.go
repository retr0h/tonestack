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

// CopyDevice puts what one slot holds into another, on an attached device.
//
// The preset is moved exactly as the device wrote it. Nothing is decoded and
// nothing is rebuilt, which is what makes this the safest thing to write: a
// device seeks through a preset by a table of byte offsets, and the surest way
// to keep those right is to change nothing.
//
// The destination is overwritten. There is no undo on a device.
func CopyDevice(ctx context.Context, opts EditOptions) (result.Change, error) {
	return editDevice(ctx, opts, result.Copied, copyOne)
}

// SwapDevice exchanges what two slots hold.
func SwapDevice(ctx context.Context, opts EditOptions) (result.Change, error) {
	return editDevice(ctx, opts, result.Swapped, swapTwo)
}

// editDevice opens a session and hands it to one of the two above.
func editDevice(
	ctx context.Context,
	opts EditOptions,
	action result.Action,
	apply applier,
) (result.Change, error) {
	s, err := OpenDevice(ctx)
	if err != nil {
		return result.Change{}, err
	}

	defer s.Close()

	return editWith(ctx, s, opts, action, apply)
}

// applier performs one edit against a session and says what it did.
type applier func(context.Context, device.Editor, EditOptions) (edited, error)

// edited is what an edit found and kept on the way past.
type edited struct {
	// from and to are what the two slots were called before the write.
	from, to string
	// kept are the backups written before anything was overwritten.
	kept []string
}

// CopyWith puts what one slot holds into another, on the given session.
func CopyWith(
	ctx context.Context,
	s device.Editor,
	opts EditOptions,
) (result.Change, error) {
	return editWith(ctx, s, opts, result.Copied, copyOne)
}

// SwapWith exchanges what two slots hold, on the given session.
func SwapWith(
	ctx context.Context,
	s device.Editor,
	opts EditOptions,
) (result.Change, error) {
	return editWith(ctx, s, opts, result.Swapped, swapTwo)
}

// editWith performs one edit against the given session.
func editWith(
	ctx context.Context,
	s device.Editor,
	opts EditOptions,
	action result.Action,
	apply applier,
) (result.Change, error) {
	did, err := apply(ctx, s, opts)
	if err != nil {
		return result.Change{}, err
	}

	return result.Change{
		Action:   action,
		From:     &result.At{Slot: opts.FromSlot, Name: did.from},
		To:       result.At{Slot: opts.ToSlot, Name: did.to},
		Replaced: did.to,
		Kept:     did.kept,
	}, nil
}

// copyOne writes what the source holds into the destination.
func copyOne(
	ctx context.Context,
	s device.Editor,
	opts EditOptions,
) (edited, error) {
	from, to, err := names(ctx, s, opts)
	if err != nil {
		return edited{}, err
	}

	body, err := slotBytes(ctx, s, opts.FromSetlist, opts.FromSlot)
	if err != nil {
		return edited{}, err
	}

	// Before the backup: a session that cannot write is not going to
	// replace anything, so reading the destination to keep it would be a
	// round trip to the device for nothing.
	w, err := writerFor(s)
	if err != nil {
		return edited{}, err
	}

	// The destination is about to stop being what it was, and unlike the
	// source nobody has read it yet.
	kept, err := replacing(ctx, s, opts.Deps, opts.CatalogPath, opts.BackupDir,
		opts.ToSetlist, opts.ToSlot)
	if err != nil {
		return edited{}, err
	}

	// Named, because the destination takes the source's name along with its
	// contents. Writing without one would leave the slot called whatever it
	// was, which is not what copying a preset means.
	if err := w.WriteNamedPreset(
		ctx, opts.ToSetlist, opts.ToSlot, from, body); err != nil {
		return edited{}, fmt.Errorf("writing slot %s: %w",
			slotpkg.Label(opts.ToSlot), err)
	}

	return edited{from: from, to: to, kept: kept}, nil
}

// swapTwo exchanges what two slots hold.
//
// Both are read before either is written. A device that fails halfway through
// would otherwise leave one slot holding a copy of the other and the original
// gone.
func swapTwo(
	ctx context.Context,
	s device.Editor,
	opts EditOptions,
) (edited, error) {
	from, to, err := names(ctx, s, opts)
	if err != nil {
		return edited{}, err
	}

	source, err := slotBytes(ctx, s, opts.FromSetlist, opts.FromSlot)
	if err != nil {
		return edited{}, err
	}

	destination, err := slotBytes(ctx, s, opts.ToSetlist, opts.ToSlot)
	if err != nil {
		return edited{}, err
	}

	w, err := writerFor(s)
	if err != nil {
		return edited{}, err
	}

	// Both of them, because a swap replaces both. No extra reads: a swap has
	// already read what it is about to move.
	kept, err := keep(opts.Deps, opts.CatalogPath, opts.BackupDir,
		at{body: destination, slot: opts.ToSlot},
		at{body: source, slot: opts.FromSlot})
	if err != nil {
		return edited{}, err
	}

	if err := w.WriteNamedPreset(
		ctx, opts.ToSetlist, opts.ToSlot, from, source); err != nil {
		return edited{}, fmt.Errorf("writing slot %s: %w",
			slotpkg.Label(opts.ToSlot), err)
	}

	if err := w.WriteNamedPreset(
		ctx, opts.FromSetlist, opts.FromSlot, to, destination); err != nil {
		return edited{}, fmt.Errorf("writing slot %s: %w",
			slotpkg.Label(opts.FromSlot), err)
	}

	return edited{from: from, to: to, kept: kept}, nil
}

// slotBytes reads one slot as the bytes the device holds.
func slotBytes(
	ctx context.Context,
	s device.Editor,
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
func names(ctx context.Context, s device.Editor, opts EditOptions) (string, string, error) {
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
func writerFor(s device.Editor) (device.Writer, error) {
	w, ok := s.(device.Writer)
	if !ok {
		return nil, fmt.Errorf("this session cannot write to a device")
	}

	return w, nil
}
