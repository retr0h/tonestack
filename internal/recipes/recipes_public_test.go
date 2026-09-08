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
package recipes_test

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/internal/recipes"
	riggen "github.com/retr0h/tonestack/pkg/rig/gen"
)

type RecipesPublicTestSuite struct {
	suite.Suite
}

func (s *RecipesPublicTestSuite) good() string  { return "testdata-good" }
func (s *RecipesPublicTestSuite) mixed() string { return "testdata" }

// TestLoad reads a directory of recipes.
func (s *RecipesPublicTestSuite) TestLoad() {
	locked := s.T().TempDir()
	s.Require().NoError(os.MkdirAll(filepath.Join(locked, "artists"), 0o750))

	path := filepath.Join(locked, "artists", "locked.yaml")
	s.Require().NoError(os.WriteFile(path, []byte("id: locked"), 0o600))
	s.Require().NoError(os.Chmod(path, 0o000))

	tests := []struct {
		name  string
		dir   string
		ids   []string
		empty bool
		err   string
	}{
		{
			name: "every recipe in the directory, sorted",
			dir:  s.good(),
			ids:  []string{"mike-dirnt", "mike-dirnt-longview", "minimal"},
		},
		{
			// A half-read knowledge base is worse than a complaint about the
			// file to fix, so one bad recipe fails the whole load.
			name: "one that will not parse stops the load",
			dir:  s.mixed(),
			err:  "broken.yaml",
		},
		{
			name: "one that cannot be opened",
			dir:  locked,
			err:  "opening",
		},
		{
			name:  "a directory holding none",
			dir:   s.T().TempDir(),
			empty: true,
		},
		{
			// No directory is the case for anyone running an installed
			// binary rather than working in a checkout.
			name: "no directory falls back to the built-in recipes",
			dir:  "",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			all, err := recipes.Load(tt.dir)

			if tt.err != "" {
				s.Require().Error(err)
				s.Require().Contains(err.Error(), tt.err)

				return
			}

			s.Require().NoError(err)

			if tt.empty {
				s.Require().Empty(all)

				return
			}

			if tt.ids == nil {
				s.Require().NotEmpty(all, "recipes ship in the binary")

				return
			}

			s.Require().Equal(tt.ids, ids(all))
		})
	}
}

// TestFind looks one recipe up.
func (s *RecipesPublicTestSuite) TestFind() {
	tests := []struct {
		name string
		dir  string
		id   string
		want string
		errs []string
	}{
		{
			name: "by identifier",
			dir:  s.good(),
			id:   "mike-dirnt",
			want: "mike-dirnt",
		},
		{
			name: "whatever case somebody typed",
			dir:  s.good(),
			id:   "MIKE-DIRNT",
			want: "mike-dirnt",
		},
		{
			name: "by an alias the recipe claims",
			dir:  s.good(),
			id:   "spare",
			want: "minimal",
		},
		{
			name: "one nobody wrote",
			dir:  s.good(),
			id:   "nobody",
			errs: []string{"recipes list"},
		},
		{
			name: "a directory that will not load",
			dir:  s.mixed(),
			id:   "mike-dirnt",
			errs: []string{"broken.yaml"},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := recipes.Find(tt.dir, tt.id)

			if tt.errs != nil {
				s.Require().Error(err)

				for _, want := range tt.errs {
					s.Require().Contains(err.Error(), want)
				}

				return
			}

			s.Require().NoError(err)
			s.Require().Equal(tt.want, got.ID)
		})
	}
}

