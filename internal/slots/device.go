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
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/retr0h/tonestack/internal/catalogview"
	"github.com/retr0h/tonestack/internal/cli"
	"github.com/retr0h/tonestack/pkg/catalog"
	"github.com/retr0h/tonestack/pkg/chain"
	"github.com/retr0h/tonestack/pkg/sdk"
	"github.com/retr0h/tonestack/pkg/sdk/wire"
	slotpkg "github.com/retr0h/tonestack/pkg/slot"
)

// openDevice is how a session is obtained, so a test can stand in for it.
//
// The one line in this package that needs hardware; everything reached
// through it takes the session as an argument instead.
var openDevice = sdk.Open

// DeviceOptions says which setlist to read off an attached device.
type DeviceOptions struct {
	// Setlist selects one of the device's setlists, from zero.
	Setlist int
	// All includes slots holding nothing.
	All bool
	// Slot selects a position within the setlist, for reading one preset.
	Slot int
	// Name is what the device calls the preset, when it is already known.
	Name string
	// As is the format a read is written in. Empty means a rig.
	As Format
	// CatalogPath is the generated catalog for the target device.
	CatalogPath string
}

// ShowDevice reads one slot off an attached device.
//
// Read-only: the device hands back the preset and goes on playing whatever it
// was. Nothing is selected, loaded or written.
func ShowDevice(ctx context.Context, w io.Writer, opts DeviceOptions) error {
	s, err := openDevice(ctx)
	if err != nil {
		return err
	}

	defer s.Close()

	// Somebody who asked to look at a slot is told it holds nothing. Somebody
	// exporting one gets the error, because there is no file to write.
	if err := ShowWith(ctx, w, s, opts); err != nil {
		if !errors.Is(err, ErrEmptySlot) {
			return err
		}

		_, err := fmt.Fprintf(w, "# %s is empty\n", slotpkg.Label(opts.Slot))

		return err
	}

	return nil
}

// ShowWith reads one slot off the given session.
//
// Taking the session makes reading a device testable without one attached,
// which is the only part of this that needs hardware.
func ShowWith(
	ctx context.Context,
	w io.Writer,
	s sdk.Editor,
	opts DeviceOptions,
) error {
	// The name comes from the listing rather than the preset: what the device
	// hands back for one slot does not carry it.
	if opts.Name == "" {
		if found, err := s.Presets(ctx, opts.Setlist); err == nil {
			opts.Name = nameOf(found, opts.Slot)
		}
	}

	body, err := s.ReadPreset(ctx, opts.Setlist, opts.Slot)

	// A device that answered with something else is not a failure to report
	// as one: what arrived is worth keeping and showing, because it is how a
	// protocol change becomes visible.
	var answer *sdk.NotAPresetError
	if errors.As(err, &answer) {
		if err := dump(answer.Result); err != nil {
			return err
		}

		return describe(w, s.Model().Name, opts.Slot, answer.Shape())
	}

	if err != nil {
		return fmt.Errorf("reading slot %s: %w", slotpkg.Label(opts.Slot), err)
	}

	if err := dump(body); err != nil {
		return err
	}

	// A device answers an empty slot with no document at all. That is a slot
	// holding nothing rather than a failure, and a backup has to know the
	// difference to put a pedal back the way it was found.
	if body == nil {
		return fmt.Errorf("%w: %s", ErrEmptySlot, slotpkg.Label(opts.Slot))
	}

	return writeDeviceRig(w, body, opts)
}

// ErrEmptySlot is returned for a slot holding no preset.
//
// Reported rather than written out: an export that answered an empty slot
// with a file would leave somebody a preset that is not one.
var ErrEmptySlot = errors.New("the slot holds no preset")

// nameOf finds what a listing calls one slot.
//
// Searched rather than indexed. A device answers with every slot in order, so
// the two are the same today — and a listing that ever skipped an empty slot
// would silently name every preset after it wrongly.
func nameOf(found []wire.Preset, slot int) string {
	for _, p := range found {
		if p.Slot == slot {
			return p.Name
		}
	}

	return ""
}

