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
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/retr0h/tonestack/internal/cli"
	"github.com/retr0h/tonestack/pkg/sdk/device"
	slotpkg "github.com/retr0h/tonestack/pkg/sdk/slot"
)

// backupDir is where a slot's contents go before something overwrites them.
//
// Under the state directory rather than beside whatever the caller is doing,
// because these are written without being asked for and littering somebody's
// working directory with files they did not request is its own kind of rude.
func backupDir(named string) (string, error) {
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

// backup writes what a slot holds to a file, before something replaces it.
//
// A device has no undo. Every write here lands on somebody's own preset, and
// losing a tone to a bug in a sequence counter is not recoverable by apology,
// so the slot is read and kept first and a write that could not be backed up
// does not happen.
//
// An empty slot is not backed up and does not stop anything: there is nothing
// in it to lose, and writing a file saying so would only leave somebody
// wondering what it was for.
func backup(body []byte, opts DeviceOptions, dir string) (string, error) {
	// Nothing to keep. A slot the device answered nothing for holds nothing.
	if len(body) == 0 {
		return "", nil
	}

	var buf bytes.Buffer

	opts.As = FormatPreset

	if err := writeDeviceRig(&buf, body, opts); err != nil {
		return "", fmt.Errorf("reading slot %s before replacing it: %w",
			slotpkg.Label(opts.Slot), err)
	}

	// What ShowWith writes for a slot holding nothing. A comment is not a
	// preset, and keeping one would be keeping nothing.
	if strings.HasPrefix(buf.String(), "#") {
		return "", nil
	}

	dir, err := backupDir(dir)
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", fmt.Errorf("making room for a backup in %s: %w", dir, err)
	}

	// The slot and the moment, so a second write to the same slot does not
	// overwrite the copy taken before the first one.
	path := filepath.Join(dir, fmt.Sprintf("%s-%s.hlx",
		slotpkg.Label(opts.Slot), time.Now().UTC().Format("20060102-150405")))

	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		return "", fmt.Errorf("writing %s: %w", path, err)
	}

	return path, nil
}

// said names the file a slot's old contents went to.
//
// Printed rather than kept quiet, because a backup nobody knows about is a
// backup nobody restores from.
func said(w io.Writer, kept ...string) error {
	for _, path := range kept {
		if path == "" {
			continue
		}

		if _, err := fmt.Fprintf(w, "\n%s%s %s",
			cli.Indent, cli.Mute(w, "kept"), path); err != nil {
			return err
		}
	}

	return nil
}

// holds returns what a slot has in it, or nothing at all.
//
// Unlike slotBytes, a slot holding nothing is an answer rather than a
// failure. Reading the destination of a write is how a backup is taken, and
// writing into an empty slot has to keep working: there is nothing there to
// lose, which is a reason to carry on rather than a reason to stop.
func holds(ctx context.Context, s device.Editor, setlist, slot int) ([]byte, error) {
	body, err := s.ReadPreset(ctx, setlist, slot)
	if err != nil {
		return nil, fmt.Errorf("reading slot %s before replacing it: %w",
			slotpkg.Label(slot), err)
	}

	return body, nil
}

// at is one slot's contents, and which slot they came out of.
type at struct {
	body []byte
	slot int
}

// keep backs up every slot an edit is about to replace.
//
// Several, because a swap replaces two, and a loop rather than a call each so
// that there is one place a backup can fail rather than one per slot.
func keep(
	deps Deps,
	catalogPath, dir string,
	all ...at,
) ([]string, error) {
	out := []string(nil)

	for _, one := range all {
		path, err := backup(one.body, DeviceOptions{
			Deps:        deps,
			Slot:        one.slot,
			CatalogPath: catalogPath,
		}, dir)
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
func replacing(
	ctx context.Context,
	s device.Editor,
	deps Deps,
	catalogPath, dir string,
	setlist, slot int,
) ([]string, error) {
	body, err := holds(ctx, s, setlist, slot)
	if err != nil {
		return nil, err
	}

	return keep(deps, catalogPath, dir, at{body: body, slot: slot})
}
