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

package presets_test

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/retr0h/tonestack/pkg/sdk/chain"
	"github.com/retr0h/tonestack/pkg/sdk/internal/presets"
	presetmocks "github.com/retr0h/tonestack/pkg/sdk/internal/presets/mocks"
	riggen "github.com/retr0h/tonestack/pkg/sdk/rig/gen"
)

// TypesPublicTestSuite covers standing something else in for a collaborator.
//
// Every other suite here leaves Deps zero and gets the real thing, which is
// what the command does in earnest. These drive the other half: a build given
// a double uses it, and the failure it reports is the double's.
type TypesPublicTestSuite struct {
	suite.Suite

	ctrl *gomock.Controller
}

func (s *TypesPublicTestSuite) SetupTest() { s.ctrl = gomock.NewController(s.T()) }

func (s *TypesPublicTestSuite) TearDownTest() { s.ctrl.Finish() }

// options names a build that would succeed on the real collaborators.
func (s *TypesPublicTestSuite) options(deps presets.Deps) presets.MakeOptions {
	return presets.MakeOptions{
		Deps:       deps,
		RecipeID:   "mike-dirnt",
		RecipesDir: filepath.Join("..", "recipes", "testdata-good"),
		OutputPath: filepath.Join(s.T().TempDir(), "out.hlx"),
	}
}

// TestRecipes covers a build finding its rig through a double.
func (s *TypesPublicTestSuite) TestRecipes() {
	want := errors.New("no such recipe here")

	rec := presetmocks.NewMockRecipes(s.ctrl)
	rec.EXPECT().Find(gomock.Any(), "mike-dirnt").Return(riggen.RigSpec{}, want)

	_, err := presets.Make(s.options(presets.Deps{Recipes: rec}))

	s.Require().ErrorIs(err, want)
}

// TestCatalogs covers a build opening its catalog through a double.
func (s *TypesPublicTestSuite) TestCatalogs() {
	want := errors.New("no catalog here")

	cat := presetmocks.NewMockCatalogs(s.ctrl)
	cat.EXPECT().Open(gomock.Any()).Return(nil, want)

	_, err := presets.Make(s.options(presets.Deps{Catalogs: cat}))

	s.Require().ErrorIs(err, want)
}

// TestCompiler covers a build resolving its chain through a double.
func (s *TypesPublicTestSuite) TestCompiler() {
	want := errors.New("cannot resolve that")

	comp := presetmocks.NewMockCompiler(s.ctrl)
	comp.EXPECT().Resolve(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(chain.Chain{}, nil, want)

	_, err := presets.Make(s.options(presets.Deps{Compiler: comp}))

	s.Require().ErrorIs(err, want)
}

func TestTypesPublicTestSuite(t *testing.T) {
	suite.Run(t, new(TypesPublicTestSuite))
}
