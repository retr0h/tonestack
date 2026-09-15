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
package cmd_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/cmd"
)

// RecipesPublicTestSuite covers where the recipes commands read and write.
type RecipesPublicTestSuite struct {
	suite.Suite
}

// run executes the command tree with args and returns what it printed.
func (s *RecipesPublicTestSuite) run(
	args ...string,
) string {
	out, err := s.try(args...)
	s.Require().NoError(err, out)

	return out
}

// try executes the command tree with args, and returns what it printed and
// whether it failed.
func (s *RecipesPublicTestSuite) try(
	args ...string,
) (string, error) {
	var out bytes.Buffer

	root := cmd.Root()
	reset(root)
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(args)

	err := root.ExecuteContext(context.Background())

	return out.String(), err
}

// reset puts every flag in the tree back to its default and unset.
//
// The tree is one value shared by every run, and a flag keeps both what it
// was given and that it was given it. Left alone, --amp from one run and
// --from from the next would be refused as naming both.
func reset(
	c *cobra.Command,
) {
	for _, flags := range []*pflag.FlagSet{c.Flags(), c.PersistentFlags()} {
		flags.VisitAll(func(f *pflag.Flag) {
			// Setting a flag's own default cannot fail.
			if list, ok := f.Value.(pflag.SliceValue); ok {
				_ = list.Replace(nil)
			} else {
				_ = f.Value.Set(f.DefValue)
			}

			f.Changed = false
		})
	}

	for _, sub := range c.Commands() {
		reset(sub)
	}
}

// theirs is a recipe of somebody's own, for the subject "Their Player".
func theirs(
	id string,
	aliases string,
) string {
	return fmt.Sprintf(`schema: RigSpec
version: 2
id: %s
%s

subject:
  kind: artist
  name: Their Player

instrument: bass

chain:
  - role: amp
    gear: Aguilar DB51
    evidence:
      - { kind: cited, note: "a test says so" }
    confidence: high

confidence: high
`, id, aliases)
}

// TestTheirsBesideTheShippedOnes covers which rig list, show and make find
// when somebody's own directory and the shipped rigs are both in play.
func (s *RecipesPublicTestSuite) TestTheirsBesideTheShippedOnes() {
	tests := []struct {
		name string
		// rig is a recipe of theirs to write, or empty for none.
		rig string
		// missing leaves their directory absent.
		missing bool
		// locked leaves their directory unreadable.
		locked bool
		// want is the subject every command answers mike-dirnt with.
		want string
		// variant is a rig show must name as departing from mike-dirnt.
		variant string
		// listErr is what list alone fails with, while show and make work.
		listErr string
		err     bool
	}{
		{
			// Variants are read across both, so the copy shows under the
			// shipped rig it was made from.
			name:    "a rig of theirs made from a shipped one",
			rig:     theirs("mike-dirnt-live", "extends: mike-dirnt"),
			want:    "Mike Dirnt",
			variant: "mike-dirnt-live",
		},
		{
			// One mistake of theirs does not stop a shipped rig building.
			name:    "a file of theirs that is not a rig",
			rig:     "schema: RigSpec\nid: broken\n",
			want:    "Mike Dirnt",
			listErr: "theirs.yaml",
		},
		{
			name: "the same identifier as a shipped rig",
			rig:  theirs("mike-dirnt", ""),
			want: "Their Player",
		},
		{
			// Otherwise show and make would find theirs through the alias
			// while list still showed the shipped one.
			name: "an alias that is a shipped rig's identifier, in any case",
			rig:  theirs("their-player", "aliases: [MIKE-DIRNT]"),
			want: "Their Player",
		},
		{
			name: "an identifier that is a shipped rig's alias",
			rig:  theirs("dirnt", ""),
			want: "Their Player",
		},
		{
			name: "a rig of theirs sharing no name with a shipped one",
			rig:  theirs("their-player", ""),
			want: "Mike Dirnt",
		},
		{
			// Nobody has written a recipe of their own yet.
			name:    "a directory that is not there",
			missing: true,
			want:    "Mike Dirnt",
		},
		{
			// Not the shipped rigs alone, which would hide theirs without
			// saying why.
			name:   "a directory that cannot be read",
			rig:    theirs("mike-dirnt", ""),
			locked: true,
			err:    true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			if tt.locked && os.Geteuid() == 0 {
				s.T().Skip("root reads a directory whatever its mode")
			}

			s.T().Chdir(s.T().TempDir())

			data := s.T().TempDir()
			s.T().Setenv("XDG_DATA_HOME", data)

			dir := filepath.Join(data, "tonestack", "recipes")

			if !tt.missing {
				s.Require().NoError(os.MkdirAll(filepath.Join(dir, "artists"), 0o750))
			}

			if tt.rig != "" {
				s.Require().NoError(os.WriteFile(
					filepath.Join(dir, "artists", "theirs.yaml"), []byte(tt.rig), 0o600))
			}

			if tt.locked {
				s.Require().NoError(os.Chmod(dir, 0o000))
				// Put back so the directory can be removed; a failure shows
				// up in TempDir's own cleanup.
				s.T().Cleanup(func() { _ = os.Chmod(dir, 0o750) })
			}

			hlx := filepath.Join(s.T().TempDir(), "out.hlx")

			listed, listErr := s.try("recipes", "list", "--dir=")
			shown, showErr := s.try("recipes", "show", "--dir=", "--id", "mike-dirnt")
			made, makeErr := s.try("presets", "make", "--recipes=",
				"--id", "mike-dirnt", "--out", hlx)

			if tt.err {
				for _, err := range []error{listErr, showErr, makeErr} {
					s.Require().ErrorContains(err, "reading "+dir)
				}

				return
			}

			s.Require().NoError(showErr, shown)
			s.Require().NoError(makeErr, made)
			s.Require().Contains(shown, tt.want)
			s.Require().Contains(made, tt.want)

			if tt.variant != "" {
				s.Require().Contains(shown, tt.variant)
			}

			if tt.listErr != "" {
				s.Require().ErrorContains(listErr, tt.listErr)

				return
			}

			s.Require().NoError(listErr, listed)
			s.Require().Contains(listed, tt.want)

			if tt.want != "Mike Dirnt" {
				s.Require().NotContains(listed, "Mike Dirnt",
					"list shows the rig show and make use, not both")
			}
		})
	}
}

