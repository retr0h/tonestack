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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/internal/slots"
	"github.com/retr0h/tonestack/pkg/setlist"
)

type SlotsPublicTestSuite struct {
	suite.Suite
}

func fixture(name string) string { return filepath.Join("testdata", name) }

func catalogPath() string { return fixture("catalog.json") }

func (s *SlotsPublicTestSuite) TestListShowsWhatIsInUse() {
	var out bytes.Buffer

	s.Require().NoError(slots.List(&out, slots.ListOptions{
		Path: fixture("setlist.hls"), CatalogPath: catalogPath(),
	}))

	s.Require().Contains(out.String(), "Test Setlist")
	s.Require().Contains(out.String(), "4 slots · 3 in use")
	s.Require().Contains(out.String(), "01A")
	s.Require().Contains(out.String(), "First")
	// The empty slot is hidden unless asked for.
	s.Require().NotContains(out.String(), "New Preset")
}

func (s *SlotsPublicTestSuite) TestListShowsEveryySlotWhenAsked() {
	var out bytes.Buffer

	s.Require().NoError(slots.List(&out, slots.ListOptions{
		Path: fixture("setlist.hls"), CatalogPath: catalogPath(), All: true,
	}))

	s.Require().Contains(out.String(), "New Preset")
}

func (s *SlotsPublicTestSuite) TestListReachesIntoABundle() {
	var out bytes.Buffer

	s.Require().NoError(slots.List(&out, slots.ListOptions{
		Path: fixture("bundle.hlb"), Setlist: 1, CatalogPath: catalogPath(),
	}))

	s.Require().Contains(out.String(), "Second Setlist")
	s.Require().Contains(out.String(), "Only")
}

func (s *SlotsPublicTestSuite) TestListSaysWhenNothingIsThere() {
	var out bytes.Buffer

	s.Require().NoError(slots.List(&out, slots.ListOptions{
		Path: fixture("broken.hls"), CatalogPath: catalogPath(),
	}))

	s.Require().Contains(out.String(), "no presets")
}

func (s *SlotsPublicTestSuite) TestListReportsProblems() {
	tests := []struct {
		name    string
		opts    slots.ListOptions
		message string
	}{
		{
			"a file that is not there",
			slots.ListOptions{Path: fixture("nope.hls"), CatalogPath: catalogPath()},
			"opening",
		},
		{
			"a file that is not a setlist",
			slots.ListOptions{Path: fixture("notasetlist.hls"), CatalogPath: catalogPath()},
			"not a setlist",
		},
		{
			"a catalog that is not there",
			slots.ListOptions{Path: fixture("setlist.hls"), CatalogPath: fixture("nope.json")},
			"catalog",
		},
		{
			"a setlist that is not there",
			slots.ListOptions{
				Path: fixture("setlist.hls"), Setlist: 9, CatalogPath: catalogPath(),
			},
			"no such slot",
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			err := slots.List(&bytes.Buffer{}, tc.opts)

			s.Require().Error(err)
			s.Require().Contains(err.Error(), tc.message)
		})
	}
}

func (s *SlotsPublicTestSuite) TestListReportsAWriterThatFails() {
	tests := []struct {
		name string
		w    interface{ Write([]byte) (int, error) }
	}{
		{"on the header", &failingWriter{}},
		{"on the rows", &oneGoodWrite{}},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			err := slots.List(tc.w, slots.ListOptions{
				Path: fixture("setlist.hls"), CatalogPath: catalogPath(),
			})

			s.Require().Error(err)
		})
	}
}

func (s *SlotsPublicTestSuite) TestListReportsAWriterThatFailsWithNoPresets() {
	err := slots.List(&failingWriter{}, slots.ListOptions{
		Path: fixture("broken.hls"), CatalogPath: catalogPath(),
	})

	s.Require().Error(err)
}

