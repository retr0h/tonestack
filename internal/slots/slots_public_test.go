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
	"errors"
	"io"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/internal/catalogview"
	"github.com/retr0h/tonestack/internal/cli"
	"github.com/retr0h/tonestack/internal/slots"
)

type SlotsPublicTestSuite struct {
	suite.Suite
}

func fixture(name string) string { return filepath.Join("testdata", name) }

func catalogPath() string { return fixture("catalog.json") }

// TestList prints what a setlist holds.
func (s *SlotsPublicTestSuite) TestList() {
	tests := []struct {
		name     string
		opts     slots.ListOptions
		contains []string
		absent   []string
		errText  string
	}{
		{
			name: "the slots in use",
			opts: slots.ListOptions{
				Path: fixture("setlist.hls"), CatalogPath: catalogPath(),
			},
			contains: []string{"Test Setlist", "4 slots · 3 in use", "01A", "First"},
			// The empty slot is hidden unless asked for.
			absent: []string{"New Preset"},
		},
		{
			name: "every slot, when somebody asks",
			opts: slots.ListOptions{
				Path: fixture("setlist.hls"), CatalogPath: catalogPath(), All: true,
			},
			// A bank holds three, so the fourth slot opens the second bank.
			// This file labelled slots in banks of four while the flag that
			// addresses them parsed banks of three, so `--slot 04A` came
			// back as `03B`.
			contains: []string{"New Preset", "01C", "02A"},
			absent:   []string{"01D"},
		},
		{
			name: "one setlist out of a bundle",
			opts: slots.ListOptions{
				Path: fixture("bundle.hlb"), Setlist: 1, CatalogPath: catalogPath(),
			},
			contains: []string{"Second Setlist", "Only"},
		},
		{
			name: "a setlist holding nothing",
			opts: slots.ListOptions{
				Path: fixture("broken.hls"), CatalogPath: catalogPath(),
			},
			contains: []string{"no presets"},
		},
		{
			name: "a file that is not there",
			opts: slots.ListOptions{
				Path: fixture("nope.hls"), CatalogPath: catalogPath(),
			},
			errText: "opening",
		},
		{
			name: "a file that is not a setlist",
			opts: slots.ListOptions{
				Path: fixture("notasetlist.hls"), CatalogPath: catalogPath(),
			},
			errText: "not a setlist",
		},
		{
			name: "a setlist that is not there",
			opts: slots.ListOptions{
				Path: fixture("setlist.hls"), Setlist: 9, CatalogPath: catalogPath(),
			},
			errText: "no such slot",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			listing, err := slots.List(tt.opts)

			if tt.errText != "" {
				s.Require().Error(err)
				s.Require().Contains(err.Error(), tt.errText)

				return
			}

			s.Require().NoError(err)

			// Rendered the way the command renders it, because what a reader
			// sees is the renderer's answer rather than the operation's.
			var out bytes.Buffer

			cat, err := catalogview.Open(tt.opts.CatalogPath)
			s.Require().NoError(err)
			s.Require().NoError(cli.Listing(&out, listing, cat, tt.opts.All))

			for _, want := range tt.contains {
				s.Require().Contains(out.String(), want)
			}

			for _, unwanted := range tt.absent {
				s.Require().NotContains(out.String(), unwanted)
			}
		})
	}
}

// TestListNeedsNoCatalogToRead covers what the split moved.
//
// Reading a setlist says which blocks are in it, and a catalog is what turns
// a block into a name. So a catalog nobody can open is a rendering failure
// now rather than a reading one, which is the right place for it: a TUI
// holding a listing has already read the file.
func (s *SlotsPublicTestSuite) TestListNeedsNoCatalogToRead() {
	listing, err := slots.List(slots.ListOptions{Path: fixture("setlist.hls")})
	s.Require().NoError(err)
	s.Require().NotEmpty(listing.Slots)

	_, err = catalogview.Open(fixture("nope.json"))
	s.Require().ErrorContains(err, "catalog")
}

// TestListReportsAWriterThatFails covers a listing nobody can read.
//
// The failure belongs to the renderer now rather than to the operation, which
// is the point of the split: reading a setlist cannot fail because somebody's
// terminal went away.
func (s *SlotsPublicTestSuite) TestListReportsAWriterThatFails() {
	tests := []struct {
		name string
		path string
		w    io.Writer
	}{
		{name: "on the header", path: "setlist.hls", w: &failingWriter{}},
		{name: "on the rows", path: "setlist.hls", w: &oneGoodWrite{}},
		{name: "with no presets to show", path: "broken.hls", w: &failingWriter{}},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			listing, err := slots.List(slots.ListOptions{
				Path: fixture(tt.path), CatalogPath: catalogPath(),
			})
			s.Require().NoError(err)

			cat, err := catalogview.Open(catalogPath())
			s.Require().NoError(err)

			s.Require().Error(cli.Listing(tt.w, listing, cat, false))
		})
	}
}

