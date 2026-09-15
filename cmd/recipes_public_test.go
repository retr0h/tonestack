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
	var out bytes.Buffer

	root := cmd.Root()
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(args)

	err := root.ExecuteContext(context.Background())
	s.Require().NoError(err, out.String())

	return out.String()
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
