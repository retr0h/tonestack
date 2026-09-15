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
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/internal/slots"
	slotmocks "github.com/retr0h/tonestack/pkg/sdk/internal/slots/mocks"
	"github.com/retr0h/tonestack/pkg/sdk/result"
	"github.com/retr0h/tonestack/pkg/sdk/rig"
	slotpkg "github.com/retr0h/tonestack/pkg/sdk/slot"
)

type SlotsPublicTestSuite struct {
	suite.Suite
}

func fixture(
	name string,
) string {
	return filepath.Join("testdata", name)
}

func catalogPath() string { return fixture("catalog.json") }

// cancelled is a context whose caller has already stopped waiting.
func cancelled() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	return ctx
}

// flows are the operations, naming gear against the catalog at path. An empty
// path is the catalog built into this binary.
//
// The catalog comes from a generated double that opens the file whenever it is
// asked, which is what the sdk Client hands over after its first open.
func flows(
	t *testing.T,
	path string,
) *slots.Flows {
	t.Helper()

	c := slotmocks.NewMockCatalogs(gomock.NewController(t))
	c.EXPECT().Catalog(gomock.Any()).DoAndReturn(
		func(context.Context) (*catalog.Catalog, error) { return catalog.Open(path) },
	).AnyTimes()

	return &slots.Flows{Catalogs: c}
}

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