// TestShow prints one slot as a rig.
func (s *SlotsPublicTestSuite) TestShow() {
	tests := []struct {
		name     string
		opts     slots.ShowOptions
		contains []string
		errText  string
	}{
		{
			name: "a slot in a setlist",
			opts: slots.ShowOptions{
				Path: fixture("setlist.hls"), Slot: 0, CatalogPath: catalogPath(),
			},
			contains: []string{"name: First", "gear: Ampeg SVT", "schema: RigSpec"},
		},
		{
			// A rig, because a rig is what this project reads and writes.
			// What comes out here is what compiles back into the preset it
			// came from.
			name: "a preset in a file of its own",
			opts: slots.ShowOptions{
				File: fixture("preset.hlx"), CatalogPath: catalogPath(),
			},
			contains: []string{"schema: RigSpec", "gear: Ampeg SVT"},
		},
		{
			name: "a slot holding nothing",
			opts: slots.ShowOptions{
				Path: fixture("setlist.hls"), Slot: 2, CatalogPath: catalogPath(),
			},
			contains: []string{"empty"},
		},
		{
			name: "gear the catalog cannot name",
			opts: slots.ShowOptions{
				Path: fixture("setlist.hls"), Slot: 3, CatalogPath: catalogPath(),
			},
			// It is written as the identifier the preset carried, so the rig
			// still rebuilds it exactly rather than dropping it.
			contains: []string{
				"gear: HD2_NotInCatalog",
				"HX Stomp: HD2_NotInCatalog",
			},
		},
		{
			name: "a setlist that is not there",
			opts: slots.ShowOptions{
				Path: fixture("nope.hls"), CatalogPath: catalogPath(),
			},
			errText: "opening",
		},
		{
			name: "a preset that is not there",
			opts: slots.ShowOptions{
				File: fixture("nope.hlx"), CatalogPath: catalogPath(),
			},
			errText: "opening",
		},
		{
			name: "a file that is not a preset",
			opts: slots.ShowOptions{
				File: fixture("notapreset.hlx"), CatalogPath: catalogPath(),
			},
			errText: "not a preset",
		},
		{
			name: "a slot that is not there",
			opts: slots.ShowOptions{
				Path: fixture("setlist.hls"), Slot: 99, CatalogPath: catalogPath(),
			},
			errText: "no such slot",
		},
		{
			name: "a slot that will not parse",
			opts: slots.ShowOptions{
				Path: fixture("broken.hls"), CatalogPath: catalogPath(),
			},
			errText: "reading slot 0",
		},
		{
			name: "a catalog that is not there",
			opts: slots.ShowOptions{
				Path: fixture("setlist.hls"), CatalogPath: fixture("nope.json"),
			},
			errText: "catalog",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var out bytes.Buffer

			err := slots.Show(&out, tt.opts)

			if tt.errText != "" {
				s.Require().Error(err)
				s.Require().Contains(err.Error(), tt.errText)

				return
			}

			s.Require().NoError(err)

			for _, want := range tt.contains {
				s.Require().Contains(out.String(), want)
			}
		})
	}
}

// TestShowReportsAWriterThatFails covers a rig nobody can read.
func (s *SlotsPublicTestSuite) TestShowReportsAWriterThatFails() {
	tests := []struct {
		name string
		slot int
	}{
		{name: "on the rig"},
		{name: "on an empty slot", slot: 2},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().Error(slots.Show(&failingWriter{}, slots.ShowOptions{
				Path: fixture("setlist.hls"), Slot: tt.slot, CatalogPath: catalogPath(),
			}))
		})
	}
}

type failingWriter struct{}

func (*failingWriter) Write([]byte) (int, error) { return 0, errors.New("boom") }

// oneGoodWrite fails only after the first write, so a caller that writes a
// header before its body reports the body's failure rather than the header's.
type oneGoodWrite struct{ n int }

func (w *oneGoodWrite) Write(p []byte) (int, error) {
	w.n++
	if w.n > 1 {
		return 0, errors.New("boom")
	}

	return len(p), nil
}

func TestSlotsPublicTestSuite(t *testing.T) {
	suite.Run(t, new(SlotsPublicTestSuite))
}
