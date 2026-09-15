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

// Package backup keeps what a device slot held before a write replaces it.
//
// A device has no undo. Every write lands on somebody's own preset, and losing
// a tone to a bug in a sequence counter is not recoverable by apology, so a
// slot is kept on disk before it is overwritten. The policy is:
//
//  1. A slot the device answered nothing for is not kept. There is nothing in
//     it to lose.
//  2. A slot with no blocks, still called New Preset, is not kept. Nothing in
//     it is anybody's.
//  3. A slot that decodes is kept as .hlx, so it can be put back with an
//     import.
//  4. Anything else is kept as the device's bytes, .bin: a slot with no blocks
//     that somebody renamed, or that no listing named. Nothing here reads a
//     chain out of it, which is not the same as there being nothing there.
//  5. A backup never replaces another.
//  6. A write whose backup failed does not happen. That rule is the caller's:
//     the device flows call Keep before they write, and stop when it fails.
//
// An answer the Decoder cannot read at all is reported as an error rather than
// kept, and by rule 6 that stops the write.
package backup

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/retr0h/tonestack/pkg/sdk/internal/atomicfile"
	"github.com/retr0h/tonestack/pkg/sdk/preset"
	"github.com/retr0h/tonestack/pkg/sdk/slot"
)

// untouched is what a device calls a slot nobody has named.
const untouched = "New Preset"

// Keeper writes what slots held to a directory, by the policy above.
type Keeper struct {
	dir     string
	decoder Decoder
}

// New is a Keeper writing into dir and reading answers through d.
//
// An empty dir is the state directory: $XDG_STATE_HOME/tonestack/presets, or
// ~/.local/state/tonestack/presets when that variable is unset. It is worked
// out when something is kept, not here, so New cannot fail.
func New(
	dir string,
	d Decoder,
) *Keeper {
	return &Keeper{dir: dir, decoder: d}
}

// Keep writes every slot an edit is about to replace, and returns where each
// one went.
//
// Several, because a swap replaces two, and a loop rather than a call each so
// there is one place a backup can fail rather than one per slot. A slot not
// kept has nothing to report and is left out.
func (k *Keeper) Keep(
	ctx context.Context,
	held ...Held,
) ([]string, error) {
	out := []string(nil)

	for _, one := range held {
		path, err := k.one(ctx, one)
		if err != nil {
			return nil, err
		}

		if path == "" {
			continue
		}

		out = append(out, path)
	}

	return out, nil
}

// one writes what a slot holds to a file, or nothing when there is nothing of
// anybody's in it.
func (k *Keeper) one(
	ctx context.Context,
	h Held,
) (string, error) {
	// Rule 1.
	if len(h.Body) == 0 {
		return "", nil
	}

	data, ext, err := k.contents(ctx, h)
	if err != nil {
		return "", err
	}

	// Rule 2.
	if data == nil {
		return "", nil
	}

	dir, err := where(k.dir)
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", fmt.Errorf("making room for a backup in %s: %w", dir, err)
	}

	// Rule 5. The slot, its setlist and the moment to the nanosecond, so a
	// second backup of the same slot does not land on the first. Should two
	// ever share a name, the second fails rather than replacing the first.
	path := filepath.Join(dir, fmt.Sprintf("%s-s%d-%s%s",
		slot.Label(h.At.Slot), h.At.Setlist,
		time.Now().UTC().Format("20060102-150405.000000000"), ext))

	if err := atomicfile.WriteNew(path, data, 0o600); err != nil {
		return "", err
	}

	return path, nil
}

// contents decides what a backup holds, and the extension that says which.
//
// Nil data is a slot nobody has used, which is not kept.
func (k *Keeper) contents(
	ctx context.Context,
	h Held,
) ([]byte, string, error) {
	doc, err := k.decoder.Document(ctx, h.Body, h.At, h.Name)
	if err != nil {
		return nil, "", fmt.Errorf("reading slot %s before replacing it: %w",
			slot.Label(h.At.Slot), err)
	}

	// No blocks and the name it shipped with: a blank slot nobody has used.
	if doc == nil && h.Name == untouched {
		return nil, "", nil
	}

	// Rule 4. No blocks, and somebody renamed it or nothing named it.
	if doc == nil {
		return h.Body, ".bin", nil
	}

	// Rule 3.
	var buf bytes.Buffer

	if err := preset.Write(&buf, doc); err != nil {
		return nil, "", fmt.Errorf("keeping slot %s: %w",
			slot.Label(h.At.Slot), err)
	}

	return buf.Bytes(), ".hlx", nil
}
