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
	"strings"

	"github.com/retr0h/tonestack/pkg/sdk/internal/backup"
	"github.com/retr0h/tonestack/pkg/sdk/internal/device"
	"github.com/retr0h/tonestack/pkg/sdk/internal/fileslots"
	"github.com/retr0h/tonestack/pkg/sdk/result"
	slotpkg "github.com/retr0h/tonestack/pkg/sdk/slot"
)

// ErrSameSlot is a copy or a swap whose two slots are one slot.
//
// The file flows' error, because they refuse it too and one behaviour is what
// a caller wants. It is declared there because this package imports that one.
var ErrSameSlot = fileslots.ErrSameSlot

// EmptySwapError is a swap refused because a slot it names holds no preset.
//
// A swap with an empty slot would be a move, and a move has to leave the
// source empty again. A device answers an empty slot with no document at all,
// and nothing here can write that: the only empty preset this project holds,
// wire.Blank, is a document with no blocks off another slot and another
// firmware, so writing it would leave a slot the device never produced. The
// swap is refused before anything is kept or written.
type EmptySwapError struct {
	// Empty are the slots that hold no preset, in the order the swap named
	// them.
	Empty []slotpkg.Address
	// Full is the slot that holds one, or nil when neither does.
	Full *slotpkg.Address
}

// Error implements the error interface.
func (e *EmptySwapError) Error() string {
	labels := make([]string, 0, len(e.Empty))
	for _, at := range e.Empty {
		labels = append(labels, slotpkg.Label(at.Slot))
	}

	if e.Full == nil {
		return fmt.Sprintf("%v: %s, so there is nothing to swap",
			ErrEmptySlot, strings.Join(labels, " and "))
	}

	return fmt.Sprintf("%v: %s, and a swap would have to leave %s empty, "+
		"which nothing here can write", ErrEmptySlot, labels[0],
		slotpkg.Label(e.Full.Slot))
}

// Unwrap returns ErrEmptySlot, so callers can match it.
func (*EmptySwapError) Unwrap() error { return ErrEmptySlot }

// KeptError is a write to a device that failed after what a slot held was
// backed up.
//
// A slot may already be overwritten by then, so its backup is the only copy
// left. A result is not returned alongside an error, so the error has to say
// where the backup is.
type KeptError struct {
	// Kept are the backups written before the write failed.
	Kept []string
	// Err is what went wrong.
	Err error
}

// Error implements the error interface.
func (e *KeptError) Error() string {
	held := "the slot held is"
	if len(e.Kept) > 1 {
		held = "the slots held are"
	}

	return fmt.Sprintf("%v; what %s kept in %s",
		e.Err, held, strings.Join(e.Kept, ", "))
}

// Unwrap returns what went wrong, so callers can still match it.
func (e *KeptError) Unwrap() error { return e.Err }

// keptError names the backups in err, when there are any.
func keptError(
	err error,
	kept []string,
) error {
	if len(kept) == 0 {
		return err
	}

	return &KeptError{Kept: kept, Err: err}
}

// applier performs one edit against a session and says what it did.
type applier func(context.Context, device.Editor, slotpkg.Address, slotpkg.Address) (edited, error)

// edited is what an edit found and kept on the way past.
type edited struct {
	// from and to are what the two slots were called before the write.
	from, to string
	// kept are the backups written before anything was overwritten.
	kept []string
}

// Copy puts what one slot holds into another, on the given session.
//
// The preset is moved exactly as the device wrote it. Nothing is decoded and
// nothing is rebuilt, which is what makes this the safest thing to write: a
// device seeks through a preset by a table of byte offsets, and the surest way
// to keep those right is to change nothing.
//
// The destination is overwritten and kept first. There is no undo on a device.
func (f *Flows) Copy(
	ctx context.Context,
	s device.Editor,
	from, to slotpkg.Address,
) (result.Change, error) {
	return edit(ctx, s, from, to, result.Copied, f.copyOne)
}

// Swap exchanges what two slots hold, on the given session. Both are kept
// first.
func (f *Flows) Swap(
	ctx context.Context,
	s device.Editor,
	a, b slotpkg.Address,
) (result.Change, error) {
	return edit(ctx, s, a, b, result.Swapped, f.swapTwo)
}

// edit performs one edit against the given session.
func edit(
	ctx context.Context,
	s device.Editor,
	from, to slotpkg.Address,
	action result.Action,
	apply applier,
) (result.Change, error) {
	// Before anything is read or kept: moving a slot onto itself changes
	// nothing, and is almost certainly not what was meant.
	if from == to {
		return result.Change{}, fmt.Errorf("%w: %s",
			ErrSameSlot, slotpkg.Label(from.Slot))
	}

	did, err := apply(ctx, s, from, to)
	if err != nil {
		return result.Change{}, err
	}

	return result.Change{
		Action:   action,
		From:     &result.At{Slot: from.Slot, Name: did.from},
		To:       result.At{Slot: to.Slot, Name: did.to},
		Replaced: did.to,
		Kept:     did.kept,
	}, nil
}

