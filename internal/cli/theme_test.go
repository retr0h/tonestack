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

package cli

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type ThemeTestSuite struct {
	suite.Suite
}

func (s *ThemeTestSuite) TearDownTest() { active = &ThemeTube }

func (s *ThemeTestSuite) TestApplyEnv() {
	tests := []struct {
		name string
		env  string
		want string
	}{
		{"a theme that exists", "tube", "tube"},
		{"one nobody registered", "chartreuse", "tube"},
		{"nothing set", "", "tube"},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			ApplyEnv(tc.env)

			s.Require().Equal(tc.want, ActiveTheme().Name,
				"an unknown name leaves the default rather than failing")
		})
	}
}

func (s *ThemeTestSuite) TestMarkedFallsBackWithoutColour() {
	// lipgloss emits no escapes when the sink is not a terminal, and an
	// uncoloured tick carries no meaning on its own.
	s.Require().Equal("[ok] done", Marked("✓", "[ok]", "done"))
	s.Require().Equal(
		"\x1b[32m✓\x1b[0m done", Marked("\x1b[32m✓\x1b[0m", "[ok]", "done"))
}

func (s *ThemeTestSuite) TestThemeEnvIsNamedForTheProject() {
	s.Require().Equal("TONESTACK_THEME", ThemeEnv)
}

func TestThemeTestSuite(t *testing.T) {
	suite.Run(t, new(ThemeTestSuite))
}
