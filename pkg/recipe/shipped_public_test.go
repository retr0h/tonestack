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
package recipe_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/recipe"
)

// ShippedPublicTestSuite checks the recipes this repository ships.
//
// Drift between the Go types and the contract is impossible — the types are
// generated from schemas/recipe.openapi.yaml. What generation does not
// guarantee is that the files on disk satisfy it, or that a recipe's filename
// matches the identifier inside it.
type ShippedPublicTestSuite struct {
	suite.Suite
}

func (s *ShippedPublicTestSuite) TestEveryShippedRecipeLoads() {
	paths, err := filepath.Glob(filepath.Join("..", "..", "recipes", "*", "*.yaml"))
	s.Require().NoError(err)
	s.Require().NotEmpty(paths, "no recipes found to check")

	for _, path := range paths {
		s.Run(filepath.Base(path), func() {
			f, err := os.Open(path)
			s.Require().NoError(err)

			defer func() { s.Require().NoError(f.Close()) }()

			r, err := recipe.Load(f)
			s.Require().NoError(err)

			s.Require().Equal(r.ID+".yaml", filepath.Base(path),
				"a recipe must be findable by name without opening it")
		})
	}
}

func TestShippedPublicTestSuite(t *testing.T) {
	suite.Run(t, new(ShippedPublicTestSuite))
}
