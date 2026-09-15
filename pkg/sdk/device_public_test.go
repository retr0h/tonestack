//go:build device

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

package sdk_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk"
	"github.com/retr0h/tonestack/pkg/sdk/internal/device"
	"github.com/retr0h/tonestack/pkg/sdk/rig"
	"github.com/retr0h/tonestack/pkg/sdk/slot"
)

// DevicePublicTestSuite runs against a Helix on the bus.
//
// It writes a slot, so nothing runs it but somebody who asked for it: the
// device build tag keeps it out of `just test` and continuous integration, and
// the slot it may overwrite is named in the environment rather than chosen
// here. `just test-device` runs it.
type DevicePublicTestSuite struct {
	suite.Suite
}

// TestRoundTrip is the acceptance test the SDK design record asks for.
//
// A preset read off the device, written into another slot and read again
// must carry the same chain: the same gear in the same places, switched the
// same way, with every parameter the source set still set to that value.
// Import is the path under test: it encodes a preset rather than copying
// bytes, and a device seeks through a preset by a table of byte offsets that a
// wrong encoding shifts. The scratch slot is put back as it was afterwards,
// from the copy the write kept, and that is checked too.
//
// The chain, not the whole rig. Import builds into a blank slot and places the
// blocks, so footswitches, snapshot state and routing come from the blank
// rather than the source. That is known and tracked, and a test demanding
// otherwise would fail on the import rather than on the transport.
//
// The whole round trip is one Session: one claim, one handshake, and the read
// loop running between every step, which is the timing a held Session gives
// an agent.
func (s *DevicePublicTestSuite) TestRoundTrip() {
	ctx := context.Background()

	source := s.slotFrom("TONESTACK_SOURCE_SLOT", "01A")

	// The same switch the CLI reads, so a hardware run leaves a frame trace.
	var opts []sdk.Option
	if os.Getenv("TONESTACK_USB_DEBUG") != "" {
		opts = append(opts, sdk.WithTrace(os.Stderr))
	}

	session, err := sdk.New(opts...).Open(ctx)
	if errors.Is(err, device.ErrNoDevice) {
		s.T().Skip("no Helix attached")
	}

	s.Require().NoError(err)

	// Registered before anything that puts the slot back, so it runs after
	// it: cleanups run last first.
	s.T().Cleanup(func() {
		if err := session.Close(); err != nil {
			s.T().Errorf("closing the session: %v", err)
		}
	})

	want, err := session.Preset(ctx, slot.Address{Slot: source})
	s.Require().NoError(err)
	s.Require().NotNil(want.Doc,
		"%s holds nothing to round-trip; set TONESTACK_SOURCE_SLOT", slot.Label(source))

	// Writing a slot on somebody's pedal is a choice they make, not a default.
	s.Require().NotEmpty(os.Getenv("TONESTACK_SCRATCH_SLOT"),
		"set TONESTACK_SCRATCH_SLOT to a slot this test may overwrite, such as 42C")

	scratch := s.slotFrom("TONESTACK_SCRATCH_SLOT", "")
	s.Require().NotEqual(source, scratch, "the scratch slot must not be the source")

	before, err := session.Preset(ctx, slot.Address{Slot: scratch})
	s.Require().NoError(err)

	exported, err := session.Export(ctx, slot.Address{Slot: source},
		filepath.Join(s.T().TempDir(), "source.hlx"), sdk.FormatPreset)
	s.Require().NoError(err)

	put, err := session.Import(ctx, exported.Path, slot.Address{Slot: scratch})
	s.Require().NoError(err)

	// Registered the moment something was overwritten, so a failure below
	// still puts the slot back instead of leaving a copy in it.
	if len(put.Kept) > 0 {
		t, kept := s.T(), put.Kept[0]
		t.Logf("kept what %s held at %s", slot.Label(scratch), kept)
		t.Cleanup(func() { restore(ctx, t, session, kept, scratch, before) })
	} else {
		s.T().Logf("%s was empty, so it is left holding a copy of %s",
			slot.Label(scratch), slot.Label(source))
	}

	got, err := session.Preset(ctx, slot.Address{Slot: scratch})
	s.Require().NoError(err)
	s.Require().Empty(chainDiff(want.Rig, got.Rig),
		"a preset written to %s must read back with the chain in %s",
		slot.Label(scratch), slot.Label(source))
}

// chainDiff lists how got's chain falls short of want's.
//
// Parameters are compared one way: every parameter want sets must be set to
// the same value in got. A blank slot can carry parameters a source never set,
// and those are not a loss.
func chainDiff(want, got rig.Spec) []string {
	var diffs []string

	if len(want.Chain) != len(got.Chain) {
		return []string{fmt.Sprintf("%d blocks, want %d", len(got.Chain), len(want.Chain))}
	}

	for i, w := range want.Chain {
		g := got.Chain[i]

		if w.Gear != g.Gear || w.Role != g.Role {
			diffs = append(diffs, fmt.Sprintf("block %d is %s %q, want %s %q",
				i, g.Role, g.Gear, w.Role, w.Gear))
		}

		if !reflect.DeepEqual(w.Models, g.Models) || !reflect.DeepEqual(w.Enabled, g.Enabled) ||
			!reflect.DeepEqual(w.Path, g.Path) || !reflect.DeepEqual(w.Position, g.Position) {
			diffs = append(diffs, fmt.Sprintf("block %d moved, changed model or switched", i))
		}

		if w.Params == nil {
			continue
		}

		for key, value := range *w.Params {
			var have any
			if g.Params != nil {
				have = (*g.Params)[key]
			}

			if !reflect.DeepEqual(value, have) {
				diffs = append(
					diffs,
					fmt.Sprintf("block %d %s is %v, want %v", i, key, have, value),
				)
			}
		}
	}

	return diffs
}

// slotFrom reads a slot label from the environment, or falls back to one.
func (s *DevicePublicTestSuite) slotFrom(name, fallback string) int {
	label := os.Getenv(name)
	if label == "" {
		label = fallback
	}

	var n int

	s.Require().NoError(slot.NewValue(&n).Set(label), "%s=%q is not a slot", name, label)

	return n
}

// restore puts the kept copy back and checks it came back as it was.
//
// Reported rather than asserted, because it runs after the test body, and a
// failure here has to say where the original still is.
func restore(
	ctx context.Context,
	t *testing.T,
	session *sdk.Session,
	kept string,
	at int,
	before sdk.Reading,
) {
	if _, err := session.Import(ctx, kept, slot.Address{Slot: at}); err != nil {
		t.Errorf("could not put %s back: %v; the original is at %s", slot.Label(at), err, kept)

		return
	}

	after, err := session.Preset(ctx, slot.Address{Slot: at})
	if err != nil {
		t.Errorf("could not read %s after putting it back: %v", slot.Label(at), err)

		return
	}

	if diffs := chainDiff(before.Rig, after.Rig); len(diffs) > 0 {
		t.Errorf("%s did not come back as it was: %v; the original is at %s",
			slot.Label(at), diffs, kept)
	}
}

func TestDevicePublicTestSuite(t *testing.T) {
	suite.Run(t, new(DevicePublicTestSuite))
}
