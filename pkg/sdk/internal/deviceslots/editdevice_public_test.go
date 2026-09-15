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

package deviceslots_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/retr0h/tonestack/pkg/sdk/internal/backup"
	"github.com/retr0h/tonestack/pkg/sdk/internal/device"
	"github.com/retr0h/tonestack/pkg/sdk/internal/device/mocks"
	"github.com/retr0h/tonestack/pkg/sdk/internal/deviceslots"
	"github.com/retr0h/tonestack/pkg/sdk/internal/wire"
	"github.com/retr0h/tonestack/pkg/sdk/result"
	slotpkg "github.com/retr0h/tonestack/pkg/sdk/slot"
)

// did flattens what a write reported, so a test can assert on the facts of it
// without also asserting on how a terminal paints them.
func did(
	c result.Change,
) string {
	parts := []string{
		string(c.Action),
		slotpkg.Label(c.To.Slot), c.To.Name,
		c.Replaced, c.Path,
	}

	if c.From != nil {
		parts = append(parts, slotpkg.Label(c.From.Slot), c.From.Name)
	}

	return strings.Join(append(parts, c.Kept...), " ")
}

// keepingIn is f with what a write replaces kept in dir, the way the sdk
// Client configures it.
func keepingIn(
	f *deviceslots.Flows,
	dir string,
) *deviceslots.Flows {
	f.Backups = backup.New(dir, deviceslots.NewDecoder(f))

	return f
}

// EditDevicePublicTestSuite covers moving a preset between slots on a device.
//
// Nothing here reaches hardware. What it establishes is that the bytes leaving
// for the device are the bytes that arrived from it, because a preset is
// seeked through by a table of byte offsets and the surest way to keep those
// right is to change nothing.
type EditDevicePublicTestSuite struct {
	suite.Suite

	ctrl *gomock.Controller
	dev  *editable
}

// editable is a session that can both read and write.
type editable struct {
	*mocks.MockEditor
	*mocks.MockWriter
}

func (e *editable) Close() error { return nil }

func (s *EditDevicePublicTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.dev = &editable{
		MockEditor: mocks.NewMockEditor(s.ctrl),
		MockWriter: mocks.NewMockWriter(s.ctrl),
	}
}

func (s *EditDevicePublicTestSuite) TearDownTest() { s.ctrl.Finish() }

// errWriteRefused is what a device that will not take a write says.
var errWriteRefused = errors.New("boom")

// requireKept checks that a failed write names the n backups it made first,
// and still wraps what went wrong. None made means no KeptError at all.
func requireKept(
	r *require.Assertions,
	err error,
	n int,
) {
	var kept *deviceslots.KeptError

	if n == 0 {
		r.NotErrorAs(err, &kept, "nothing was kept, so nothing is named")

		return
	}

	r.ErrorAs(err, &kept)
	r.Len(kept.Kept, n)
	r.NotNil(errors.Unwrap(err), "what went wrong is still there to match")

	for _, path := range kept.Kept {
		r.FileExists(path)
		r.Contains(err.Error(), path, "the error says where the backup is")
	}
}

// answer returns one slot as the hardware sent it.
func (s *EditDevicePublicTestSuite) answer() []byte {
	raw, err := os.ReadFile(
		filepath.Join("..", "wire", "testdata", "preset.bin"))
	s.Require().NoError(err)

	return raw
}

// listing is what the device says the setlist holds.
func (s *EditDevicePublicTestSuite) listing() []wire.Preset {
	return []wire.Preset{
		{Slot: 0, Name: "Chunky Monkey"},
		{Slot: 3, Name: "Black Rusty"},
	}
}

// elsewhere is what the device says a second setlist holds. Slot 01A there
// is called something else than slot 01A in the first, which is how a name
// looked up in the wrong setlist shows.
func (s *EditDevicePublicTestSuite) elsewhere() []wire.Preset {
	return []wire.Preset{{Slot: 0, Name: "Holiday"}}
}

// backupDir returns somewhere a backup can go, or somewhere it cannot.
func (s *EditDevicePublicTestSuite) backupDir(bad bool) string {
	dir := s.T().TempDir()
	if !bad {
		return dir
	}

	// A file where a directory would have to be, so nothing can be made
	// under it.
	path := filepath.Join(dir, "in-the-way")
	s.Require().NoError(os.WriteFile(path, []byte("x"), 0o600))

	return filepath.Join(path, "under-it")
}

