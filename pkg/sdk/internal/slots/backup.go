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
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/retr0h/tonestack/pkg/sdk/internal/atomicfile"
	"github.com/retr0h/tonestack/pkg/sdk/internal/device"
	"github.com/retr0h/tonestack/pkg/sdk/preset"
	"github.com/retr0h/tonestack/pkg/sdk/result"
	slotpkg "github.com/retr0h/tonestack/pkg/sdk/slot"
)

// backupDir is where a slot's contents go before something overwrites them.
//
// Under the state directory rather than beside whatever the caller is doing,
// because these are written without being asked for and littering somebody's
// working directory with files they did not request is its own kind of rude.
func backupDir(
	named string,
) (string, error) {
	if named != "" {
		return named, nil
	}

	if state := os.Getenv("XDG_STATE_HOME"); state != "" {
		return filepath.Join(state, "tonestack", "presets"), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("finding somewhere to keep a backup: %w", err)
	}

	return filepath.Join(home, ".local", "state", "tonestack", "presets"), nil
}

// held is one slot's contents, where they came out of, and what the device
// calls them.
type held struct {
	body []byte
	at   slotpkg.Address
	name string
}

// backup writes what a slot holds to a file, before something replaces it.
//
// A device has no undo. Every write here lands on somebody's own preset, and
// losing a tone to a bug in a sequence counter is not recoverable by apology,
// so the slot is read and kept first and a write that could not be backed up
// does not happen.
//
// A slot the device answered nothing for is not backed up and does not stop
// anything: there is nothing in it to lose. A slot with no blocks in it is
// different. Nothing here reads a chain out of it, which is not the same as
// there being nothing there, so it is kept as the bytes the device sent. The
// exception is one still called what the device names a slot nobody has
// touched: nothing in that is anybody's.
func (f *Flows) backup(
	ctx context.Context,
	h held,
) (string, error) {
	// Nothing to keep. A slot the device answered nothing for holds nothing.
	if len(h.body) == 0 {
		return "", nil
	}

	data, ext, err := f.keeping(ctx, h)
	if err != nil {
		return "", err
	}

	// Nothing of anybody's in it.
	if data == nil {
		return "", nil
	}

	dir, err := backupDir(f.BackupDir)
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", fmt.Errorf("making room for a backup in %s: %w", dir, err)
	}

	// The slot, its setlist and the moment to the nanosecond, so a second
	// backup of the same slot does not land on the first. Should two ever
	// share a name, the second fails rather than replacing the first.
	path := filepath.Join(dir, fmt.Sprintf("%s-s%d-%s%s",
		slotpkg.Label(h.at.Slot), h.at.Setlist,
		time.Now().UTC().Format("20060102-150405.000000000"), ext))

	if err := atomicfile.WriteNew(path, data, 0o600); err != nil {
		return "", err
	}

	return path, nil
}

// keeping decides what a backup holds, and the extension that says which.
//
// A preset file where the slot reads as one, so it can be put back with an
// import. The device's own bytes where it does not, so nothing it held is lost
// to a reader that does not understand it yet.
func (f *Flows) keeping(
	ctx context.Context,
	h held,
) ([]byte, string, error) {
	read, err := f.deviceReading(ctx, h.body, h.at.Slot, h.name, result.FormatPreset)
	if err != nil {
		return nil, "", fmt.Errorf("reading slot %s before replacing it: %w",
			slotpkg.Label(h.at.Slot), err)
	}

	// No blocks and the name it shipped with: a blank slot nobody has used.
	// A slot with no blocks that somebody renamed, or that no listing
	// named, might still hold something of theirs.
	if read.Empty() && h.name == untouched {
		return nil, "", nil
	}

	if read.Empty() {
		return h.body, ".bin", nil
	}

	var buf bytes.Buffer

	if err := preset.Write(&buf, read.Doc); err != nil {
		return nil, "", fmt.Errorf("keeping slot %s: %w",
			slotpkg.Label(h.at.Slot), err)
	}

	return buf.Bytes(), ".hlx", nil
}

// holds returns what a slot has in it, or nothing at all.
//
// Unlike slotBytes, a slot holding nothing is an answer rather than a
// failure. Reading the destination of a write is how a backup is taken, and
// writing into an empty slot has to keep working: there is nothing there to
// lose, which is a reason to carry on rather than a reason to stop.
func holds(
	ctx context.Context,
	s device.Editor,
	at slotpkg.Address,
) ([]byte, error) {
	body, err := s.ReadPreset(ctx, at.Setlist, at.Slot)
	if err != nil {
		return nil, fmt.Errorf("reading slot %s before replacing it: %w",
			slotpkg.Label(at.Slot), err)
	}

	return body, nil
}

// keep backs up every slot an edit is about to replace.
//
// Several, because a swap replaces two, and a loop rather than a call each so
// that there is one place a backup can fail rather than one per slot.
func (f *Flows) keep(
	ctx context.Context,
	all ...held,
) ([]string, error) {
	out := []string(nil)

	for _, one := range all {
		path, err := f.backup(ctx, one)
		if err != nil {
			return nil, err
		}

		// A slot holding nothing was not kept and has nothing to report.
		if path == "" {
			continue
		}

		out = append(out, path)
	}

	return out, nil
}

// replacing reads what a slot holds and keeps it, for a write that is about
// to put something else there.
//
// The read and the keeping together, because a caller that did one without
// the other would be either reading for nothing or replacing something it
// never looked at.
func (f *Flows) replacing(
	ctx context.Context,
	s device.Editor,
	at slotpkg.Address,
	name string,
) ([]string, error) {
	body, err := holds(ctx, s, at)
	if err != nil {
		return nil, err
	}

	return f.keep(ctx, held{body: body, at: at, name: name})
}
