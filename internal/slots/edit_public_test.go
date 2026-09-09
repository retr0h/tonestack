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

package slots_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/internal/slots"
	"github.com/retr0h/tonestack/pkg/setlist"
)

type EditPublicTestSuite struct {
	suite.Suite
}

// reread loads a setlist this suite just wrote.
func (s *EditPublicTestSuite) reread(path string) *setlist.Document {
	f, err := os.Open(path) //nolint:gosec // a path this test chose
	s.Require().NoError(err)

	defer func() { s.Require().NoError(f.Close()) }()

	doc, err := setlist.Read(f)
	s.Require().NoError(err)

	return doc
}

func (s *EditPublicTestSuite) opts(out string) slots.EditOptions {
	return slots.EditOptions{
		Path: fixture("setlist.hls"), FromSlot: 0, ToSlot: 1, OutputPath: out,
	}
}

// TestCopy writes one slot over another.
func (s *EditPublicTestSuite) TestCopy() {
	tests := []struct {
		name string
		path string
		from int
		to   int
		// where to write, under this case's own directory.
		out string
		// a writer that fails, so a report nobody can read is an error.
		deaf bool

		// what the two slots must hold afterwards.
		want     []string
		contains []string
		errText  string
	}{
		{
			name:     "a copy, which leaves the source alone",
			to:       1,
			want:     []string{"First", "First"},
			contains: []string{"copied", "01A", "01B"},
		},
		{
			name:    "a file that is not there",
			path:    fixture("nope.hls"),
			to:      1,
			errText: "opening",
		},
		{name: "a source that is not there", from: 99, to: 1, errText: "no such slot"},
		{name: "a destination that is not there", to: 99, errText: "no such slot"},
		{
			name:    "a destination directory that is not there",
			to:      1,
			out:     filepath.Join("no", "out.hls"),
			errText: "writing",
		},
		{name: "a writer that fails", to: 1, deaf: true},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			dir := s.T().TempDir()

			out := filepath.Join(dir, "out.hls")
			if tt.out != "" {
				out = filepath.Join(dir, tt.out)
			}

			o := s.opts(out)
			o.FromSlot, o.ToSlot = tt.from, tt.to

			if tt.path != "" {
				o.Path = tt.path
			}

			var log bytes.Buffer

			w := io.Writer(&log)
			if tt.deaf {
				w = &failingWriter{}
			}

			err := slots.Copy(w, o)

			if tt.errText != "" || tt.deaf {
				s.Require().Error(err)

				if tt.errText != "" {
					s.Require().Contains(err.Error(), tt.errText)
				}

				return
			}

			s.Require().NoError(err)

			doc := s.reread(out)
			for i, want := range tt.want {
				s.Require().Equal(want, doc.Setlists[0].Slots[i].Meta.Name)
			}

			for _, want := range tt.contains {
				s.Require().Contains(log.String(), want)
			}
		})
	}
}

// TestSwap exchanges two slots.
func (s *EditPublicTestSuite) TestSwap() {
	out := filepath.Join(s.T().TempDir(), "out.hls")

	var log bytes.Buffer
	s.Require().NoError(slots.Swap(&log, s.opts(out)))

	doc := s.reread(out)
	s.Require().Equal("Second", doc.Setlists[0].Slots[0].Meta.Name)
	s.Require().Equal("First", doc.Setlists[0].Slots[1].Meta.Name)
	s.Require().Contains(log.String(), "swapped")
}

