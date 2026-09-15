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

package fileslots_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/internal/fileslots"
	"github.com/retr0h/tonestack/pkg/sdk/internal/setlist"
	"github.com/retr0h/tonestack/pkg/sdk/result"
	slotpkg "github.com/retr0h/tonestack/pkg/sdk/slot"
)

// EditPublicTestSuite covers the operations that write a file: copying,
// swapping, exporting and importing.
type EditPublicTestSuite struct {
	suite.Suite
}

// reread loads a setlist this suite just wrote.
func (s *EditPublicTestSuite) reread(
	path string,
) *setlist.Document {
	f, err := os.Open(path) //nolint:gosec // a path this test chose
	s.Require().NoError(err)

	defer func() { s.Require().NoError(f.Close()) }()

	doc, err := setlist.Read(f)
	s.Require().NoError(err)

	return doc
}

// background is ctx, or a context nobody has stopped waiting on.
func background(
	ctx context.Context,
) context.Context {
	if ctx == nil {
		return context.Background()
	}

	return ctx
}

// formatFor is as, or a rig for a row that says nothing about the format. An
// export refuses the zero Format, so a row has to ask for one.
func formatFor(
	as result.Format,
) result.Format {
	if as == "" {
		return result.FormatRig
	}

	return as
}

// TestCopy covers writing one slot of a file over another.
func (s *EditPublicTestSuite) TestCopy() {
	tests := []struct {
		name     string
		ctx      context.Context
		path     string
		from, to slotpkg.Address
		// where to write, under this case's own directory.
		out string
		// what the first slots must hold afterwards.
		want     []string
		contains []string
		errText  string
	}{
		{
			name:     "a copy, which leaves the source alone",
			to:       slotpkg.Address{Slot: 1},
			want:     []string{"First", "First"},
			contains: []string{"copied", "01A", "01B"},
		},
		{
			name:    "a file that is not there",
			path:    fixture("nope.hls"),
			to:      slotpkg.Address{Slot: 1},
			errText: "opening",
		},
		{
			name:    "a source that is not there",
			from:    slotpkg.Address{Slot: 99},
			to:      slotpkg.Address{Slot: 1},
			errText: "no such slot",
		},
		{
			name:    "a destination that is not there",
			to:      slotpkg.Address{Slot: 99},
			errText: "no such slot",
		},
		{
			// The destination's setlist is read, not only its slot.
			name:    "a destination in a setlist the file does not hold",
			to:      slotpkg.Address{Setlist: 3, Slot: 1},
			errText: "no such slot",
		},
		{
			name:    "a destination directory that is not there",
			to:      slotpkg.Address{Slot: 1},
			out:     filepath.Join("no", "out.hls"),
			errText: "writing",
		},
		{
			name:    "a caller who stopped waiting",
			ctx:     cancelled(),
			to:      slotpkg.Address{Slot: 1},
			errText: context.Canceled.Error(),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			dir := s.T().TempDir()

			out := filepath.Join(dir, "out.hls")
			if tt.out != "" {
				out = filepath.Join(dir, tt.out)
			}

			path := fixture("setlist.hls")
			if tt.path != "" {
				path = tt.path
			}

			change, err := (&fileslots.Flows{}).Copy(background(tt.ctx), path, tt.from, tt.to, out)

			if tt.errText != "" {
				s.Require().ErrorContains(err, tt.errText)
				s.Require().NoFileExists(out)

				return
			}

			s.Require().NoError(err)
			s.Require().Equal(out, change.Path)

			doc := s.reread(out)
			for i, want := range tt.want {
				s.Require().Equal(want, doc.Setlists[0].Slots[i].Meta.Name)
			}

			for _, want := range tt.contains {
				s.Require().Contains(did(change), want)
			}
		})
	}
}