// expectListing sets up the listing every edit starts from.
func (s *EditDevicePublicTestSuite) expectListing(reader *mocks.MockEditor, ok bool) {
	if !ok {
		reader.EXPECT().Presets(gomock.Any(), 0).Return(nil, errors.New("boom"))

		return
	}

	reader.EXPECT().Presets(gomock.Any(), 0).Return(s.listing(), nil)
}

// expectRead sets up one slot being read: answered, refused, or answered with
// something that is not a preset.
func (s *EditDevicePublicTestSuite) expectRead(
	reader *mocks.MockEditor,
	setlist, slot int,
	outcome string,
) *gomock.Call {
	switch outcome {
	case "refused":
		return reader.EXPECT().ReadPreset(gomock.Any(), setlist, slot).
			Return(nil, errors.New("boom"))
	case "not a preset":
		return reader.EXPECT().ReadPreset(gomock.Any(), setlist, slot).
			Return(nil, nil)
	default:
		return reader.EXPECT().ReadPreset(gomock.Any(), setlist, slot).
			Return(s.answer(), nil)
	}
}

// expectWrite sets up one slot being written, byte for byte. A preset that
// changed on the way through would leave the device's offset table pointing at
// the wrong places, and the device would accept it and then read the preset as
// empty.
func (s *EditDevicePublicTestSuite) expectWrite(
	setlist, slot int,
	name string,
	ok bool,
) *gomock.Call {
	call := s.dev.MockWriter.EXPECT().
		WriteNamedPreset(gomock.Any(), setlist, slot, name, s.answer())

	if !ok {
		return call.Return(errWriteRefused)
	}

	return call.Return(nil)
}

// TestCopy writes one slot of a device over another.
func (s *EditDevicePublicTestSuite) TestCopy() {
	tests := []struct {
		name string
		// whether the listing comes back, how the read is answered, and
		// whether the write lands. An empty outcome means the call is never
		// reached.
		listed bool
		read   string
		write  string
		// a session that can read but not write.
		readOnly bool
		// how the destination read is answered. Empty means answered.
		readTo string
		// somewhere a backup cannot be written.
		badBackup bool
		// a destination in the second setlist, slot 01A, rather than slot
		// 02A beside the source.
		cross bool
		// the second setlist's listing is refused.
		crossRefused bool
		// the destination is the source itself.
		same bool

		contains []string
		// part of the backup's filename, which names the setlist it came from.
		keptAs string
		// how many backups the error names.
		keptInErr int
		is        error
		errText   string
	}{
		{
			// Nothing to move and nothing to keep. Reading and backing up a
			// slot only to write it back over itself is a round trip for
			// nothing, and the command was almost certainly a typo.
			name:    "a slot copied onto itself",
			same:    true,
			is:      deviceslots.ErrSameSlot,
			errText: "01A",
		},
		{
			name:   "a preset sent back byte for byte",
			listed: true,
			read:   "answered",
			write:  "landed",
			contains: []string{
				"01A", "02A", "Chunky Monkey",
				// The destination is overwritten and there is no undo on a
				// device.
				"Black Rusty",
			},
		},
		{
			// Each slot is named from its own setlist. Looking the
			// destination up in the source's setlist would report, and
			// back up, a preset under another preset's name.
			name:   "a preset copied into another setlist",
			listed: true,
			read:   "answered",
			write:  "landed",
			cross:  true,
			keptAs: "01A-s1-",
		},
		{name: "a listing it cannot get", errText: "listing presets"},
		{
			name:         "a second setlist it cannot list",
			listed:       true,
			cross:        true,
			crossRefused: true,
			errText:      "listing presets",
		},
		{
			// The destination is read so it can be kept. A device that will
			// not say what is there is a device that cannot be replaced
			// safely.
			name:    "a destination it cannot read",
			listed:  true,
			read:    "answered",
			readTo:  "refused",
			errText: "before replacing it",
		},
		{
			// Nowhere to put what the destination held, so it is not
			// replaced. A write nobody can undo does not happen.
			name:      "a backup it cannot write",
			listed:    true,
			read:      "answered",
			badBackup: true,
			errText:   "making room for a backup",
		},
		{
			name:    "a slot it cannot read",
			listed:  true,
			read:    "refused",
			errText: "reading slot 01A",
		},
		{
			name:    "an answer that is not a preset",
			listed:  true,
			read:    "not a preset",
			errText: "did not answer with a preset",
		},
		{
			// The destination was kept before the write failed, and the
			// error is the only thing left to say where.
			name:      "a slot it cannot write",
			listed:    true,
			read:      "answered",
			write:     "refused",
			is:        errWriteRefused,
			keptInErr: 1,
			errText:   "writing slot 02A",
		},
		{
			// Reading and writing are separate abilities, because writing is
			// the half that can destroy somebody's work.
			name:     "a session that cannot write",
			listed:   true,
			read:     "answered",
			readOnly: true,
			errText:  "cannot write",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			reader := s.dev.MockEditor

			dev := device.Editor(s.dev)
			if tt.readOnly {
				reader = mocks.NewMockEditor(s.ctrl)
				dev = device.Editor(reader)
			}

			toSetlist, toSlot, toName := 0, 3, "Black Rusty"
			if tt.cross {
				toSetlist, toSlot, toName = 1, 0, "Holiday"
			}

			if tt.same {
				toSlot = 0
			} else {
				s.expectListing(reader, tt.listed)
			}

			if tt.cross && tt.crossRefused {
				reader.EXPECT().Presets(gomock.Any(), 1).
					Return(nil, errors.New("boom"))
			} else if tt.cross {
				reader.EXPECT().Presets(gomock.Any(), 1).Return(s.elsewhere(), nil)
			}

			if tt.read != "" {
				s.expectRead(reader, 0, 0, tt.read)
			}

			// The destination is read as well now, so that what it held can
			// be kept before it stops holding it. A swap needs no such read:
			// it has already read both slots to move them. A session that
			// cannot write never gets that far.
			if tt.read == "answered" && !tt.readOnly {
				outcome := tt.readTo
				if outcome == "" {
					outcome = "answered"
				}

				s.expectRead(reader, toSetlist, toSlot, outcome)
			}

			// The destination takes the source's name along with its
			// contents.
			if tt.write != "" {
				s.expectWrite(toSetlist, toSlot, "Chunky Monkey", tt.write == "landed")
			}

			f := keepingIn(&deviceslots.Flows{}, s.backupDir(tt.badBackup))

			change, err := f.Copy(context.Background(), dev,
				slotpkg.Address{}, slotpkg.Address{Setlist: toSetlist, Slot: toSlot})

			if tt.errText != "" {
				s.Require().Error(err)
				s.Require().Contains(err.Error(), tt.errText)

				if tt.is != nil {
					s.Require().ErrorIs(err, tt.is)
				}

				requireKept(s.Require(), err, tt.keptInErr)

				return
			}

			s.Require().NoError(err)
			s.Require().Equal(toName, change.Replaced,
				"what the destination was called, in its own setlist")

			for _, want := range tt.contains {
				s.Require().Contains(did(change), want)
			}

			if tt.keptAs != "" {
				s.Require().Len(change.Kept, 1)
				s.Require().Contains(filepath.Base(change.Kept[0]), tt.keptAs)
			}
		})
	}
}

