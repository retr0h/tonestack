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
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/internal/cli"
)

// YAMLPublicTestSuite covers painting a document without changing it.
type YAMLPublicTestSuite struct {
	suite.Suite
}

// plain paints for a sink that takes no colour, which is what a buffer and a
// redirected file both are.
func (s *YAMLPublicTestSuite) plain(body string) string {
	return cli.YAML(&bytes.Buffer{}, body)
}

func (s *YAMLPublicTestSuite) TestADocumentSurvivesBeingPainted() {
	// The whole point: a rig redirected to a file is still the rig. Painting
	// happens only where a terminal can show it.
	body := strings.Join([]string{
		"# a rig",
		"schema: RigSpec",
		"chain:",
		"- gear: Ampeg SVT",
		"  role: amp",
		"",
		"target:",
		"  device: HX Stomp",
	}, "\n")

	s.Require().Equal(body, s.plain(body))
}

func (s *YAMLPublicTestSuite) TestALineThatIsNotAField() {
	s.Require().Equal("just words", s.plain("just words"))
}

func (s *YAMLPublicTestSuite) TestAColourIsShownLit() {
	// Painted in the colour it names where a terminal can show it, and left
	// exactly as written where one cannot.
	s.Require().Equal("  led: violet", s.plain("  led: violet"))

	// `auto` is not a colour — the light follows the block — and a name this
	// does not know gets no colour rather than a wrong one.
	s.Require().Equal("  led: auto color", s.plain("  led: auto color"))
	s.Require().Equal("  led: chartreuse", s.plain("  led: chartreuse"))
}

func (s *YAMLPublicTestSuite) TestSwatchRefusesWhatIsNotAColour() {
	for _, rgb := range []int{-1, 0x1000000} {
		s.Require().Equal("x", cli.Swatch(&bytes.Buffer{}, rgb, "x"))
	}
}

func (s *YAMLPublicTestSuite) TestSwatchOnBlack() {
	// A switch that lights nothing has no hue to raise.
	s.Require().Equal("x", cli.Swatch(&bytes.Buffer{}, 0, "x"))
}

func TestYAMLPublicTestSuite(t *testing.T) {
	suite.Run(t, new(YAMLPublicTestSuite))
}