// copyOne writes what the source holds into the destination.
func (f *Flows) copyOne(
	ctx context.Context,
	s device.Editor,
	from, to slotpkg.Address,
) (edited, error) {
	fromName, toName, err := names(ctx, s, from, to)
	if err != nil {
		return edited{}, err
	}

	body, err := slotBytes(ctx, s, from)
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
	kept, err := f.replacing(ctx, s, to, toName)
	if err != nil {
		return edited{}, err
	}

	// Named, because the destination takes the source's name along with its
	// contents. Writing without one would leave the slot called whatever it
	// was, which is not what copying a preset means.
	if err := w.WriteNamedPreset(ctx, to.Setlist, to.Slot, fromName, body); err != nil {
		return edited{}, keptError(fmt.Errorf("writing slot %s: %w",
			slotpkg.Label(to.Slot), err), kept)
	}

	return edited{from: fromName, to: toName, kept: kept}, nil
}

// swapTwo exchanges what two slots hold.
//
// Both are read before either is written. A device that fails halfway through
// would otherwise leave one slot holding a copy of the other and the original
// gone.
func (f *Flows) swapTwo(
	ctx context.Context,
	s device.Editor,
	from, to slotpkg.Address,
) (edited, error) {
	fromName, toName, err := names(ctx, s, from, to)
	if err != nil {
		return edited{}, err
	}

	source, err := readSlot(ctx, s, from)
	if err != nil {
		return edited{}, err
	}

	destination, err := readSlot(ctx, s, to)
	if err != nil {
		return edited{}, err
	}

	// Both read first, so the refusal can name every empty side, and before
	// anything is kept or written, so a refused swap changes nothing.
	if err := emptySides(from, source, to, destination); err != nil {
		return edited{}, err
	}

	w, err := writerFor(s)
	if err != nil {
		return edited{}, err
	}

	// Both of them, because a swap replaces both. No extra reads: a swap has
	// already read what it is about to move.
	kept, err := f.backups().Keep(ctx,
		backup.Held{Body: destination, At: to, Name: toName},
		backup.Held{Body: source, At: from, Name: fromName})
	if err != nil {
		return edited{}, err
	}

	if err := w.WriteNamedPreset(ctx, to.Setlist, to.Slot, fromName, source); err != nil {
		return edited{}, keptError(fmt.Errorf("writing slot %s: %w",
			slotpkg.Label(to.Slot), err), kept)
	}

	// One slot is written, so the swap finishes whoever stops waiting.
	// Stopping here would leave both slots holding the source, and what the
	// destination held only in a backup.
	if err := w.WriteNamedPreset(context.WithoutCancel(ctx),
		from.Setlist, from.Slot, toName, destination); err != nil {
		return edited{}, keptError(fmt.Errorf("writing slot %s: %w",
			slotpkg.Label(from.Slot), err), kept)
	}

	return edited{from: fromName, to: toName, kept: kept}, nil
}

// slotBytes reads one slot as the bytes the device holds, refusing a slot
// that holds none.
func slotBytes(
	ctx context.Context,
	s device.Editor,
	at slotpkg.Address,
) ([]byte, error) {
	body, err := readSlot(ctx, s, at)
	if err != nil {
		return nil, err
	}

	if body == nil {
		return nil, fmt.Errorf("%w: %s", ErrEmptySlot, slotpkg.Label(at.Slot))
	}

	return body, nil
}

// readSlot reads one slot as the bytes the device holds, or nothing for an
// empty slot.
func readSlot(
	ctx context.Context,
	s device.Editor,
	at slotpkg.Address,
) ([]byte, error) {
	body, err := s.ReadPreset(ctx, at.Setlist, at.Slot)
	if err != nil {
		return nil, fmt.Errorf("reading slot %s: %w", slotpkg.Label(at.Slot), err)
	}

	return body, nil
}

// emptySides refuses a swap when either slot holds no preset.
//
// See EmptySwapError for why that is a refusal rather than a move.
func emptySides(
	a slotpkg.Address,
	aBody []byte,
	b slotpkg.Address,
	bBody []byte,
) error {
	switch {
	case aBody == nil && bBody == nil:
		return &EmptySwapError{Empty: []slotpkg.Address{a, b}}
	case aBody == nil:
		return &EmptySwapError{Empty: []slotpkg.Address{a}, Full: &b}
	case bBody == nil:
		return &EmptySwapError{Empty: []slotpkg.Address{b}, Full: &a}
	default:
		return nil
	}
}

// names reads what the device calls both slots, before either is changed.
//
// Each from its own setlist. A slot number means nothing without the setlist
// it is in, and a destination looked up in the source's listing would be
// reported, renamed and backed up as another preset.
func names(
	ctx context.Context,
	s device.Editor,
	from, to slotpkg.Address,
) (string, string, error) {
	inFrom, err := s.Presets(ctx, from.Setlist)
	if err != nil {
		return "", "", fmt.Errorf("listing presets: %w", err)
	}

	inTo := inFrom

	if to.Setlist != from.Setlist {
		inTo, err = s.Presets(ctx, to.Setlist)
		if err != nil {
			return "", "", fmt.Errorf("listing presets: %w", err)
		}
	}

	return nameOf(inFrom, from.Slot), nameOf(inTo, to.Slot), nil
}

// writerFor asks whether this session can write.
//
// Reading and writing are separate abilities because writing is the half that
// can destroy somebody's work. A session that only reads says so here rather
// than partway through an edit.
func writerFor(
	s device.Editor,
) (device.Writer, error) {
	w, ok := s.(device.Writer)
	if !ok {
		return nil, fmt.Errorf("this session cannot write to a device")
	}

	return w, nil
}