// TestNew covers where a new recipe lands, and that it is then found.
func (s *RecipesPublicTestSuite) TestNew() {
	tests := []struct {
		name string
		// env sets the environment, given a scratch directory, and returns
		// the --dir flag to pass and the directory the recipe must land in.
		env func(scratch string) (flag, want string)
	}{
		{
			name: "by default, under XDG_DATA_HOME",
			env: func(scratch string) (string, string) {
				s.T().Setenv("XDG_DATA_HOME", scratch)

				return "", filepath.Join(scratch, "tonestack", "recipes")
			},
		},
		{
			name: "by default, under the home directory when XDG_DATA_HOME is unset",
			env: func(scratch string) (string, string) {
				s.T().Setenv("XDG_DATA_HOME", "")
				s.T().Setenv("HOME", scratch)

				return "", filepath.Join(
					scratch, ".local", "share", "tonestack", "recipes")
			},
		},
		{
			name: "wherever --dir names",
			env: func(scratch string) (string, string) {
				s.T().Setenv("XDG_DATA_HOME", s.T().TempDir())

				return scratch, scratch
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			work := s.T().TempDir()
			s.T().Chdir(work)

			flag, want := tt.env(s.T().TempDir())

			out := s.run("recipes", "new", "--dir="+flag,
				"--id", "test-player", "--name", "Test Player",
				"--instrument", "bass", "--amp", "Ampeg SVT")

			path := filepath.Join(want, "artists", "test-player.yaml")
			s.Require().FileExists(path)
			s.Require().Contains(out, path, "the output says where it went")

			left, err := os.ReadDir(work)
			s.Require().NoError(err)
			s.Require().Empty(left, "nothing is written where the command ran")

			listed := s.run("recipes", "list", "--dir="+flag)
			s.Require().Contains(listed, "test-player")

			s.run("recipes", "show", "--dir="+flag, "--id", "test-player")

			hlx := filepath.Join(s.T().TempDir(), "test-player.hlx")
			s.run("presets", "make", "--recipes="+flag,
				"--id", "test-player", "--out", hlx)
			s.Require().FileExists(hlx)

			// A directory of your own adds to the rigs that ship rather
			// than hiding them, whether it is yours or one --dir names.
			s.Require().Contains(listed, "mike-dirnt")

			shipped := filepath.Join(s.T().TempDir(), "mike-dirnt.hlx")
			s.run("presets", "make", "--recipes="+flag,
				"--id", "mike-dirnt", "--out", shipped)
			s.Require().FileExists(shipped)
		})
	}
}

