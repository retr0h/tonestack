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

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/cmd"
)

// RecipesPublicTestSuite covers where the recipes commands read and write.
type RecipesPublicTestSuite struct {
	suite.Suite
}

// run executes the command tree with args and returns what it printed.
//
// Flags hold their values between runs, so every call names each directory
// flag it depends on, empty where the default is meant.
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
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(args)

	err := root.ExecuteContext(context.Background())

	return out.String(), err
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
		err  bool
	}{
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

			s.Require().NoError(listErr, listed)
			s.Require().NoError(showErr, shown)
			s.Require().NoError(makeErr, made)

			s.Require().Contains(listed, tt.want)
			s.Require().Contains(shown, tt.want)
			s.Require().Contains(made, tt.want)

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

			if flag != "" {
				return
			}

			// A directory of your own adds to the rigs that ship rather
			// than hiding them.
			s.Require().Contains(listed, "mike-dirnt")

			shipped := filepath.Join(s.T().TempDir(), "mike-dirnt.hlx")
			s.run("presets", "make", "--recipes=",
				"--id", "mike-dirnt", "--out", shipped)
			s.Require().FileExists(shipped)
		})
	}
}

func TestRecipesPublicTestSuite(t *testing.T) {
	suite.Run(t, new(RecipesPublicTestSuite))
}
