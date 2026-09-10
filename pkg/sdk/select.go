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
	"fmt"
	"time"

	"github.com/retr0h/tonestack/pkg/sdk/wire"
)

// Opcodes that ask about, and change, what a device is playing.
const (
	// opSelectPreset loads a preset, the way stepping on a footswitch does.
	opSelectPreset = 20
	// opLoaded asks which preset that is.
	opLoaded = 23
)

// How long a switch is given, and how often the device is asked.
var (
	selectBudget = 10 * time.Second
	selectPoll   = 150 * time.Millisecond
)

// Loaded reports which preset the device is playing.
func (s *session) Loaded(ctx context.Context) (wire.Loaded, error) {
	resp, err := s.Call(ctx, channelData, opLoaded, nil)
	if err != nil {
		return wire.Loaded{}, err
	}

	return wire.DecodeLoaded(resp.Result)
}

// SelectPreset makes one preset the active one, and waits for it to land.
//
// The device loads it and starts making that sound, which is the difference
// between this and every other command here: reading a preset leaves the
// device playing whatever it was.
//
// Nothing is written. A preset is loaded into the edit buffer and the slot it
// came from is untouched, so this is the one device operation that changes
// what you hear without changing what the device holds.
//
// The wait is the whole point. A select is deferred: the device says it took
// the request and finishes afterwards, and it answers other questions while
// the switch is still in flight, so "it answered again" is not "it finished".
// A caller that returns early and closes the session leaves the device
// holding a half-finished switch, and it settles that by wiping its edit
// buffer — the preset loads with no blocks and no footswitch colours. Asking
// until the device reports the preset as current is the only honest signal
// that it is done.
func (s *session) SelectPreset(
	ctx context.Context,
	setlist, slot int,
) error {
	resp, err := s.Call(ctx, channelData, opSelectPreset, []wire.Arg{
		wire.Number(argSetlist, uint64(setlist)),
		wire.Number(argSlot, uint64(slot)),
	})
	if err != nil {
		return err
	}

	if resp.Status != wire.StatusAccepted && resp.Status != wire.StatusDone {
		return fmt.Errorf("selecting slot %d: unexpected status %d", slot, resp.Status)
	}

	return s.awaitLoaded(ctx, setlist, slot)
}

// awaitLoaded asks until the device says the preset is the one playing.
//
// A busy device refuses the question rather than answering it, which is
// patience rather than failure until the budget runs out.
func (s *session) awaitLoaded(
	ctx context.Context,
	setlist, slot int,
) error {
	deadline := time.Now().Add(selectBudget)

	for {
		if got, err := s.Loaded(ctx); err == nil {
			if got.Setlist == setlist && got.Slot == slot {
				return nil
			}
		}

		if time.Now().After(deadline) {
			return fmt.Errorf(
				"the device did not finish switching to slot %d within %s",
				slot, selectBudget)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(selectPoll):
		}
	}
}