// TestExport writes one slot to a file of its own.
func (s *EditPublicTestSuite) TestExport() {
	tests := []struct {
		name    string
		path    string
		slot    int
		as      slots.Format
		catalog string
		out     string
		deaf    bool

		// what the written file must say.
		wrote []string
		// what showing the written file must say, for the device's own
		// format.
		shows []string
		// what the report must say.
		logs    []string
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
			as:    slots.FormatPreset,
			out:   "one.hlx",
			shows: []string{"First", "Ampeg SVT"},
			logs:  []string{"wrote"},
		},
		{
			name:    "a file that is not there",
			path:    fixture("nope.hls"),
			errText: "opening",
		},
		{
			name:    "a slot that is not there",
			slot:    99,
			as:      slots.FormatPreset,
			errText: "no such slot",
		},
		{
			name:    "a destination directory that is not there",
			as:      slots.FormatPreset,
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
			slot:    2,
			catalog: catalogPath(),
			out:     "x.yaml",
			errText: "chain minimum number of items is 1",
		},
		{name: "a writer that fails", as: slots.FormatPreset, deaf: true},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			dir := s.T().TempDir()

			out := filepath.Join(dir, "x.hlx")
			if tt.out != "" {
				out = filepath.Join(dir, tt.out)
			}

			o := slots.ExportOptions{
				Path: fixture("setlist.hls"), Slot: tt.slot, OutputPath: out,
				As: tt.as, CatalogPath: tt.catalog,
			}
			if tt.path != "" {
				o.Path = tt.path
			}

			var log bytes.Buffer

			w := io.Writer(&log)
			if tt.deaf {
				w = &failingWriter{}
			}

			err := slots.Export(w, o)

			if tt.errText != "" || tt.deaf {
				s.Require().Error(err)

				if tt.errText != "" {
					s.Require().Contains(err.Error(), tt.errText)
				}

				return
			}

			s.Require().NoError(err)

			if tt.wrote != nil {
				raw, err := os.ReadFile(out) //nolint:gosec // a path this test chose
				s.Require().NoError(err)

				for _, want := range tt.wrote {
					s.Require().Contains(string(raw), want)
				}
			}

			if tt.shows != nil {
				var show bytes.Buffer
				s.Require().NoError(slots.Show(&show, slots.ShowOptions{
					File: out, CatalogPath: catalogPath(),
				}))

				for _, want := range tt.shows {
					s.Require().Contains(show.String(), want)
				}
			}

			for _, want := range tt.logs {
				s.Require().Contains(log.String(), want)
			}
		})
	}
}

// TestImport puts a preset file into a slot.
func (s *EditPublicTestSuite) TestImport() {
	tests := []struct {
		name string
		path string
		file string
		slot int
		out  string
		// export slot 0 first and import that, rather than a fixture.
		exported bool
		deaf     bool
		// a writer that takes one write before failing, for a report with a
		// warning above it.
		partial bool

		// what the destination slot must hold afterwards.
		want     string
		contains []string
		errText  string
	}{
		{
			name:     "a preset this setlist itself wrote",
			exported: true,
			slot:     1,
			want:     "First",
			contains: []string{"replaced", "Second"},
		},
		{
			name:     "a preset from another device",
			file:     fixture("otherdevice.hlx"),
			slot:     1,
			contains: []string{"different device"},
		},
		{
			name:    "a setlist that is not there",
			path:    fixture("nope.hls"),
			errText: "opening",
		},
		{
			name:    "a preset that is not there",
			file:    fixture("nope.hlx"),
			errText: "opening",
		},
		{
			name:    "a file that is not a preset",
			file:    fixture("notapreset.hlx"),
			errText: "not a preset",
		},
		{name: "a slot that is not there", slot: 99, errText: "no such slot"},
		{
			name:    "a destination directory that is not there",
			out:     filepath.Join("no", "o.hls"),
			errText: "writing",
		},
		{name: "a writer that fails", slot: 1, deaf: true},
		{
			name: "a writer that fails on the warning",
			file: fixture("otherdevice.hlx"),
			slot: 1,
			deaf: true,
		},
		{
			name:    "a writer that fails after the warning",
			file:    fixture("otherdevice.hlx"),
			slot:    1,
			partial: true,
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
				s.Require().NoError(slots.Export(&bytes.Buffer{}, slots.ExportOptions{
					Path: fixture("setlist.hls"), Slot: 0, OutputPath: file,
					As: slots.FormatPreset,
				}))
			}

			o := slots.ImportOptions{
				Path: fixture("setlist.hls"), File: file,
				Slot: tt.slot, OutputPath: out,
			}
			if tt.path != "" {
				o.Path = tt.path
			}

			var log bytes.Buffer

			w := io.Writer(&log)

			switch {
			case tt.deaf:
				w = &failingWriter{}
			case tt.partial:
				w = &oneGoodWrite{}
			}

			err := slots.Import(w, o)

			if tt.errText != "" || tt.deaf || tt.partial {
				s.Require().Error(err)

				if tt.errText != "" {
					s.Require().Contains(err.Error(), tt.errText)
				}

				return
			}

			s.Require().NoError(err)

			if tt.want != "" {
				s.Require().Equal(
					tt.want, s.reread(out).Setlists[0].Slots[tt.slot].Meta.Name)
			}

			for _, want := range tt.contains {
				s.Require().Contains(log.String(), want)
			}
		})
	}
}

func TestEditPublicTestSuite(t *testing.T) {
	suite.Run(t, new(EditPublicTestSuite))
}
