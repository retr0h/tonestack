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
package commanddoc_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/cmd"
	"github.com/retr0h/tonestack/cmd/internal/commanddoc"
)

// CommanddocPublicTestSuite covers the page the CLI generates.
type CommanddocPublicTestSuite struct {
	suite.Suite
}

// tree builds a small command tree with one of everything the page renders.
func (s *CommanddocPublicTestSuite) tree() *cobra.Command {
	root := &cobra.Command{Use: "tool", Short: "the tool"}
	group := &cobra.Command{Use: "thing", Short: "work with things"}

	run := func(*cobra.Command, []string) {}

	build := &cobra.Command{
		Use: "make", Short: "make one", Long: "Make one, at length.", Run: run,
	}
	build.Flags().String("id", "", "which one | exactly")
	build.Flags().BoolP("all", "a", false, "every one")
	build.Flags().Int("count", 3, "how many")
	build.Flags().String("secret", "", "not for you")
	_ = build.Flags().MarkHidden("secret")

	hidden := &cobra.Command{Use: "ghost", Short: "nobody sees this", Hidden: true, Run: run}

	group.AddCommand(build, hidden)
	root.AddCommand(group)

	return root
}

// TestRender turns a command tree into a page.
func (s *CommanddocPublicTestSuite) TestRender() {
	tests := []struct {
		name     string
		contains []string
		absent   []string
	}{
		{
			name:     "every command gets a heading of its full path",
			contains: []string{"## tool\n", "## tool thing\n", "## tool thing make\n"},
		},
		{
			name: "a group names its usage and links each command it holds",
			contains: []string{
				"tool thing <command> [flags]",
				"| [make](#tool-thing-make) | make one |",
			},
		},
		{
			// The same choice --help makes.
			name:     "the long description where there is one",
			contains: []string{"Make one, at length."},
		},
		{
			name: "a flag's value, default and meaning",
			contains: []string{
				"| `--count` | int | `3` | how many |",
				"| `-a, --all` |  |  | every one |",
			},
		},
		{
			// A pipe in prose would end the table cell early.
			name:     "a flag whose description holds a table's delimiter",
			contains: []string{"| `--id` | string |  | which one \\| exactly |"},
		},
		{
			name:   "what the help deliberately leaves out",
			absent: []string{"ghost", "nobody sees this", "--secret"},
		},
	}

	page := string(commanddoc.Render(s.tree()))

	for _, tt := range tests {
		s.Run(tt.name, func() {
			for _, want := range tt.contains {
				s.Require().Contains(page, want)
			}

			for _, not := range tt.absent {
				s.Require().NotContains(page, not)
			}
		})
	}
}

// TestTheShippedPageIsCurrent keeps the committed page honest.
//
// A generated reference nobody regenerates is a hand-written one with extra
// steps. This fails the moment a flag is added, renamed or removed without
// the page following, in the ordinary test run.
func (s *CommanddocPublicTestSuite) TestTheShippedPageIsCurrent() {
	want := commanddoc.Render(cmd.Root())

	got, err := os.ReadFile( //nolint:gosec // a path this repository owns
		filepath.Join("..", "..", "..", "docs", "commands.md"))
	s.Require().NoError(err)

	s.Require().Equal(string(want), string(got),
		"docs/commands.md is out of date — run `just generate`")
}

func TestCommanddocPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(CommanddocPublicTestSuite))
}
