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

package paint_test

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/cli/internal/paint"
	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/chain"
)

type ThemePublicTestSuite struct {
	suite.Suite
}

func (s *ThemePublicTestSuite) TearDownTest() {
	paint.SetTheme("tube")
}

// TestRoles covers every colour a theme names.
func (s *ThemePublicTestSuite) TestRoles() {
	var out bytes.Buffer

	tests := []struct {
		name string
		call func() string
	}{
		{"mute", func() string { return paint.Mute(&out, "x") }},
		{"accent", func() string { return paint.Accent(&out, "x") }},
		{"ok", func() string { return paint.OK(&out, "x") }},
		{"err", func() string { return paint.Err(&out, "x") }},
		{"info", func() string { return paint.Info(&out, "x") }},
		{"title", func() string { return paint.Title(&out, "x") }},
		{
			// A *os.File gets its own renderer so NO_COLOR and TTY detection
			// apply to the sink actually written to, not to stdout.
			"a sink with a renderer of its own",
			func() string { return paint.Mute(os.Stderr, "x") },
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().Contains(tt.call(), "x")
		})
	}
}

// TestHeading shouts a column name.
func (s *ThemePublicTestSuite) TestHeading() {
	s.Require().Equal("SLOT", paint.Heading(&bytes.Buffer{}, "slot"))
}

// TestBanner names the tool.
func (s *ThemePublicTestSuite) TestBanner() {
	got := paint.Banner(&bytes.Buffer{})

	s.Require().Len(strings.Split(strings.TrimRight(got, "\n"), "\n"), 2)
	// The E carries a middle bar, so it is not the same glyph as the C.
	s.Require().NotEqual(
		glyph(got, 3), glyph(got, 7),
		"E and C must be distinguishable")
}

// glyph returns the nth space-separated letter of a two-line banner.
func glyph(banner string, n int) string {
	lines := strings.Split(strings.TrimRight(banner, "\n"), "\n")

	return strings.Fields(lines[0])[n] + strings.Fields(lines[1])[n]
}

// TestSuccessAndFailure covers the two marks a command ends with. A buffer is
// not a terminal, so lipgloss emits no escapes and the marks degrade to words
// rather than vanishing.
func (s *ThemePublicTestSuite) TestSuccessAndFailure() {
	var out bytes.Buffer

	tests := []struct {
		name string
		got  string
		want string
	}{
		{name: "a command that worked", got: paint.Success(&out, "done"), want: "[ok] done"},
		{name: "one that did not", got: paint.Failure(&out, "broke"), want: "[err] broke"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.want, tt.got)
		})
	}
}

// TestFailurePrefix is the mark on its own, for an error to be printed
// behind.
func (s *ThemePublicTestSuite) TestFailurePrefix() {
	s.Require().Equal("[err]", paint.FailurePrefix(&bytes.Buffer{}))
}

// TestSetTheme picks a theme by name.
func (s *ThemePublicTestSuite) TestSetTheme() {
	tests := []struct {
		name  string
		theme string
		want  bool
	}{
		{name: "the default", theme: "tube", want: true},
		{name: "a different case", theme: "TUBE", want: true},
		{name: "surrounded by space", theme: "  tube  ", want: true},
		{name: "one nobody registered", theme: "chartreuse"},
		{name: "nothing at all"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.want, paint.SetTheme(tt.theme))
		})
	}
}

// TestActiveTheme names what is in use, and what could be.
func (s *ThemePublicTestSuite) TestActiveTheme() {
	s.Require().Equal("tube", paint.ActiveTheme().Name)
	s.Require().Equal([]string{"tube"}, paint.ThemeNames())
}

func TestThemePublicTestSuite(t *testing.T) {
	suite.Run(t, new(ThemePublicTestSuite))
}

type UIPublicTestSuite struct {
	suite.Suite
}