// TestSwap exchanges two slots on a device.
func (s *EditDevicePublicTestSuite) TestSwap() {
	tests := []struct {
		name string
		// the listing, then how each of the two slots answers, then whether
		// each of the two writes lands. An empty outcome means the call is
		// never reached.
		listed   bool
		reads    []string
		writes   []string
		readOnly bool
		// somewhere a backup cannot be written.
		badBackup bool

		// the second slot is 01A of the second setlist rather than 02A of the
		// first.
		cross bool
		// part of each backup's filename, in the order they were kept.
		keptIn []string
		// the second slot is the first one.
		same bool
		// the caller stops waiting while the first write is going out.
		cancelsOnFirstWrite bool

		contains string
		// how many backups the error names.
		keptInErr int
		is        error
		errText   string
	}{
		{
			name:    "a slot swapped with itself",
			same:    true,
			is:      deviceslots.ErrSameSlot,
			errText: "01A",
		},
		{
			// A swap that has written one slot finishes. Stopping between
			// the two writes leaves both slots holding the same preset, with
			// one original only in a backup.
			name:                "a caller who stops waiting during the first write",
			listed:              true,
			reads:               []string{"answered", "answered"},
			cancelsOnFirstWrite: true,
			contains:            "swapped",
		},
		{
			name:     "both slots, each holding what the other did",
			listed:   true,
			reads:    []string{"answered", "answered"},
			writes:   []string{"landed", "landed"},
			contains: "swapped",
		},
		{
			// Each slot keeps the name it had in its own setlist when it
			// moves. Named from the source's setlist, both would come back
			// called Chunky Monkey.
			name:   "two slots in two setlists",
			listed: true,
			reads:  []string{"answered", "answered"},
			writes: []string{"landed", "landed"},
			cross:  true,
			// The destination first, from the second setlist, then the
			// source from the first.
			keptIn:   []string{"01A-s1-", "01A-s0-"},
			contains: "swapped",
		},
		{name: "a listing it cannot get", errText: "listing presets"},
		{
			// A swap reads both slots to move them and keeps them from those
			// same reads, so nowhere to put them stops it.
			name:      "a backup it cannot write",
			listed:    true,
			reads:     []string{"answered", "answered"},
			badBackup: true,
			errText:   "making room for a backup",
		},
		{
			name:    "the first slot, which it cannot read",
			listed:  true,
			reads:   []string{"refused"},
			errText: "reading slot 01A",
		},
		{
			name:    "the second slot, which it cannot read",
			listed:  true,
			reads:   []string{"answered", "refused"},
			errText: "reading slot 02A",
		},
		{
			name:      "the destination, which it cannot write",
			listed:    true,
			reads:     []string{"answered", "answered"},
			writes:    []string{"refused"},
			is:        errWriteRefused,
			keptInErr: 2,
			errText:   "writing slot 02A",
		},
		{
			// The worst place to fail: both slots hold the source, and the
			// destination's preset is only in a backup. The error says which.
			name:      "the source, which it cannot write",
			listed:    true,
			reads:     []string{"answered", "answered"},
			writes:    []string{"landed", "refused"},
			is:        errWriteRefused,
			keptInErr: 2,
			errText:   "writing slot 01A",
		},
		{
			name:     "a session that cannot write",
			listed:   true,
			reads:    []string{"answered", "answered"},
			readOnly: true,
			errText:  "cannot write",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			reader := s.dev.MockEditor

			dev := device.Editor(s.dev)
			if tt.readOnly {
				reader = mocks.NewMockEditor(s.ctrl)
				dev = device.Editor(reader)
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			toSetlist, toSlot, toName := 0, 3, "Black Rusty"
			if tt.cross {
				toSetlist, toSlot, toName = 1, 0, "Holiday"

				reader.EXPECT().Presets(gomock.Any(), 1).Return(s.elsewhere(), nil)
			}

			if tt.same {
				toSlot = 0
			} else {
				s.expectListing(reader, tt.listed)
			}

			if tt.cancelsOnFirstWrite {
				// The first write lands and the caller gives up while it
				// does. The second is a device refusing a context that has
				// ended, the way a session's write does before sending.
				first := s.dev.MockWriter.EXPECT().
					WriteNamedPreset(gomock.Any(), toSetlist, toSlot, "Chunky Monkey", s.answer()).
					DoAndReturn(func(context.Context, int, int, string, []byte) error {
						cancel()

						return nil
					})
				s.dev.MockWriter.EXPECT().
					WriteNamedPreset(gomock.Any(), 0, 0, toName, s.answer()).
					DoAndReturn(func(ctx context.Context, _, _ int, _ string, _ []byte) error {
						return ctx.Err()
					}).
					After(first)
			}

			// A device that failed halfway through would leave one slot
			// holding a copy of the other and the original gone, so both
			// slots are read before either is written.
			var last *gomock.Call

			for i, outcome := range tt.reads {
				call := s.expectRead(reader,
					[]int{0, toSetlist}[i], []int{0, toSlot}[i], outcome)
				if last != nil {
					call.After(last)
				}

				last = call
			}

			for i, outcome := range tt.writes {
				call := s.expectWrite(
					[]int{toSetlist, 0}[i],
					[]int{toSlot, 0}[i],
					[]string{"Chunky Monkey", toName}[i],
					outcome == "landed")

				if last != nil {
					call.After(last)
				}
			}

			f := keepingIn(&deviceslots.Flows{}, s.backupDir(tt.badBackup))

			change, err := f.Swap(ctx, dev,
				slotpkg.Address{}, slotpkg.Address{Setlist: toSetlist, Slot: toSlot})

			if tt.errText != "" {
				s.Require().Error(err)
				s.Require().Contains(err.Error(), tt.errText)

				if tt.is != nil {
					s.Require().ErrorIs(err, tt.is)
				}

				requireKept(s.Require(), err, tt.keptInErr)

				return
			}

			s.Require().NoError(err)
			s.Require().Contains(did(change), tt.contains)
			s.Require().Equal(toName, change.Replaced,
				"what the second slot was called, in its own setlist")

			if tt.keptIn != nil {
				s.Require().Len(change.Kept, len(tt.keptIn))

				for i, want := range tt.keptIn {
					s.Require().Contains(filepath.Base(change.Kept[i]), want)
				}
			}
		})
	}
}

func TestEditDevicePublicTestSuite(t *testing.T) {
	suite.Run(t, new(EditDevicePublicTestSuite))
}