// said renders a reading the way something displaying one would.
//
// The assertions here are about what was read, and what was read is a rig.
// Rendering it in the test rather than importing the one renderer keeps these
// operations free of anything that knows what a terminal is.
func said(
	t *testing.T,
	r result.Reading,
) string {
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

// TestList covers what one setlist of a file holds.
func (s *SlotsPublicTestSuite) TestList() {
	tests := []struct {
		name    string
		ctx     context.Context
		path    string
		setlist int
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
			name:    "what a setlist holds",
			path:    fixture("setlist.hls"),
			called:  "Test Setlist",
			slots:   4,
			used:    3,
			holding: []string{"First", "New Preset"},
		},
		{
			name:    "one setlist out of a bundle",
			path:    fixture("bundle.hlb"),
			setlist: 1,
			called:  "Second Setlist",
			slots:   1,
			used:    1,
			holding: []string{"Only"},
		},
		{
			// A slot that is there and holds no chain, which is a different
			// thing from a setlist with no slots in it.
			name:  "a setlist whose slots hold nothing",
			path:  fixture("broken.hls"),
			slots: 1,
		},
		{name: "a file that is not there", path: fixture("nope.hls"), errText: "opening"},
		{
			// A file that opens and is not a setlist fails at the read
			// rather than at the open, and both have to be reported.
			name:    "a file that is not a setlist",
			path:    fixture("notasetlist.hls"),
			errText: "reading",
		},
		{
			name:    "a setlist a bundle does not hold",
			path:    fixture("setlist.hls"),
			setlist: 9,
			errText: "no such slot",
		},
		{
			name:    "a caller who stopped waiting",
			ctx:     cancelled(),
			path:    fixture("setlist.hls"),
			errText: context.Canceled.Error(),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctx := tt.ctx
			if ctx == nil {
				ctx = context.Background()
			}

			// No expectation on the catalog. A listing says which blocks a
			// slot holds and naming them is the renderer's, so reaching for a
			// catalog fails the row.
			f := &slots.Flows{Catalogs: slotmocks.NewMockCatalogs(gomock.NewController(s.T()))}

			listing, err := f.List(ctx, tt.path, tt.setlist)

			if tt.errText != "" {
				s.Require().ErrorContains(err, tt.errText)

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

// TestShow covers reading one slot of a file as a rig.
func (s *SlotsPublicTestSuite) TestShow() {
	tests := []struct {
		name string
		ctx  context.Context
		path string
		at   slotpkg.Address
		// a catalog other than this suite's fixture.
		catalog string
		// the slot holds nothing, which is an answer and not a failure.
		empty    bool
		contains []string
		errText  string
	}{
		{
			name:     "a slot in a setlist",
			path:     fixture("setlist.hls"),
			contains: []string{"name: First", "gear: Ampeg SVT", "schema: RigSpec"},
		},
		{
			// The setlist half of the address is read too, not only the slot.
			name:     "a slot in the second setlist of a bundle",
			path:     fixture("bundle.hlb"),
			at:       slotpkg.Address{Setlist: 1},
			contains: []string{"Only"},
		},
		{
			// It names no gear, and a rig holds at least one thing. The slot
			// still has a name, which is what somebody looking at it is told.
			name:     "a slot holding nothing",
			path:     fixture("setlist.hls"),
			at:       slotpkg.Address{Slot: 2},
			empty:    true,
			contains: []string{"empty"},
		},
		{
			name: "gear the catalog cannot name",
			path: fixture("setlist.hls"),
			at:   slotpkg.Address{Slot: 3},
			// It is written as the identifier the preset carried, so the rig
			// still rebuilds it exactly rather than dropping it.
			contains: []string{"gear: HD2_NotInCatalog", "HX Stomp: HD2_NotInCatalog"},
		},
		{name: "a setlist that is not there", path: fixture("nope.hls"), errText: "opening"},
		{
			name:    "a slot that is not there",
			path:    fixture("setlist.hls"),
			at:      slotpkg.Address{Slot: 99},
			errText: "no such slot",
		},
		{
			name:    "a slot that will not parse",
			path:    fixture("broken.hls"),
			errText: "reading slot 01A",
		},
		{
			name:    "a catalog that is not there",
			path:    fixture("setlist.hls"),
			catalog: fixture("nope.json"),
			errText: "catalog",
		},
		{
			name:    "a caller who stopped waiting",
			ctx:     cancelled(),
			path:    fixture("setlist.hls"),
			errText: context.Canceled.Error(),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctx := tt.ctx
			if ctx == nil {
				ctx = context.Background()
			}

			cat := catalogPath()
			if tt.catalog != "" {
				cat = tt.catalog
			}

			read, err := flows(s.T(), cat).Show(ctx, tt.path, tt.at)

			if tt.errText != "" {
				s.Require().ErrorContains(err, tt.errText)

				return
			}

			s.Require().NoError(err)
			s.Require().Equal(tt.empty, read.Empty())
			s.Require().NotEmpty(read.Name)

			got := said(s.T(), read)

			for _, want := range tt.contains {
				s.Require().Contains(got, want)
			}
		})
	}
}

// TestShowFile covers reading a standalone preset as a rig.
func (s *SlotsPublicTestSuite) TestShowFile() {
	tests := []struct {
		name     string
		ctx      context.Context
		file     string
		catalog  string
		contains []string
		errText  string
	}{
		{
			// A rig, because a rig is what this project reads and writes.
			// What comes out here is what compiles back into the preset it
			// came from.
			name:     "a preset in a file of its own",
			file:     fixture("preset.hlx"),
			contains: []string{"schema: RigSpec", "gear: Ampeg SVT"},
		},
		{name: "a preset that is not there", file: fixture("nope.hlx"), errText: "opening"},
		{
			name:    "a file that is not a preset",
			file:    fixture("notapreset.hlx"),
			errText: "not a preset",
		},
		{
			name:    "a catalog that is not there",
			file:    fixture("preset.hlx"),
			catalog: fixture("nope.json"),
			errText: "catalog",
		},
		{
			name:    "a caller who stopped waiting",
			ctx:     cancelled(),
			file:    fixture("preset.hlx"),
			errText: context.Canceled.Error(),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctx := tt.ctx
			if ctx == nil {
				ctx = context.Background()
			}

			cat := catalogPath()
			if tt.catalog != "" {
				cat = tt.catalog
			}

			read, err := flows(s.T(), cat).ShowFile(ctx, tt.file)

			if tt.errText != "" {
				s.Require().ErrorContains(err, tt.errText)

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

func TestSlotsPublicTestSuite(t *testing.T) {
	suite.Run(t, new(SlotsPublicTestSuite))
}
