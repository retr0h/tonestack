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

package compile_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/chain"
	"github.com/retr0h/tonestack/pkg/sdk/internal/compile"
	"github.com/retr0h/tonestack/pkg/sdk/preset"
	riggen "github.com/retr0h/tonestack/pkg/sdk/rig/gen"
)

// CompilePublicTestSuite covers the package's work reached as a value.
//
// Each method is the package-level function of the same name, so what is
// asserted here is that calling it through the type and calling it directly
// agree. The behaviour itself is covered by the function's own suite.
type CompilePublicTestSuite struct {
	suite.Suite

	cat *catalog.Catalog
}

func (s *CompilePublicTestSuite) SetupSuite() { s.cat = loadCatalog(&s.Suite) }

// TestNew covers what a caller is handed.
func (s *CompilePublicTestSuite) TestNew() {
	s.Require().NotNil(compile.New())
}

// TestLift covers reading a preset into a rig through the type.
func (s *CompilePublicTestSuite) TestLift() {
	tests := []struct {
		name string
		doc  func() *preset.Document
	}{
		{name: "a blank preset", doc: func() *preset.Document {
			doc, _ := preset.Blank()

			return doc
		}},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			doc := tt.doc()
			want, wantErr := compile.Lift(doc, s.cat)
			got, err := compile.New().Lift(doc, s.cat)

			s.Require().Equal(wantErr == nil, err == nil)
			s.Require().Equal(want, got)
		})
	}
}

// TestLower covers writing a rig back into a preset through the type.
func (s *CompilePublicTestSuite) TestLower() {
	tests := []struct {
		name string
		spec riggen.RigSpec
	}{
		{name: "a rig naming an amplifier", spec: recipe("Ampeg SVT", "")},
		{name: "a rig naming nothing that resolves", spec: riggen.RigSpec{}},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			first, _ := preset.Blank()
			second, _ := preset.Blank()

			want := compile.Lower(first, tt.spec, s.cat)
			got := compile.New().Lower(second, tt.spec, s.cat)

			s.Require().Equal(want == nil, got == nil)
			s.Require().Equal(first, second)
		})
	}
}

// TestResolve covers turning a rig into a chain through the type.
func (s *CompilePublicTestSuite) TestResolve() {
	tests := []struct {
		name string
		spec riggen.RigSpec
	}{
		{name: "a rig naming an amplifier", spec: recipe("Ampeg SVT", "")},
		{name: "a rig naming gear no model emulates", spec: recipe("Nothing At All", "")},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			want, wantAdded, wantErr := compile.Resolve(tt.spec, s.cat, nil)
			got, gotAdded, err := compile.New().Resolve(tt.spec, s.cat, nil)

			s.Require().Equal(wantErr == nil, err == nil)
			s.Require().Equal(want, got)
			s.Require().Equal(wantAdded, gotAdded)
		})
	}
}

// TestFit covers dropping what a device has no room for, through the type.
func (s *CompilePublicTestSuite) TestFit() {
	built, _, err := compile.Resolve(recipe("Ampeg SVT", ""), s.cat, nil)
	s.Require().NoError(err)

	tests := []struct {
		name string
		lim  chain.Limits
	}{
		{name: "room for everything", lim: twoChips(1)},
		{name: "room for nothing", lim: oneChip(0)},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().Equal(
				compile.Fit(built, s.cat, tt.lim),
				compile.New().Fit(built, s.cat, tt.lim))
		})
	}
}

func TestCompilePublicTestSuite(t *testing.T) {
	suite.Run(t, new(CompilePublicTestSuite))
}
