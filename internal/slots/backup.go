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
	"github.com/retr0h/tonestack/pkg/sdk"
	slotpkg "github.com/retr0h/tonestack/pkg/slot"
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
func holds(ctx context.Context, s sdk.Editor, setlist, slot int) ([]byte, error) {
	body, err := s.ReadPreset(ctx, setlist, slot)
	if err != nil {
		return nil, fmt.Errorf("reading slot %s before replacing it: %w",
			slotpkg.Label(slot), err)
	}

	return body, nil
}

// keep backs up one slot an edit is about to replace.
//
// A slice rather than a string so a caller replacing two slots can gather
// both, and so a slot holding nothing contributes nothing to report.
func keep(body []byte, opts EditOptions, slot int) ([]string, error) {
	path, err := backup(body, DeviceOptions{
		Deps:        opts.Deps,
		Slot:        slot,
		CatalogPath: opts.CatalogPath,
	}, opts.BackupDir)
	if err != nil {
		return nil, err
	}

	if path == "" {
		return nil, nil
	}

	return []string{path}, nil
}
