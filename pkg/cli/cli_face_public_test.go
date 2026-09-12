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
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/cli"
)

// ThemeFacePublicTestSuite covers the part of the visual language a caller
// can reach.
//
// The language itself is private: how a table is drawn and what a colour
// resolves to belong to whoever draws them. What is public is picking a theme
// and the two marks a command needs, because those are the things a command
// configures rather than paints.
type ThemeFacePublicTestSuite struct {
	suite.Suite
}

// TestSetTheme covers picking one by name.
func (s *ThemeFacePublicTestSuite) TestSetTheme() {
	restore := cli.ActiveTheme().Name
	defer func() { cli.SetTheme(restore) }()

	tests := []struct {
		name  string
		theme string
		want  bool
	}{
		{name: "one there is", theme: "tube", want: true},
		{name: "one there is not", theme: "no such theme"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.want, cli.SetTheme(tt.theme))

			if tt.want {
				s.Require().Equal(tt.theme, cli.ActiveTheme().Name)
			}
		})
	}
}

// TestThemeNames covers listing what there is to pick from.
//
// The default comes first, because a help page shows it that way and a flag
// that offered them in map order would shuffle between runs.
func (s *ThemeFacePublicTestSuite) TestThemeNames() {
	got := cli.ThemeNames()

	s.Require().NotEmpty(got)
	s.Require().Equal(cli.ThemeNames(), got)

	for _, name := range got {
		s.Require().True(cli.SetTheme(name), "%s is offered but unknown", name)
	}

	s.Require().True(cli.SetTheme(got[0]))
}

// TestFailurePrefix covers the mark that goes in front of an error line.
//
// Whatever reports the error prints it; this is what makes that line read in
// this project's voice rather than the framework's.
func (s *ThemeFacePublicTestSuite) TestFailurePrefix() {
	var buf bytes.Buffer

	s.Require().NotEmpty(cli.FailurePrefix(&buf))
}

// TestBanner covers the heading a help page opens with.
func (s *ThemeFacePublicTestSuite) TestBanner() {
	var buf bytes.Buffer

	s.Require().NotEmpty(cli.Banner(&buf))
}

func TestThemeFacePublicTestSuite(t *testing.T) {
	suite.Run(t, new(ThemeFacePublicTestSuite))
}