// ExportDevice writes one slot off an attached device to a file.
//
// The same rig `presets show` prints, which is the point: a slot read off the
// hardware and one read out of a backup are the same document.
func ExportDevice(ctx context.Context, w io.Writer, opts ExportOptions) error {
	s, err := openDevice(ctx)
	if err != nil {
		return err
	}

	defer s.Close()

	return ExportWith(ctx, w, s, opts)
}

// ExportWith writes one slot off the given session to a file.
func ExportWith(
	ctx context.Context,
	w io.Writer,
	s sdk.Editor,
	opts ExportOptions,
) error {
	var buf bytes.Buffer

	err := ShowWith(ctx, &buf, s, DeviceOptions{
		Setlist:     opts.Setlist,
		Slot:        opts.Slot,
		As:          opts.As,
		CatalogPath: opts.CatalogPath,
	})
	if err != nil {
		return err
	}

	if err := os.WriteFile(opts.OutputPath, buf.Bytes(), 0o600); err != nil {
		return fmt.Errorf("writing %s: %w", opts.OutputPath, err)
	}

	_, err = fmt.Fprintf(w, "\n%s%s\n\n",
		cli.Indent, cli.Success(w, "wrote "+opts.OutputPath))

	return err
}

// ListDevice prints what an attached device holds.
//
// Read-only: it asks the device to describe a setlist and nothing more.
// Nothing is selected, loaded or written.
func ListDevice(ctx context.Context, w io.Writer, opts DeviceOptions) error {
	s, err := openDevice(ctx)
	if err != nil {
		return err
	}

	defer s.Close()

	return ListWith(ctx, w, s, opts)
}

// ListWith prints what the given session holds.
func ListWith(
	ctx context.Context,
	w io.Writer,
	s sdk.Editor,
	opts DeviceOptions,
) error {
	presets, err := s.Presets(ctx, opts.Setlist)
	if err != nil {
		return fmt.Errorf("listing presets: %w", err)
	}

	cat, err := catalogview.Open(opts.CatalogPath)
	if err != nil {
		return err
	}

	rows := make([][]string, 0, len(presets))

	var used int

	for _, p := range presets {
		// A slot is always named, so a name says nothing about whether
		// anything is in it. An untouched one keeps the name it shipped
		// with; a named one can still hold no blocks at all, and only
		// reading it says which.
		blank := p.Name == untouched

		chain := ""

		if !blank {
			blocks, err := chainAt(ctx, s, cat, opts.Setlist, p.Slot)
			if err != nil {
				return err
			}

			blank = len(blocks) == 0
			chain = flow(w, blocks, cat)
		}

		if blank {
			if !opts.All {
				continue
			}

			rows = append(rows, []string{
				cli.Mute(w, p.Label()), cli.Mute(w, p.Name), cli.Mute(w, "empty"),
			})

			continue
		}

		used++

		rows = append(rows, []string{cli.Accent(w, p.Label()), p.Name, chain})
	}

	return cli.Section{
		Title:  s.Model().Name,
		Detail: fmt.Sprintf("%s · %d in use", plural(len(presets), "slot"), used),
		// One address, the one printed on the pedal. What the device counts
		// underneath is its business, and --slot takes what is shown here.
		Headers: []string{"slot", "name", "chain"},
		Rows:    rows,
		Empty:   "no presets",
	}.Render(w)
}

// untouched is what a device calls a slot nobody has named.
const untouched = "New Preset"

// chainAt reads one slot and returns the blocks it holds.
//
// A slot that will not decode is reported as holding nothing rather than
// failing the listing around it: one unreadable preset should not hide the
// hundred that read.
func chainAt(
	ctx context.Context,
	s sdk.Editor,
	cat *catalog.Catalog,
	setlist, slot int,
) ([]chain.Block, error) {
	body, err := s.ReadPreset(ctx, setlist, slot)

	// An answer that is not a preset is skipped the way an undecodable one
	// is: this is a listing, and one slot nobody can read should not hide the
	// hundred that read.
	if errors.Is(err, sdk.ErrNotAPreset) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("reading slot %s: %w", slotpkg.Label(slot), err)
	}

	if body == nil {
		return nil, nil
	}

	preset, err := wire.DecodePreset(body)
	if err != nil {
		return nil, nil
	}

	c, err := chainOf("", preset, cat)
	if err != nil {
		return nil, nil
	}

	return c.Blocks, nil
}