// TestSectionRender lays out a titled table.
func (s *UIPublicTestSuite) TestSectionRender() {
	tests := []struct {
		name     string
		section  paint.Section
		contains []string
		silent   bool
	}{
		{
			name: "a table under a title",
			section: paint.Section{
				Title:   "Songs",
				Detail:  "128 slots",
				Headers: []string{"slot", "name"},
				Rows:    [][]string{{"01A", "Claptone"}},
				Summary: "1 of 128",
			},
			contains: []string{"Songs", "128 slots", "SLOT", "01A", "1 of 128"},
		},
		{
			name:     "nothing to show, and something to say about it",
			section:  paint.Section{Title: "Songs", Empty: "no presets"},
			contains: []string{"no presets"},
		},
		{name: "nothing to say at all", section: paint.Section{}, silent: true},
		{
			name:     "rows with neither headers nor a summary",
			section:  paint.Section{Rows: [][]string{{"a", "b"}}},
			contains: []string{"a"},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var out bytes.Buffer

			s.Require().NoError(tt.section.Render(&out))

			if tt.silent {
				s.Require().Empty(out.String())

				return
			}

			for _, want := range tt.contains {
				s.Require().Contains(out.String(), want)
			}
		})
	}
}

// TestSectionRenderReportsAWriterThatFails covers a writer failing at each
// point a section writes.
func (s *UIPublicTestSuite) TestSectionRenderReportsAWriterThatFails() {
	full := paint.Section{
		Title: "t", Detail: "d",
		Headers: []string{"h"}, Rows: [][]string{{"r"}}, Summary: "s",
	}

	tests := []struct {
		name    string
		section paint.Section
		after   int
	}{
		{name: "on the title", section: full},
		{name: "on the headers", section: full, after: 1},
		{name: "on the rows", section: full, after: 2},
		{name: "on the summary", section: full, after: 3},
		{
			name:    "on what it says instead of rows",
			section: paint.Section{Title: "t", Empty: "none"},
			after:   1,
		},
		{
			name:    "on rows with no title above them",
			section: paint.Section{Rows: [][]string{{"r"}}},
			after:   1,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().Error(tt.section.Render(&failAfter{ok: tt.after}))
		})
	}
}

// TestDetailRender lays out a titled list of fields.
func (s *UIPublicTestSuite) TestDetailRender() {
	var out bytes.Buffer

	s.Require().NoError(paint.Detail{
		Title:    "Mike Dirnt",
		Subtitle: "mike-dirnt",
		Fields: []paint.Field{
			{Label: "amp", Value: "Ampeg SVT"},
			{Label: "cab", Value: "unknown", Muted: true},
		},
		Note: "unverified",
	}.Render(&out))

	for _, want := range []string{
		"Mike Dirnt", "mike-dirnt", "amp", "Ampeg SVT", "unknown", "unverified",
	} {
		s.Require().Contains(out.String(), want)
	}
}

// TestDetailRenderReportsAWriterThatFails covers a writer failing at each
// point a detail writes.
func (s *UIPublicTestSuite) TestDetailRenderReportsAWriterThatFails() {
	full := paint.Detail{
		Title: "t", Subtitle: "s",
		Fields: []paint.Field{{Label: "l", Value: "v"}}, Note: "n",
	}

	for i := range 4 {
		s.Run(fmt.Sprintf("after %d writes", i), func() {
			s.Require().Error(full.Render(&failAfter{ok: i}))
		})
	}
}

