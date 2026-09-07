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

package cli_test

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/internal/cli"
	"github.com/retr0h/tonestack/pkg/catalog"
	"github.com/retr0h/tonestack/pkg/chain"
)

type ThemePublicTestSuite struct {
	suite.Suite
}

func (s *ThemePublicTestSuite) TearDownTest() {
	cli.SetTheme("tube")
}

func (s *ThemePublicTestSuite) TestEveryRoleRenders() {
	var out bytes.Buffer

	tests := []struct {
		name string
		call func() string
	}{
		{"mute", func() string { return cli.Mute(&out, "x") }},
		{"accent", func() string { return cli.Accent(&out, "x") }},
		{"ok", func() string { return cli.OK(&out, "x") }},
		{"err", func() string { return cli.Err(&out, "x") }},
		{"info", func() string { return cli.Info(&out, "x") }},
		{"title", func() string { return cli.Title(&out, "x") }},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			s.Require().Contains(tc.call(), "x")
		})
	}
}

func (s *ThemePublicTestSuite) TestHeadingIsUppercased() {
	s.Require().Equal("SLOT", cli.Heading(&bytes.Buffer{}, "slot"))
}

func (s *ThemePublicTestSuite) TestBannerNamesTheTool() {
	got := cli.Banner(&bytes.Buffer{})

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

func (s *ThemePublicTestSuite) TestSuccessAndFailureFallBackWithoutColor() {
	// A buffer is not a terminal, so lipgloss emits no escapes and the marks
	// degrade to words rather than vanishing.
	var out bytes.Buffer

	s.Require().Equal("[ok] done", cli.Success(&out, "done"))
	s.Require().Equal("[err] broke", cli.Failure(&out, "broke"))
}

func (s *ThemePublicTestSuite) TestThemeLookup() {
	tests := []struct {
		name  string
		theme string
		want  bool
	}{
		{"the default", "tube", true},
		{"a different case", "TUBE", true},
		{"surrounded by space", "  tube  ", true},
		{"one nobody registered", "chartreuse", false},
		{"nothing at all", "", false},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			s.Require().Equal(tc.want, cli.SetTheme(tc.theme))
		})
	}
}

func (s *ThemePublicTestSuite) TestActiveThemeAndNames() {
	s.Require().Equal("tube", cli.ActiveTheme().Name)
	s.Require().Equal([]string{"tube"}, cli.ThemeNames())
}

func (s *ThemePublicTestSuite) TestRendererFollowsTheSink() {
	// A *os.File gets its own renderer so NO_COLOR and TTY detection apply to
	// the sink actually written to, not to stdout.
	s.Require().NotPanics(func() { cli.Mute(os.Stderr, "x") })
}

func TestThemePublicTestSuite(t *testing.T) {
	suite.Run(t, new(ThemePublicTestSuite))
}

type HelpPublicTestSuite struct {
	suite.Suite
}

func (s *HelpPublicTestSuite) TestRenderShowsEverySection() {
	var out bytes.Buffer

	s.Require().NoError(cli.Help{
		Name:        "tonestack presets make",
		Description: "Build a preset.\n\nFrom a recipe.",
		Usage:       "tonestack presets make [flags]",
		Commands:    []cli.Item{{Name: "list", Description: "list them"}},
		Flags:       []cli.Item{{Name: "--id string", Description: "which one"}},
		Footer:      "Run --help for more.",
	}.Render(&out))

	got := out.String()
	for _, want := range []string{
		"tonestack presets make", "Build a preset.", "USAGE",
		"COMMANDS", "list", "FLAGS", "--id string", "Run --help for more.",
	} {
		s.Require().Contains(got, want)
	}
}

func (s *HelpPublicTestSuite) TestRenderOmitsWhatIsNotThere() {
	var out bytes.Buffer

	s.Require().NoError(cli.Help{Usage: "tonestack"}.Render(&out))

	got := out.String()
	s.Require().NotContains(got, "COMMANDS")
	s.Require().NotContains(got, "FLAGS")
	s.Require().Contains(got, "USAGE")
}

func (s *HelpPublicTestSuite) TestRenderReportsAWriterThatFails() {
	full := cli.Help{
		Name:        "n",
		Description: "d",
		Usage:       "u",
		Commands:    []cli.Item{{Name: "c"}},
		Flags:       []cli.Item{{Name: "f"}},
		Footer:      "foot",
	}

	// Each section is a separate write, so a writer failing at any point must
	// surface rather than leaving a half-rendered page and a success.
	// Ten writes make a complete page: title, description, three headings
	// with a row each, the footer, and the closing newline.
	for i := range 10 {
		s.Run(fmt.Sprintf("after %d writes", i), func() {
			s.Require().Error(full.Render(&failAfter{ok: i}))
		})
	}
}

func TestHelpPublicTestSuite(t *testing.T) {
	suite.Run(t, new(HelpPublicTestSuite))
}

