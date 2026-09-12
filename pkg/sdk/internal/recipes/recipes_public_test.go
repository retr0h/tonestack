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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/internal/recipes"
	"github.com/retr0h/tonestack/pkg/sdk/rig"
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
		name string
		dir  string
		// the identifiers the answer must carry, in any order.
		ids []string
		err bool
	}{
		{
			name: "one entry per recipe",
			dir:  s.good(),
			ids:  []string{"mike-dirnt"},
		},
		{
			// A shelf with nothing on it is not a failure. Somebody who
			// just made the directory is owed an empty answer.
			name: "a shelf with nothing on it",
			dir:  s.T().TempDir(),
		},
		{
			name: "a directory that will not load",
			dir:  s.mixed(),
			err:  true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			all, err := recipes.List(tt.dir)

			if tt.err {
				s.Require().Error(err)

				return
			}

			s.Require().NoError(err)
			s.Require().Equal(tt.dir, all.Dir)

			got := ids(all.Rigs)

			for _, want := range tt.ids {
				s.Require().Contains(got, want)
			}

			if tt.ids == nil {
				s.Require().Empty(all.Rigs)
			}
		})
	}
}

// TestShow writes out one recipe.
func (s *RecipesPublicTestSuite) TestShow() {
	tests := []struct {
		name string
		dir  string
		id   string
		// the rigs that say they are a small change on this one.
		variants []string
		err      bool
		is       error
	}{
		{
			// The link points the other way — a variant names what it
			// extends — so only reading the whole set answers this.
			name:     "a rig something else departs from",
			dir:      s.good(),
			id:       "mike-dirnt",
			variants: []string{"mike-dirnt-longview"},
		},
		{
			name: "a rig nothing departs from",
			dir:  s.good(),
			id:   "minimal",
		},
		{
			// Matched with errors.Is, so a caller can tell "no such recipe"
			// from "the shelf would not open" without reading the message.
			name: "a recipe nobody wrote",
			dir:  s.good(),
			id:   "nobody",
			err:  true,
			is:   recipes.ErrNotFound,
		},
		{
			name: "a directory that will not load",
			dir:  s.mixed(),
			id:   "mike-dirnt",
			err:  true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			one, err := recipes.Show(tt.dir, tt.id)

			if tt.err {
				s.Require().Error(err)

				if tt.is != nil {
					s.Require().ErrorIs(err, tt.is)
				}

				return
			}

			s.Require().NoError(err)
			s.Require().Equal(tt.id, one.Rig.ID)

			got := make([]string, 0, len(one.Variants))
			for _, v := range one.Variants {
				got = append(got, v.ID)
			}

			s.Require().ElementsMatch(tt.variants, got)
		})
	}
}

// ids reads the identifiers out of a set of rigs, so a test can say which
// were found without also saying what else each one holds.
func ids(all []rig.Spec) []string {
	out := make([]string, 0, len(all))
	for _, r := range all {
		out = append(out, r.ID)
	}

	return out
}

func TestRecipesPublicTestSuite(t *testing.T) {
	suite.Run(t, new(RecipesPublicTestSuite))
}