// TestNewFlags covers what recipes new does with --kind and --instrument
// beside --from.
func (s *RecipesPublicTestSuite) TestNewFlags() {
	tests := []struct {
		name string
		args []string
		// err is what the command fails with, and nothing is written.
		err string
		// out must be in what it printed, and absent must not.
		out    string
		absent string
		// body must be in the file written.
		body string
		// example puts the meteor rig in the directory written to.
		example bool
	}{
		{
			// --kind names what a copy is attributed to, and a rig from gear
			// is always an artist, so taking it silently would be a lie.
			name: "--kind without --from",
			args: []string{"--amp", "Ampeg SVT", "--kind", "song"},
			err:  "--kind says what a copy is attributed to, so it needs --from",
		},
		{
			name: "--kind with --from",
			args: []string{"--from", "mike-dirnt", "--kind", "song"},
			body: "  kind: song",
		},
		{
			// The copy is played on what the copied rig is, whatever the
			// flag a copy does not read says.
			name:   "a copy reports the copied rig's instrument",
			args:   []string{"--from", "mike-dirnt", "--instrument", "guitar"},
			out:    "bass",
			absent: "guitar",
		},
		{
			// The copy is called what the copied rig is, since nothing
			// renamed it.
			name: "a copy with no --name reports the copied rig's name",
			args: []string{"--from", "mike-dirnt"},
			out:  "Mike Dirnt",
		},
		{
			name:   "a copy with --name reports that name",
			args:   []string{"--from", "mike-dirnt", "--name", "Basket Case"},
			out:    "Basket Case",
			absent: "Mike Dirnt",
		},
		{
			// A copy names no gear of its own, so the amp is the one the
			// chain it copied holds.
			name: "a copy reports the copied rig's amp",
			args: []string{"--from", "flea"},
			out:  "Gallien-Krueger 2001RB",
		},
		{
			name: "a copy reports the copied rig's cab",
			args: []string{"--from", "flea"},
			out:  "Gallien-Krueger 410",
		},
		{
			// Everything else the chain holds, in signal order.
			name:    "a copy reports the copied rig's pedals",
			args:    []string{"--from", "dir-angl-meteor"},
			example: true,
			out:     "Arbiter Cry Baby, Ibanez® TS808 Tube Screamer®",
		},
		{
			// Written bare, this name is a mapping and the file is not a rig.
			name: "a copy named with YAML syntax",
			args: []string{"--from", "flea", "--name", `a: "b" #c`},
			body: `name: 'a: "b" #c'`,
		},
		{
			// Written bare, a rig loads this name as the boolean true and
			// refuses the file. Yes is a band.
			name: "a copy named for a band called Yes",
			args: []string{"--from", "flea", "--name", "Yes"},
			body: `name: "Yes"`,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			dir := s.T().TempDir()
			s.T().Setenv("XDG_DATA_HOME", s.T().TempDir())

			if tt.example {
				raw, err := os.ReadFile(
					filepath.Join("..", "examples", "rigspec", "dir-angl-meteor.yaml"))
				s.Require().NoError(err)
				s.Require().NoError(os.MkdirAll(filepath.Join(dir, "artists"), 0o750))
				s.Require().NoError(os.WriteFile(
					filepath.Join(dir, "artists", "dir-angl-meteor.yaml"), raw, 0o600))
			}

			args := append([]string{"recipes", "new", "--dir", dir, "--id", "the-copy"}, tt.args...)
			out, err := s.try(args...)
			path := filepath.Join(dir, "artists", "the-copy.yaml")

			if tt.err != "" {
				s.Require().ErrorContains(err, tt.err)
				s.Require().NoFileExists(path)

				return
			}

			s.Require().NoError(err, out)

			if tt.out != "" {
				s.Require().Contains(out, tt.out)
			}

			if tt.absent != "" {
				s.Require().NotContains(out, tt.absent)
			}

			if tt.body != "" {
				body, err := os.ReadFile(path)
				s.Require().NoError(err)
				s.Require().Contains(string(body), tt.body)
			}
		})
	}
}

func TestRecipesPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(RecipesPublicTestSuite))
}