func (s *SlotsPublicTestSuite) TestShowRendersASlot() {
	var out bytes.Buffer

	s.Require().NoError(slots.Show(&out, slots.ShowOptions{
		Path: fixture("setlist.hls"), Slot: 0, CatalogPath: catalogPath(),
	}))

	s.Require().Contains(out.String(), "name: First")
	s.Require().Contains(out.String(), "gear: Ampeg SVT")
	s.Require().Contains(out.String(), "schema: RigSpec")
}

func (s *SlotsPublicTestSuite) TestShowRendersAFile() {
	var out bytes.Buffer

	s.Require().NoError(slots.Show(&out, slots.ShowOptions{
		File: fixture("preset.hlx"), CatalogPath: catalogPath(),
	}))

	// A rig, because a rig is what this project reads and writes. What comes
	// out here is what compiles back into the preset it came from.
	s.Require().Contains(out.String(), "schema: RigSpec")
	s.Require().Contains(out.String(), "gear: Ampeg SVT")
}

func (s *SlotsPublicTestSuite) TestShowSaysWhenASlotIsEmpty() {
	var out bytes.Buffer

	s.Require().NoError(slots.Show(&out, slots.ShowOptions{
		Path: fixture("setlist.hls"), Slot: 2, CatalogPath: catalogPath(),
	}))

	s.Require().Contains(out.String(), "empty")
}

func (s *SlotsPublicTestSuite) TestShowMarksGearTheCatalogDoesNotKnow() {
	var out bytes.Buffer

	s.Require().NoError(slots.Show(&out, slots.ShowOptions{
		Path: fixture("setlist.hls"), Slot: 3, CatalogPath: catalogPath(),
	}))

	// Gear the catalog cannot name is written as the identifier the preset
	// carried, so the rig still rebuilds it exactly rather than dropping it.
	s.Require().Contains(out.String(), "gear: HD2_NotInCatalog")
	s.Require().Contains(out.String(), "HX Stomp: HD2_NotInCatalog")
}

func (s *SlotsPublicTestSuite) TestShowReportsProblems() {
	tests := []struct {
		name    string
		opts    slots.ShowOptions
		message string
	}{
		{
			"a setlist that is not there",
			slots.ShowOptions{Path: fixture("nope.hls"), CatalogPath: catalogPath()},
			"opening",
		},
		{
			"a preset that is not there",
			slots.ShowOptions{File: fixture("nope.hlx"), CatalogPath: catalogPath()},
			"opening",
		},
		{
			"a file that is not a preset",
			slots.ShowOptions{File: fixture("notapreset.hlx"), CatalogPath: catalogPath()},
			"not a preset",
		},
		{
			"a slot that is not there",
			slots.ShowOptions{
				Path: fixture("setlist.hls"), Slot: 99, CatalogPath: catalogPath(),
			},
			"no such slot",
		},
		{
			"a slot that will not parse",
			slots.ShowOptions{Path: fixture("broken.hls"), CatalogPath: catalogPath()},
			"reading slot 0",
		},
		{
			"a catalog that is not there",
			slots.ShowOptions{Path: fixture("setlist.hls"), CatalogPath: fixture("nope.json")},
			"catalog",
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			err := slots.Show(&bytes.Buffer{}, tc.opts)

			s.Require().Error(err)
			s.Require().Contains(err.Error(), tc.message)
		})
	}
}

func (s *SlotsPublicTestSuite) TestShowReportsAWriterThatFails() {
	tests := []struct {
		name string
		slot int
		w    interface{ Write([]byte) (int, error) }
	}{
		{"on the rig", 0, &failingWriter{}},
		{"on an empty slot", 2, &failingWriter{}},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			err := slots.Show(tc.w, slots.ShowOptions{
				Path: fixture("setlist.hls"), Slot: tc.slot, CatalogPath: catalogPath(),
			})

			s.Require().Error(err)
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

var (
	_ = os.Open
	_ = setlist.ErrNoSuchSlot
)
