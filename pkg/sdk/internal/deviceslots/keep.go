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

	"github.com/retr0h/tonestack/pkg/sdk/internal/backup"
	"github.com/retr0h/tonestack/pkg/sdk/internal/device"
	slotpkg "github.com/retr0h/tonestack/pkg/sdk/slot"
)

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

	return f.backups().Keep(ctx, backup.Held{At: at, Name: name, Body: body})
}