// TestTable lays rows out in columns.
func (s *UIPublicTestSuite) TestTable() {
	tests := []struct {
		name      string
		rows      [][]string
		align     []lipgloss.Position
		aligned   bool
		sameWidth bool
		contains  string
		silent    bool
	}{
		{
			name: "columns that line up",
			rows: [][]string{
				{"a", "1", "end"},
				{"bbbb", "22", "end"},
			},
			align:   []lipgloss.Position{lipgloss.Left, lipgloss.Right},
			aligned: true,
		},
		{
			// A styled cell is measured by what it shows rather than by the
			// escapes around it.
			name: "a cell somebody painted",
			rows: [][]string{
				{lipgloss.NewStyle().
					Foreground(lipgloss.Color("#ffa032")).Render("ab"), "x"},
				{"abcd", "y"},
			},
			sameWidth: true,
		},
		{
			name:     "rows of different lengths",
			rows:     [][]string{{"a"}, {"bb", "cc"}},
			contains: "cc",
		},
		{name: "no rows at all", silent: true},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var out bytes.Buffer

			s.Require().NoError(paint.Table(&out, tt.rows, tt.align))

			if tt.silent {
				s.Require().Empty(out.String())

				return
			}

			lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")

			if tt.contains != "" {
				s.Require().Contains(out.String(), tt.contains)
			}

			if tt.sameWidth {
				s.Require().Equal(lipgloss.Width(lines[0]), lipgloss.Width(lines[1]))
			}

			if !tt.aligned {
				return
			}

			s.Require().Len(lines, 2)

			// A right-aligned column ends at the same offset on both lines.
			s.Require().Equal(
				strings.Index(lines[0], "1")+1, strings.Index(lines[1], "22")+2)

			// The last column starts at the same offset on both lines.
			s.Require().Equal(
				strings.Index(lines[0], "end"), strings.Index(lines[1], "end"))

			// Nothing trails a line.
			for _, l := range lines {
				s.Require().Equal(l, strings.TrimRight(l, " "))
			}
		})
	}
}

// TestTableReportsAWriterThatFails covers a row nobody can read.
func (s *UIPublicTestSuite) TestTableReportsAWriterThatFails() {
	err := paint.Table(&failAfter{}, [][]string{{"a"}}, nil)

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "writing row")
}

func TestUIPublicTestSuite(t *testing.T) {
	suite.Run(t, new(UIPublicTestSuite))
}

type ChainPublicTestSuite struct {
	suite.Suite
}

func (s *ChainPublicTestSuite) cat() *catalog.Catalog {
	return &catalog.Catalog{Blocks: map[catalog.ModelID]catalog.Block{
		"amp": {
			Name: "Test Amp", Category: catalog.CategoryAmp,
			BasedOn: "Some Amp", DSP: catalog.DSPCost{Mono: 25},
		},
		"cab": {
			Name: "Test Cab", Category: catalog.CategoryCab,
			DSP: catalog.DSPCost{Mono: 5},
		},
		"weird": {
			Name: "Utility", Category: catalog.Category("nothing"),
			DSP: catalog.DSPCost{Mono: 1},
		},
		"HD2_ImpulseResponse1024": {
			Name: "IR 1024", Category: catalog.CategoryCab,
			DSP: catalog.DSPCost{Mono: 7},
		},
	}}
}

func (s *ChainPublicTestSuite) spec(blocks ...chain.Block) chain.Chain {
	return chain.Chain{Name: "Test", Blocks: blocks}
}

