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
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/internal/slots"
	"github.com/retr0h/tonestack/pkg/sdk/result"
	"github.com/retr0h/tonestack/pkg/sdk/rig"
	slotpkg "github.com/retr0h/tonestack/pkg/sdk/slot"
)

type SlotsPublicTestSuite struct {
	suite.Suite
}

func fixture(name string) string { return filepath.Join("testdata", name) }

func catalogPath() string { return fixture("catalog.json") }

// did flattens what a write reported, so a test can assert on the facts of it
// without also asserting on how a terminal paints them.
func did(c result.Change) string {
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

// said renders a reading the way something displaying one would.
//
// The assertions here are about what was read, and what was read is a rig.
// Rendering it in the test rather than importing the one renderer keeps these
// operations free of anything that knows what a terminal is.
func said(t *testing.T, r result.Reading) string {
	t.Helper()

	if r.Answer != nil {
		return r.Answer.Shape
	}

	if r.Empty() {
		return "# " + r.Name + " is empty"
	}

	var buf bytes.Buffer

	require.NoError(t, rig.Write(&buf, r.Rig))

	return r.Name + "\n" + buf.String()
}

// TestList prints what a setlist holds.
func (s *SlotsPublicTestSuite) TestList() {
	tests := []struct {
		name string
		opts slots.ListOptions
		// what the setlist is called, and how many slots it answers with.
		called  string
		slots   int
		used    int
		holding []string
		errText string
	}{
		{
			// Every slot, including the ones holding nothing. A device
			// answers for all of them either way, and which to show is the
			// renderer's decision rather than this one's.
			name: "what a setlist holds",
			opts: slots.ListOptions{
				Path: fixture("setlist.hls"), CatalogPath: catalogPath(),
			},
			called:  "Test Setlist",
			slots:   4,
			used:    3,
			holding: []string{"First", "New Preset"},
		},
		{
			name: "one setlist out of a bundle",
			opts: slots.ListOptions{
				Path: fixture("bundle.hlb"), Setlist: 1, CatalogPath: catalogPath(),
			},
			called:  "Second Setlist",
			slots:   1,
			used:    1,
			holding: []string{"Only"},
		},
		{
			// A slot that is there and holds no chain, which is a different
			// thing from a setlist with no slots in it.
			name: "a setlist whose slots hold nothing",
			opts: slots.ListOptions{
				Path: fixture("broken.hls"), CatalogPath: catalogPath(),
			},
			slots: 1,
			used:  0,
		},
		{
			name: "a file that is not there",
			opts: slots.ListOptions{
				Path: fixture("nope.hls"), CatalogPath: catalogPath(),
			},
			errText: "opening",
		},
		{
			// A file that opens and is not a setlist fails at the read
			// rather than at the open, and both have to be reported.
			name: "a file that is not a setlist",
			opts: slots.ListOptions{
				Path: fixture("notasetlist.hls"), CatalogPath: catalogPath(),
			},
			errText: "reading",
		},
		{
			name: "a setlist a bundle does not hold",
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
			s.Require().Len(listing.Slots, tt.slots)
			s.Require().Equal(tt.used, listing.Used())

			if tt.called != "" {
				s.Require().Equal(tt.called, listing.Name)
			}

			named := make([]string, 0, len(listing.Slots))
			for _, h := range listing.Slots {
				named = append(named, h.Name)
			}

			for _, want := range tt.holding {
				s.Require().Contains(named, want)
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

	_, err = catalog.Open(fixture("nope.json"))
	s.Require().ErrorContains(err, "catalog")
}

// TestShow reads one slot as a rig.
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
			read, err := slots.Show(tt.opts)

			if tt.errText != "" {
				s.Require().Error(err)
				s.Require().Contains(err.Error(), tt.errText)

				return
			}

			s.Require().NoError(err)

			got := said(s.T(), read)

			for _, want := range tt.contains {
				s.Require().Contains(got, want)
			}
		})
	}
}

// TestShowOnASlotHoldingNothing covers a slot that is not a rig.
//
// It names no gear, and a rig holds at least one thing. The slot still has a
// name, which is what somebody looking at it is told.
func (s *SlotsPublicTestSuite) TestShowOnASlotHoldingNothing() {
	read, err := slots.Show(slots.ShowOptions{
		Path: fixture("setlist.hls"), Slot: 2, CatalogPath: catalogPath(),
	})

	s.Require().NoError(err)
	s.Require().True(read.Empty())
	s.Require().NotEmpty(read.Name)
}

func TestSlotsPublicTestSuite(t *testing.T) {
	suite.Run(t, new(SlotsPublicTestSuite))
}