// TestSwap covers exchanging two slots of a file.
func (s *EditPublicTestSuite) TestSwap() {
	tests := []struct {
		name    string
		ctx     context.Context
		path    string
		a, b    slotpkg.Address
		want    []string
		errText string
	}{
		{
			name: "two slots, each holding what the other did",
			b:    slotpkg.Address{Slot: 1},
			want: []string{"Second", "First"},
		},
		{
			name:    "a file that is not there",
			path:    fixture("nope.hls"),
			b:       slotpkg.Address{Slot: 1},
			errText: "opening",
		},
		{
			// The first slot's setlist is read, not only its slot.
			name:    "a first slot in a setlist the file does not hold",
			a:       slotpkg.Address{Setlist: 2},
			b:       slotpkg.Address{Slot: 1},
			errText: "no such slot",
		},
		{
			name:    "a caller who stopped waiting",
			ctx:     cancelled(),
			b:       slotpkg.Address{Slot: 1},
			errText: context.Canceled.Error(),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			out := filepath.Join(s.T().TempDir(), "out.hls")

			path := fixture("setlist.hls")
			if tt.path != "" {
				path = tt.path
			}

			change, err := (&fileslots.Flows{}).Swap(background(tt.ctx), path, tt.a, tt.b, out)

			if tt.errText != "" {
				s.Require().ErrorContains(err, tt.errText)

				return
			}

			s.Require().NoError(err)
			s.Require().Equal(result.Swapped, change.Action)

			doc := s.reread(out)
			for i, want := range tt.want {
				s.Require().Equal(want, doc.Setlists[0].Slots[i].Meta.Name)
			}
		})
	}
}

// TestExport covers writing one slot of a file to a file of its own.
func (s *EditPublicTestSuite) TestExport() {
	tests := []struct {
		name    string
		ctx     context.Context
		path    string
		at      slotpkg.Address
		as      result.Format
		catalog string
		out     string

		// what the written file must say.
		wrote []string
		// what showing the written file must say, for the device's own
		// format.
		shows []string
		// what the answer must name.
		named   string
		errText string
	}{
		{
			// A rig is what this project speaks, and the format that reads on
			// other hardware. The device's own file is a faithful copy, which
			// is a different thing and has to be asked for.
			name:    "a rig, which is what somebody gets by default",
			catalog: catalogPath(),
			out:     "one.yaml",
			wrote: []string{
				"schema: RigSpec",
				"gear:",
				// A lifted rig records the exact model, since a name does not
				// identify one.
				"models:",
			},
		},
		{
			name:  "the device's own file, when asked for",
			as:    result.FormatPreset,
			out:   "one.hlx",
			shows: []string{"First", "Ampeg SVT"},
			named: "First",
		},
		{
			// The setlist half of the address is read too.
			name:  "a slot in the second setlist of a bundle",
			path:  fixture("bundle.hlb"),
			at:    slotpkg.Address{Setlist: 1},
			as:    result.FormatPreset,
			out:   "only.hlx",
			named: "Only",
		},
		{name: "a file that is not there", path: fixture("nope.hls"), errText: "opening"},
		{
			// Refused before the file is opened, with the error a flag gives,
			// rather than written as a rig nobody asked for.
			name:    "a format that is neither a rig nor the device's own file",
			path:    fixture("nope.hls"),
			as:      result.Format("yaml"),
			errText: result.ErrUnknownFormat.Error(),
		},
		{
			name:    "a slot that is not there",
			at:      slotpkg.Address{Slot: 99},
			as:      result.FormatPreset,
			errText: "no such slot",
		},
		{
			name:    "a destination directory that is not there",
			as:      result.FormatPreset,
			out:     filepath.Join("no", "x.hlx"),
			errText: "writing",
		},
		{
			name:    "a catalog that is not there",
			catalog: fixture("nope.json"),
			out:     "x.yaml",
			errText: "catalog",
		},
		{
			name:    "a slot holding nothing, which is not a rig",
			at:      slotpkg.Address{Slot: 2},
			catalog: catalogPath(),
			out:     "x.yaml",
			errText: "chain minimum number of items is 1",
		},
		{
			name:    "a caller who stopped waiting",
			ctx:     cancelled(),
			errText: context.Canceled.Error(),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			dir := s.T().TempDir()

			out := filepath.Join(dir, "x.hlx")
			if tt.out != "" {
				out = filepath.Join(dir, tt.out)
			}

			path := fixture("setlist.hls")
			if tt.path != "" {
				path = tt.path
			}

			written, err := flows(s.T(), tt.catalog).
				Export(background(tt.ctx), path, tt.at, out, formatFor(tt.as), result.ReplaceExisting)

			if tt.errText != "" {
				s.Require().ErrorContains(err, tt.errText)

				return
			}

			s.Require().NoError(err)
			s.Require().Equal(out, written.Path)

			if tt.wrote != nil {
				raw, err := os.ReadFile(out) //nolint:gosec // a path this test chose
				s.Require().NoError(err)

				for _, want := range tt.wrote {
					s.Require().Contains(string(raw), want)
				}
			}

			if tt.shows != nil {
				read, err := flows(s.T(), catalogPath()).ShowFile(context.Background(), out)
				s.Require().NoError(err)

				for _, want := range tt.shows {
					s.Require().Contains(said(s.T(), read), want)
				}
			}

			if tt.named != "" {
				s.Require().Equal(tt.named, written.Name)
			}
		})
	}
}