type UIPublicTestSuite struct {
	suite.Suite
}

func (s *UIPublicTestSuite) TestSectionRendersATable() {
	var out bytes.Buffer

	s.Require().NoError(cli.Section{
		Title:   "Songs",
		Detail:  "128 slots",
		Headers: []string{"slot", "name"},
		Rows:    [][]string{{"01A", "Claptone"}},
		Summary: "1 of 128",
	}.Render(&out))

	got := out.String()
	for _, want := range []string{"Songs", "128 slots", "SLOT", "01A", "1 of 128"} {
		s.Require().Contains(got, want)
	}
}

func (s *UIPublicTestSuite) TestSectionSaysWhenThereIsNothing() {
	var out bytes.Buffer

	s.Require().NoError(cli.Section{
		Title: "Songs", Empty: "no presets",
	}.Render(&out))

	s.Require().Contains(out.String(), "no presets")
}

func (s *UIPublicTestSuite) TestSectionStaysSilentWithNothingToSay() {
	var out bytes.Buffer

	s.Require().NoError(cli.Section{}.Render(&out))

	s.Require().Empty(out.String())
}

func (s *UIPublicTestSuite) TestSectionWithoutHeadersOrSummary() {
	var out bytes.Buffer

	s.Require().NoError(cli.Section{
		Rows: [][]string{{"a", "b"}},
	}.Render(&out))

	s.Require().Contains(out.String(), "a")
}

func (s *UIPublicTestSuite) TestSectionReportsAWriterThatFails() {
	full := cli.Section{
		Title: "t", Detail: "d",
		Headers: []string{"h"}, Rows: [][]string{{"r"}}, Summary: "s",
	}

	for i := range 4 {
		s.Run(fmt.Sprintf("after %d writes", i), func() {
			s.Require().Error(full.Render(&failAfter{ok: i}))
		})
	}

	s.Require().Error(cli.Section{Title: "t", Empty: "none"}.
		Render(&failAfter{ok: 1}))
	s.Require().Error(cli.Section{Rows: [][]string{{"r"}}}.
		Render(&failAfter{ok: 1}))
}

func (s *UIPublicTestSuite) TestDetailRendersFields() {
	var out bytes.Buffer

	s.Require().NoError(cli.Detail{
		Title:    "Mike Dirnt",
		Subtitle: "mike-dirnt",
		Fields: []cli.Field{
			{Label: "amp", Value: "Ampeg SVT"},
			{Label: "cab", Value: "unknown", Muted: true},
		},
		Note: "unverified",
	}.Render(&out))

	got := out.String()
	for _, want := range []string{
		"Mike Dirnt", "mike-dirnt", "amp", "Ampeg SVT", "unknown", "unverified",
	} {
		s.Require().Contains(got, want)
	}
}

func (s *UIPublicTestSuite) TestDetailReportsAWriterThatFails() {
	full := cli.Detail{
		Title: "t", Subtitle: "s",
		Fields: []cli.Field{{Label: "l", Value: "v"}}, Note: "n",
	}

	for i := range 4 {
		s.Run(fmt.Sprintf("after %d writes", i), func() {
			s.Require().Error(full.Render(&failAfter{ok: i}))
		})
	}
}

func (s *UIPublicTestSuite) TestTableAlignsColumns() {
	var out bytes.Buffer

	s.Require().NoError(cli.Table(&out, [][]string{
		{"a", "1", "end"},
		{"bbbb", "22", "end"},
	}, []lipgloss.Position{lipgloss.Left, lipgloss.Right}))

	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
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
}

func (s *UIPublicTestSuite) TestTableMeasuresStyledCells() {
	var out bytes.Buffer

	styled := lipgloss.NewStyle().Foreground(lipgloss.Color("#ffa032")).Render("ab")
	s.Require().NoError(cli.Table(&out, [][]string{
		{styled, "x"},
		{"abcd", "y"},
	}, nil))

	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	s.Require().Equal(lipgloss.Width(lines[0]), lipgloss.Width(lines[1]))
}

func (s *UIPublicTestSuite) TestTableToleratesRaggedRows() {
	var out bytes.Buffer

	s.Require().NoError(cli.Table(&out, [][]string{{"a"}, {"bb", "cc"}}, nil))

	s.Require().Contains(out.String(), "cc")
}

func (s *UIPublicTestSuite) TestTableWritesNothingForNoRows() {
	var out bytes.Buffer

	s.Require().NoError(cli.Table(&out, nil, nil))

	s.Require().Empty(out.String())
}

func (s *UIPublicTestSuite) TestTableReportsAWriterThatFails() {
	err := cli.Table(&failAfter{}, [][]string{{"a"}}, nil)

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
	}}
}

func (s *ChainPublicTestSuite) spec(blocks ...chain.Block) chain.Chain {
	return chain.Chain{Name: "Test", Blocks: blocks}
}

