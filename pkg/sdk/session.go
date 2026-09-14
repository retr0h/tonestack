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

package sdk

import (
	"context"
	"errors"
	"fmt"

	"github.com/retr0h/tonestack/pkg/sdk/internal/device"
	"github.com/retr0h/tonestack/pkg/sdk/internal/slots"
	"github.com/retr0h/tonestack/pkg/sdk/slot"
)

// Session is one claim of the pedal and one handshake, used for as many
// operations as the caller wants.
//
// Safe for concurrent use, and runs one operation at a time. The lock is held
// for the whole of an operation, because a copy is several exchanges with the
// device and a swap is four, and nothing may land between them.
//
// Close it when done. While a Session is open the pedal's front panel stops
// refreshing its footswitches, and nothing else can claim the device.
type Session struct {
	client *Client
	editor device.Editor
	// op is the operation lock. One slot, so a caller waiting for it gives up
	// when its ctx ends.
	op chan struct{}
	// closed and closeErr change only under op.
	closed   bool
	closeErr error
}

// Open claims the attached device and starts a Session with it.
//
// ctx bounds the claim and the handshake, not the Session's life: each
// method's ctx bounds that operation.
//
// A Client has one Session open at a time. Open waits, subject to ctx, while
// another Session from this Client is open, because two claims of one editor
// interface are what leaves a pedal needing a power cycle. The one-shot device
// methods each open a Session of their own, so a caller holding a Session who
// calls one on the same Client blocks until its ctx ends.
func (c *Client) Open(
	ctx context.Context,
) (*Session, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	select {
	case c.claim <- struct{}{}:
	case <-ctx.Done():
		return nil, fmt.Errorf("waiting for the open session to close: %w", ctx.Err())
	}

	editor, err := c.opts.devices.Open(ctx)
	if err != nil {
		<-c.claim

		return nil, err
	}

	return &Session{client: c, editor: editor, op: make(chan struct{}, 1)}, nil
}

// Model is which device the Session is talking to.
func (s *Session) Model() string { return s.editor.Model().Name }

// lock takes the operation lock, or gives up when ctx does.
func (s *Session) lock(
	ctx context.Context,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	select {
	case s.op <- struct{}{}:
	case <-ctx.Done():
		return fmt.Errorf("waiting for the operation in flight: %w", ctx.Err())
	}

	if s.closed {
		s.unlock()

		return ErrClosed
	}

	return nil
}

// unlock gives the operation lock back.
func (s *Session) unlock() { <-s.op }

// operation runs call under the operation lock.
//
// The unlock is deferred, so a call that panics still gives the lock back as
// it unwinds and the caller's deferred Close can take it.
func operation[T any](
	ctx context.Context,
	s *Session,
	call func() (T, error),
) (T, error) {
	if err := s.lock(ctx); err != nil {
		var zero T

		return zero, err
	}

	defer s.unlock()

	return call()
}

// Presets reports what a setlist on the device holds, slot by slot.
func (s *Session) Presets(
	ctx context.Context,
	setlist int,
) (Listing, error) {
	return operation(ctx, s, func() (Listing, error) {
		return slots.ListWith(ctx, s.editor, slots.DeviceOptions{
			Deps:        s.client.deps(ctx),
			Setlist:     setlist,
			CatalogPath: s.client.opts.catalog,
		})
	})
}

// Preset reads one slot as the rig it describes.
//
// A slot holding nothing is an answer rather than a failure: it comes back
// named for the slot, and empty.
func (s *Session) Preset(
	ctx context.Context,
	at slot.Address,
) (Reading, error) {
	return operation(ctx, s, func() (Reading, error) {
		read, err := slots.ShowWith(ctx, s.editor, slots.DeviceOptions{
			Deps:        s.client.deps(ctx),
			Setlist:     at.Setlist,
			Slot:        at.Slot,
			CatalogPath: s.client.opts.catalog,
		})

		// Somebody who asked to look at a slot is answered that it holds
		// nothing. Somebody exporting one gets the error, because there is no
		// file to write.
		if errors.Is(err, slots.ErrEmptySlot) {
			return Reading{Name: slot.Label(at.Slot)}, nil
		}

		return read, err
	})
}