// TestImport covers putting a preset file into one slot of a file.
func (s *EditPublicTestSuite) TestImport() {
	tests := []struct {
		name string
		ctx  context.Context
		path string
		file string
		at   slotpkg.Address
		out  string
		// export slot 0 first and import that, rather than a fixture.
		exported bool

		// what the destination slot must hold afterwards.
		want string
		// the preset was made for another device, so the write says so.
		mismatch bool
		contains []string
		errText  string
	}{
		{
			name:     "a preset this setlist itself wrote",
			exported: true,
			at:       slotpkg.Address{Slot: 1},
			want:     "First",
			contains: []string{"Second"},
		},
		{
			name:     "a preset from another device",
			file:     fixture("otherdevice.hlx"),
			at:       slotpkg.Address{Slot: 1},
			mismatch: true,
		},
		{name: "a setlist that is not there", path: fixture("nope.hls"), errText: "opening"},
		{name: "a preset that is not there", file: fixture("nope.hlx"), errText: "opening"},
		{
			name:    "a file that is not a preset",
			file:    fixture("notapreset.hlx"),
			errText: "not a preset",
		},
		{name: "a slot that is not there", at: slotpkg.Address{Slot: 99}, errText: "no such slot"},
		{
			// The setlist half of the address is read too.
			name:    "a setlist the file does not hold",
			at:      slotpkg.Address{Setlist: 5},
			errText: "no such slot",
		},
		{
			name:    "a destination directory that is not there",
			out:     filepath.Join("no", "o.hls"),
			errText: "writing",
		},
		{
			name:    "a caller who stopped waiting",
			ctx:     cancelled(),
			errText: context.Canceled.Error(),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			dir := s.T().TempDir()

			out := filepath.Join(dir, "out.hls")
			if tt.out != "" {
				out = filepath.Join(dir, tt.out)
			}

			file := tt.file
			if file == "" {
				file = fixture("preset.hlx")
			}

			if tt.exported {
				file = filepath.Join(dir, "one.hlx")
				_, err := (&fileslots.Flows{}).Export(context.Background(),
					fixture("setlist.hls"), slotpkg.Address{}, file, result.FormatPreset,
					result.ReplaceExisting)
				s.Require().NoError(err)
			}

			path := fixture("setlist.hls")
			if tt.path != "" {
				path = tt.path
			}

			change, err := (&fileslots.Flows{}).Import(background(tt.ctx), path, file, tt.at, out)

			if tt.errText != "" {
				s.Require().ErrorContains(err, tt.errText)

				return
			}

			s.Require().NoError(err)
			s.Require().Equal(tt.mismatch, change.Mismatch)

			if tt.want != "" {
				s.Require().Equal(tt.want,
					s.reread(out).Setlists[tt.at.Setlist].Slots[tt.at.Slot].Meta.Name)
			}

			for _, want := range tt.contains {
				s.Require().Contains(did(change), want)
			}
		})
	}
}

func TestEditPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(EditPublicTestSuite))
}