func (s *ChainPublicTestSuite) TestChainRendersEveryBlock() {
	var out bytes.Buffer

	s.Require().NoError(cli.Chain(&out, s.spec(
		chain.Block{Model: "amp", DSP: 0, Pos: 0, Enabled: true},
		chain.Block{Model: "cab", DSP: 0, Pos: 1, Enabled: false},
		chain.Block{Model: "weird", DSP: 1, Pos: 0, Enabled: true},
		chain.Block{Model: "ghost", DSP: 1, Pos: 1, Enabled: true},
	), s.cat()))

	got := out.String()
	for _, want := range []string{
		"Test Amp", "Some Amp", "Test Cab", "○", "●",
		"ghost", "not in catalog", "?", "dsp0", "dsp1", "30.0%",
	} {
		s.Require().Contains(got, want)
	}
}

func (s *ChainPublicTestSuite) TestChainFlagsABlockNeedingTheOwnersOwnIR() {
	var out bytes.Buffer

	cat := s.cat()
	cat.Blocks["HD2_ImpulseResponse1024"] = catalog.Block{
		Name: "IR 1024", Category: catalog.CategoryCab,
		DSP: catalog.DSPCost{Mono: 7},
	}

	s.Require().NoError(cli.Chain(&out, chain.Chain{Blocks: []chain.Block{
		{Model: "amp", Enabled: true},
		{
			Model: "HD2_ImpulseResponse1024", Pos: 1, Enabled: true,
			Params: chain.Params{"Index": catalog.Int(82)},
		},
	}}, cat))

	got := out.String()
	s.Require().Contains(got, "IR slot 82",
		"the slot is the useful thing to show, since the audio is not in the file")
	s.Require().Contains(got, "same IRs are loaded there")
}

func (s *ChainPublicTestSuite) TestChainDescribesAnIRWithNoSlot() {
	var out bytes.Buffer

	cat := s.cat()
	cat.Blocks["HD2_ImpulseResponse1024"] = catalog.Block{Name: "IR 1024"}

	s.Require().NoError(cli.Chain(&out, chain.Chain{Blocks: []chain.Block{
		{Model: "HD2_ImpulseResponse1024", Enabled: true},
	}}, cat))

	s.Require().Contains(out.String(), "a user IR")
}

func (s *ChainPublicTestSuite) TestChainReportsAWriterThatFailsOnTheIRWarning() {
	cat := s.cat()
	cat.Blocks["HD2_ImpulseResponse1024"] = catalog.Block{Name: "IR 1024"}

	s.Require().Error(cli.Chain(&failAfter{ok: 3}, chain.Chain{
		Blocks: []chain.Block{
			{
				Model: "HD2_ImpulseResponse1024", Enabled: true,
				Params: chain.Params{"Index": catalog.Int(82)},
			},
		},
	}, cat))
}

func (s *ChainPublicTestSuite) TestChainSkipsAProcessorNothingUses() {
	var out bytes.Buffer

	s.Require().NoError(cli.Chain(&out, s.spec(
		chain.Block{Model: "amp", DSP: 1, Enabled: true},
	), s.cat()))

	s.Require().NotContains(out.String(), "dsp0")
	s.Require().Contains(out.String(), "dsp1")
}

func (s *ChainPublicTestSuite) TestChainSaysWhenThereIsNothing() {
	var out bytes.Buffer

	s.Require().NoError(cli.Chain(&out, s.spec(), s.cat()))

	s.Require().Contains(out.String(), "empty")
}

func (s *ChainPublicTestSuite) TestChainReportsAWriterThatFails() {
	tests := []struct {
		name  string
		spec  chain.Chain
		after int
	}{
		{"with nothing to show", s.spec(), 0},
		{"on the rows", s.spec(chain.Block{Model: "amp"}), 0},
		{"on the budget", s.spec(chain.Block{Model: "amp"}), 1},
		{"on the budget's own line", s.spec(chain.Block{Model: "amp"}), 2},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			s.Require().Error(cli.Chain(&failAfter{ok: tc.after}, tc.spec, s.cat()))
		})
	}
}

func (s *ChainPublicTestSuite) TestMeterFillsInProportion() {
	var out bytes.Buffer

	tests := []struct {
		name string
		pct  float64
		want string
	}{
		{"nothing used", 0, "░░░░░░░░░░"},
		{"half used", 50, "█████░░░░░"},
		{"warning", 80, "████████░░"},
		{"critical", 95, "█████████░"},
		{"all used", 100, "██████████"},
		{"more than all used", 150, "██████████"},
		{"a negative reading", -10, "░░░░░░░░░░"},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			s.Require().Equal(tc.want, cli.Meter(&out, tc.pct, 10))
		})
	}
}

func (s *ChainPublicTestSuite) TestCategoryNamesEveryKind() {
	var out bytes.Buffer

	s.Require().Equal("amp", cli.Category(&out, catalog.CategoryAmp))
	s.Require().Equal("nothing", cli.Category(&out, catalog.Category("nothing")))
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