// Export writes one slot out to a file.
//
// as is the format. Empty writes a rig, which is what reads on other hardware;
// "hlx" writes the device's own file, a faithful copy.
func (s *Session) Export(
	ctx context.Context,
	at slot.Address,
	out string,
	as string,
) (Written, error) {
	return operation(ctx, s, func() (Written, error) {
		return slots.ExportWith(ctx, s.editor, slots.ExportOptions{
			Deps:        s.client.deps(ctx),
			Setlist:     at.Setlist,
			Slot:        at.Slot,
			OutputPath:  out,
			As:          slots.Format(as),
			CatalogPath: s.client.opts.catalog,
		})
	})
}

// Import puts a preset file into a slot.
//
// Whatever the slot held is gone. A device has no undo, so what was there is
// read and kept first, in the directory WithBackupDir named.
func (s *Session) Import(
	ctx context.Context,
	file string,
	at slot.Address,
) (Change, error) {
	return operation(ctx, s, func() (Change, error) {
		return slots.ImportWith(ctx, s.editor, slots.ImportOptions{
			BackupDir:   s.client.opts.backupDir,
			Deps:        s.client.deps(ctx),
			File:        file,
			Setlist:     at.Setlist,
			Slot:        at.Slot,
			CatalogPath: s.client.opts.catalog,
		})
	})
}

// Copy puts what one slot holds into another.
//
// The preset moves exactly as it was written. What the destination held is
// kept first.
func (s *Session) Copy(
	ctx context.Context,
	from, to slot.Address,
) (Change, error) {
	return operation(ctx, s, func() (Change, error) {
		return slots.CopyWith(ctx, s.editor, s.edit(ctx, from, to))
	})
}

// Swap exchanges what two slots hold. Both are kept first.
//
// A swap whose first write landed finishes its second, whoever stops waiting.
func (s *Session) Swap(
	ctx context.Context,
	a, b slot.Address,
) (Change, error) {
	return operation(ctx, s, func() (Change, error) {
		return slots.SwapWith(ctx, s.editor, s.edit(ctx, a, b))
	})
}

// edit is what the flows take for a move between two slots.
func (s *Session) edit(
	ctx context.Context,
	from, to slot.Address,
) slots.EditOptions {
	return slots.EditOptions{
		Deps:        s.client.deps(ctx),
		FromSetlist: from.Setlist,
		FromSlot:    from.Slot,
		ToSetlist:   to.Setlist,
		ToSlot:      to.Slot,
		BackupDir:   s.client.opts.backupDir,
		CatalogPath: s.client.opts.catalog,
	}
}

// Select makes one preset the active one, and waits for the device to say it
// has.
//
// Nothing is written: the preset goes into the edit buffer and the slot it
// came from is untouched.
func (s *Session) Select(
	ctx context.Context,
	at slot.Address,
) (Change, error) {
	return operation(ctx, s, func() (Change, error) {
		return slots.SelectWith(ctx, s.editor, slots.DeviceOptions{
			Deps:    s.client.deps(ctx),
			Setlist: at.Setlist,
			Slot:    at.Slot,
		})
	})
}

// Close ends the Session and lets the device go.
//
// An operation in flight finishes first: Close waits for the operation lock
// with no budget of its own, which is finite because every exchange with the
// device is bounded. Idempotent. It always releases the interface, and returns
// the error that ended the Session's read loop, if one did. Every method
// called after it returns ErrClosed.
func (s *Session) Close() error {
	s.op <- struct{}{}
	defer s.unlock()

	if s.closed {
		return s.closeErr
	}

	s.closed = true
	s.closeErr = s.editor.Close()

	<-s.client.claim

	return s.closeErr
}