// TestList writes out what is on the shelf.
func (s *RecipesPublicTestSuite) TestList() {
	tests := []struct {
		name     string
		dir      string
		to       io.Writer
		contains []string
		err      bool
	}{
		{
			name:     "one line per recipe",
			dir:      s.good(),
			contains: []string{"mike-dirnt", "Ampeg SVT", "llm"},
		},
		{
			name:     "a shelf with nothing on it",
			dir:      s.T().TempDir(),
			contains: []string{"no recipes here"},
		},
		{
			name: "a directory that will not load",
			dir:  s.mixed(),
			err:  true,
		},
		{
			name: "nowhere to write it",
			dir:  s.good(),
			to:   &failingWriter{},
			err:  true,
		},
		{
			name: "nowhere to write the empty case either",
			dir:  s.T().TempDir(),
			to:   &failingWriter{},
			err:  true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var buf bytes.Buffer

			to := tt.to
			if to == nil {
				to = &buf
			}

			err := recipes.List(to, tt.dir)

			if tt.err {
				s.Require().Error(err)

				return
			}

			s.Require().NoError(err)

			for _, want := range tt.contains {
				s.Require().Contains(buf.String(), want)
			}
		})
	}
}

// TestShow writes out one recipe.
func (s *RecipesPublicTestSuite) TestShow() {
	tests := []struct {
		name     string
		dir      string
		id       string
		to       io.Writer
		contains []string
		absent   []string
		err      bool
	}{
		{
			name: "everything a person wrote",
			dir:  s.good(),
			id:   "mike-dirnt",
			contains: []string{
				"Mike Dirnt", "Green Day", "Dookie through American Idiot",
				"pick, near the bridge", "mid-forward", "Longview",
				// An llm-sourced recipe must say nobody confirmed it.
				"unverified",
			},
		},
		{
			name: "nothing about what nobody wrote",
			dir:  s.good(),
			id:   "minimal",
			absent: []string{
				"character", "variants", "band",
				// A curated recipe is confirmed.
				"unverified",
			},
		},
		{
			// Saying nothing about how far to trust a rig is not a claim
			// that it can be trusted.
			name:     "an unstated confidence reads as low",
			dir:      s.good(),
			id:       "mike-dirnt-longview",
			contains: []string{"low confidence"},
		},
		{
			name: "a recipe nobody wrote",
			dir:  s.good(),
			id:   "nobody",
			err:  true,
		},
		{
			name: "a directory that will not load",
			dir:  s.mixed(),
			id:   "mike-dirnt",
			err:  true,
		},
		{
			name: "nowhere to write it",
			dir:  s.good(),
			id:   "mike-dirnt",
			to:   &failingWriter{},
			err:  true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var buf bytes.Buffer

			to := tt.to
			if to == nil {
				to = &buf
			}

			err := recipes.Show(to, tt.dir, tt.id)

			if tt.err {
				s.Require().Error(err)

				return
			}

			s.Require().NoError(err)

			for _, want := range tt.contains {
				s.Require().Contains(buf.String(), want)
			}

			for _, gone := range tt.absent {
				s.Require().NotContains(buf.String(), gone)
			}
		})
	}
}

// TestNotFoundError covers what somebody reads when the recipe is not there.
func (s *RecipesPublicTestSuite) TestNotFoundError() {
	err := &recipes.NotFoundError{ID: "flea", Known: 3}

	s.Require().Contains(err.Error(), "flea")
	s.Require().Contains(err.Error(), "3 known")
	s.Require().ErrorIs(err, recipes.ErrNotFound)
}

// TestDefaultDirIsWhereRecipesLive keeps the fallback pointing at something.
func (s *RecipesPublicTestSuite) TestDefaultDirIsWhereRecipesLive() {
	s.Require().Equal("resources/recipes", recipes.DefaultDir)
	s.Require().DirExists(filepath.Join("..", "..", recipes.DefaultDir))
}

// ids names what a load returned, in order.
func ids(all []riggen.RigSpec) []string {
	out := make([]string, 0, len(all))
	for _, r := range all {
		out = append(out, r.ID)
	}

	return out
}

type failingWriter struct{}

func (*failingWriter) Write([]byte) (int, error) { return 0, errors.New("boom") }

func TestRecipesPublicTestSuite(t *testing.T) {
	suite.Run(t, new(RecipesPublicTestSuite))
}