// TestChain draws a chain and what it costs.
func (s *ChainPublicTestSuite) TestChain() {
	tests := []struct {
		name     string
		blocks   []chain.Block
		contains []string
		absent   []string
	}{
		{
			name: "every block in it",
			blocks: []chain.Block{
				{Model: "amp", DSP: 0, Pos: 0, Enabled: true},
				{Model: "cab", DSP: 0, Pos: 1, Enabled: false},
				{Model: "weird", DSP: 1, Pos: 0, Enabled: true},
				{Model: "ghost", DSP: 1, Pos: 1, Enabled: true},
			},
			contains: []string{
				"Test Amp", "Some Amp", "Test Cab", "○", "●",
				"ghost", "not in catalog", "?", "dsp0", "dsp1", "30.0%",
			},
		},
		{
			name: "a block needing the owner's own IR",
			blocks: []chain.Block{
				{Model: "amp", Enabled: true},
				{
					Model: "HD2_ImpulseResponse1024", Pos: 1, Enabled: true,
					Params: chain.Params{"Index": catalog.Int(82)},
				},
			},
			contains: []string{
				// The slot is the useful thing to show, since the audio is
				// not in the file.
				"IR slot 82",
				"same IRs are loaded there",
			},
		},
		{
			name: "an IR naming no slot",
			blocks: []chain.Block{
				{Model: "HD2_ImpulseResponse1024", Enabled: true},
			},
			contains: []string{"a user IR"},
		},
		{
			name:     "a processor nothing uses",
			blocks:   []chain.Block{{Model: "amp", DSP: 1, Enabled: true}},
			contains: []string{"dsp1"},
			absent:   []string{"dsp0"},
		},
		{name: "nothing at all", contains: []string{"empty"}},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var out bytes.Buffer

			s.Require().NoError(paint.Chain(&out, s.spec(tt.blocks...), s.cat()))

			for _, want := range tt.contains {
				s.Require().Contains(out.String(), want)
			}

			for _, unwanted := range tt.absent {
				s.Require().NotContains(out.String(), unwanted)
			}
		})
	}
}

// TestChainReportsAWriterThatFails covers a writer failing at each point a
// chain writes.
func (s *ChainPublicTestSuite) TestChainReportsAWriterThatFails() {
	tests := []struct {
		name   string
		blocks []chain.Block
		after  int
	}{
		{name: "with nothing to show"},
		{name: "on the rows", blocks: []chain.Block{{Model: "amp"}}},
		{name: "on the budget", blocks: []chain.Block{{Model: "amp"}}, after: 1},
		{
			name:   "on the budget's own line",
			blocks: []chain.Block{{Model: "amp"}},
			after:  2,
		},
		{
			name: "on the warning about somebody's own IR",
			blocks: []chain.Block{{
				Model: "HD2_ImpulseResponse1024", Enabled: true,
				Params: chain.Params{"Index": catalog.Int(82)},
			}},
			after: 3,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().Error(paint.Chain(
				&failAfter{ok: tt.after}, s.spec(tt.blocks...), s.cat()))
		})
	}
}

// TestMeter fills in proportion to what a chain uses.
func (s *ChainPublicTestSuite) TestMeter() {
	var out bytes.Buffer

	tests := []struct {
		name string
		pct  float64
		want string
	}{
		{name: "nothing used", pct: 0, want: "░░░░░░░░░░"},
		{name: "half used", pct: 50, want: "█████░░░░░"},
		{name: "warning", pct: 80, want: "████████░░"},
		{name: "critical", pct: 95, want: "█████████░"},
		{name: "all used", pct: 100, want: "██████████"},
		{name: "more than all used", pct: 150, want: "██████████"},
		{name: "a negative reading", pct: -10, want: "░░░░░░░░░░"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.want, paint.Meter(&out, tt.pct, 10))
		})
	}
}

// TestCategory names a kind of block.
func (s *ChainPublicTestSuite) TestCategory() {
	var out bytes.Buffer

	tests := []struct {
		name string
		in   catalog.Category
		want string
	}{
		{name: "a kind this project knows", in: catalog.CategoryAmp, want: "amp"},
		{name: "one it does not", in: catalog.Category("nothing"), want: "nothing"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.want, paint.Category(&out, tt.in))
		})
	}
}

// failAfter fails once it has accepted ok writes, so a caller writing several
// times can be failed at a chosen point.
type failAfter struct {
	ok int
	n  int
}

func (w *failAfter) Write(p []byte) (int, error) {
	w.n++
	if w.n > w.ok {
		return 0, errors.New("boom")
	}

	return len(p), nil
}

func TestChainPublicTestSuite(t *testing.T) {
	suite.Run(t, new(ChainPublicTestSuite))
}
