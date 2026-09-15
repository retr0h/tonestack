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

package fileslots

import (
	"bytes"
	"context"
	"fmt"

	"github.com/retr0h/tonestack/pkg/sdk/internal/atomicfile"
	"github.com/retr0h/tonestack/pkg/sdk/preset"
	"github.com/retr0h/tonestack/pkg/sdk/result"
	"github.com/retr0h/tonestack/pkg/sdk/rig"
	slotpkg "github.com/retr0h/tonestack/pkg/sdk/slot"
)

// Export writes one slot of a file out to a file of its own.
//
// A rig by default, because that is the format this project speaks and the
// one that reads on other hardware. The device's own file is available for a
// faithful copy, which is a different thing: it carries the routing and
// snapshots a rig models but nobody chooses.
func (f *Flows) Export(
	ctx context.Context,
	path string,
	at slotpkg.Address,
	out string,
	as result.Format,
	existing result.Existing,
) (result.Written, error) {
	if err := Known(as); err != nil {
		return result.Written{}, err
	}

	doc, err := open(ctx, path)
	if err != nil {
		return result.Written{}, err
	}

	data, err := doc.Slot(at.Setlist, at.Slot)
	if err != nil {
		return result.Written{}, err
	}

	held := &preset.Document{
		Schema:  preset.Schema,
		Version: preset.Version,
		Data:    *data,
	}

	read := result.Reading{Name: data.Meta.Name, Doc: held}

	// Only the device's own file was asked for, so the lift is work nobody
	// wants.
	if as != result.FormatPreset {
		cat, err := f.catalog(ctx)
		if err != nil {
			return result.Written{}, err
		}

		spec, err := f.compiler().Lift(held, cat)
		if err != nil {
			return result.Written{}, err
		}

		read.Rig = spec
	}

	return Write(read, at.Slot, out, as, existing)
}

// Known refuses a format that is neither a rig nor the device's own file, the
// zero Format included.
//
// The same check and the same error a flag gives when it is handed a format by
// name, so a library caller cannot get a rig by misspelling hlx either.
func Known(
	as result.Format,
) error {
	probe := result.FormatRig

	return probe.Set(string(as))
}

// Write puts a reading on disk in the format that was asked for.
//
// One place, so a slot read off the hardware and one read out of a backup
// land as the same bytes. They describe the same preset, and an export that
// depended on which end it came from would be saying otherwise. So the device
// flows write through it too.
//
// A reading asked for as the device's own file carries a document. One that
// does not is an empty slot, and the caller refuses it before it gets here.
// existing says what happens to a file already at out.
func Write(
	read result.Reading,
	slot int,
	out string,
	as result.Format,
	existing result.Existing,
) (result.Written, error) {
	var buf bytes.Buffer

	if err := render(&buf, read, out, as); err != nil {
		return result.Written{}, err
	}

	if err := Save(out, buf.Bytes(), existing); err != nil {
		return result.Written{}, err
	}

	return result.Written{Slot: slot, Name: read.Name, Path: out}, nil
}

// Save puts data at path whole, doing what existing says about a file already
// there.
//
// ReplaceExisting puts it in that file's place. KeepExisting creates it only
// where nothing is, and the creation is the check: a file that appeared after
// the caller looked is refused with an error matching fs.ErrExist all the
// same. Every file an export or a build names is written through here.
func Save(
	path string,
	data []byte,
	existing result.Existing,
) error {
	if existing == result.KeepExisting {
		return atomicfile.WriteNew(path, data, 0o600)
	}

	return atomicfile.Write(path, data, 0o600)
}

// render encodes a reading in the format that was asked for.
func render(
	buf *bytes.Buffer,
	read result.Reading,
	out string,
	as result.Format,
) error {
	if as != result.FormatPreset {
		if err := rig.Write(buf, read.Rig); err != nil {
			return fmt.Errorf("writing the rig: %w", err)
		}

		return nil
	}

	if err := preset.Write(buf, read.Doc); err != nil {
		return fmt.Errorf("writing %s: %w", out, err)
	}

	return nil
}

// Import puts a standalone preset into one slot of a file.
//
// Whatever the slot held is gone, which is why the result goes to a new file.
func (*Flows) Import(
	ctx context.Context,
	path string,
	file string,
	at slotpkg.Address,
	out string,
) (result.Change, error) {
	doc, err := open(ctx, path)
	if err != nil {
		return result.Change{}, err
	}

	src, err := ReadPreset(ctx, file)
	if err != nil {
		return result.Change{}, err
	}

	dst, err := doc.Slot(at.Setlist, at.Slot)
	if err != nil {
		return result.Change{}, err
	}

	replaced := dst.Meta.Name
	mismatch := dst.Device != 0 && src.Data.Device != dst.Device

	*dst = src.Data

	if err := save(out, doc); err != nil {
		return result.Change{}, err
	}

	return result.Change{
		Action:   result.Imported,
		To:       result.At{Slot: at.Slot, Name: src.Data.Meta.Name},
		Replaced: replaced,
		Mismatch: mismatch,
		Path:     out,
	}, nil
}
