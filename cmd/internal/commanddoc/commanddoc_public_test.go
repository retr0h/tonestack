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
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/cmd"
	"github.com/retr0h/tonestack/cmd/internal/commanddoc"
)

// asTonestack, when set, makes the test binary run as tonestack itself with
// these arguments, separated by newlines.
const asTonestack = "COMMANDDOC_AS_TONESTACK"

// TestMain lets a test run the real CLI in a process of its own.
//
// --help has to be asked of what somebody actually runs: cmd.Execute, with
// the help renderer it installs. Executing the tree in this process instead
// would add cobra's help and completion commands to the tree the page is
// rendered from.
func TestMain(
	m *testing.M,
) {
	if args, ok := os.LookupEnv(asTonestack); ok {
		os.Args = append([]string{"tonestack"}, strings.Split(args, "\n")...)
		cmd.Execute()
		os.Exit(0)
	}

	os.Exit(m.Run())
}

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

	// Inherits --where from its group and has no flags of its own.
	list := &cobra.Command{Use: "list", Short: "list them", Run: run}

	group.PersistentFlags().String("where", "", "where they are")
	group.AddCommand(build, hidden, list)

	// No flags anywhere above it: --help is still one.
	status := &cobra.Command{Use: "status", Short: "say how it is", Run: run}

	// Asks for no [flags] in its usage.
	plain := &cobra.Command{
		Use: "plain", Short: "no flags shown", Run: run, DisableFlagsInUseLine: true,
	}
	plain.Flags().String("mode", "", "which way")

	// Its own --help, hidden, so cobra adds none and nothing is left to show.
	quiet := &cobra.Command{Use: "quiet", Short: "hides its help", Run: run}
	quiet.Flags().Bool("help", false, "not listed")
	_ = quiet.Flags().MarkHidden("help")

	root.AddCommand(group, status, plain, quiet)

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
			name:     "a command's usage says [flags] where it has flags of its own",
			contains: []string{"```text\ntool thing make [flags]\n```"},
		},
		{
			// Rendering merges a group's persistent flags into the commands
			// beneath it, which once changed this line on a second render.
			name:     "a command with only inherited flags says [flags], as --help does",
			contains: []string{"```text\ntool thing list [flags]\n```"},
		},
		{
			name:     "a command with no flags at all says [flags] for its --help",
			contains: []string{"```text\ntool status [flags]\n```"},
		},
		{
			name:     "a command that turns [flags] off in its usage",
			contains: []string{"```text\ntool plain\n```"},
		},
		{
			name:     "a command whose only flag is a hidden --help",
			contains: []string{"```text\ntool quiet\n```"},
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

	// The same tree twice: a page must not depend on what reading the tree
	// the first time left behind in it.
	tree := s.tree()
	first := string(commanddoc.Render(tree))
	second := string(commanddoc.Render(tree))

	for _, tt := range tests {
		s.Run(tt.name, func() {
			for _, page := range []string{first, second} {
				for _, want := range tt.contains {
					s.Require().Contains(page, want)
				}

				for _, not := range tt.absent {
					s.Require().NotContains(page, not)
				}
			}
		})
	}

	s.Run("the same tree renders the same page every time", func() {
		s.Require().Equal(first, second)
	})

	// --help adds itself to the command it is asked of before answering.
	s.Run("the same page after --help has been asked of every command", func() {
		var visit func(*cobra.Command)

		visit = func(c *cobra.Command) {
			c.InitDefaultHelpFlag()

			for _, sub := range c.Commands() {
				visit(sub)
			}
		}

		visit(tree)

		s.Require().Equal(first, string(commanddoc.Render(tree)))
	})
}

// TestUsageMatchesHelp keeps each usage line on the page the one --help
// prints for that command, so the two cannot drift apart again.
func (s *CommanddocPublicTestSuite) TestUsageMatchesHelp() {
	page := string(commanddoc.Render(cmd.Root()))

	self, err := os.Executable()
	s.Require().NoError(err)

	var paths [][]string

	var walk func(*cobra.Command, []string)

	walk = func(c *cobra.Command, path []string) {
		paths = append(paths, path)

		for _, sub := range c.Commands() {
			if sub.IsAvailableCommand() {
				walk(sub, append(append([]string{}, path...), sub.Name()))
			}
		}
	}

	walk(cmd.Root(), nil)

	for _, path := range paths {
		name := strings.Join(append([]string{"tonestack"}, path...), " ")

		s.Run(name, func() {
			run := exec.CommandContext( //nolint:gosec // this test's own binary
				s.T().Context(), self)
			run.Env = append(os.Environ(),
				asTonestack+"="+strings.Join(append(path, "--help"), "\n"),
				"NO_COLOR=1")

			out, err := run.Output()
			s.Require().NoError(err)

			want := helpUsage(string(out))
			s.Require().NotEmpty(want, "--help printed no usage line:\n%s", out)

			s.Require().Equal(want, pageUsage(page, name),
				"docs/commands.md and --help disagree on how %s is invoked", name)
		})
	}
}

// escape matches a terminal colour sequence.
var escape = regexp.MustCompile("\x1b\\[[0-9;]*m")

// helpUsage is the first line under USAGE in what --help printed.
func helpUsage(
	out string,
) string {
	lines := strings.Split(escape.ReplaceAllString(out, ""), "\n")

	for i, line := range lines {
		if strings.TrimSpace(line) == "USAGE" && i+1 < len(lines) {
			return strings.TrimSpace(lines[i+1])
		}
	}

	return ""
}

// pageUsage is the usage line under a command's heading on the page.
func pageUsage(
	page string,
	name string,
) string {
	_, section, found := strings.Cut(page, "\n## "+name+"\n")
	if !found {
		return ""
	}

	_, block, _ := strings.Cut(section, "```text\n")
	line, _, _ := strings.Cut(block, "\n")

	return line
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
