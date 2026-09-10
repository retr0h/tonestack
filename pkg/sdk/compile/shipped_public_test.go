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
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/compile"
	"github.com/retr0h/tonestack/pkg/sdk/rig"
	"github.com/retr0h/tonestack/pkg/sdk/rigs"
)

// ShippedPublicTestSuite holds the rigs this repository ships to the
// vocabulary, which nothing holds anybody else's rigs to.
//
// A rig somebody writes gets a note naming the terms nothing defines and
// still builds, because a character term moves no knob and refusing a preset
// over a word would be refusing them the right to describe a sound. The rigs
// here are different: they are the examples everybody copies, and every one
// of them said what it meant in a sentence until there was a list.
type ShippedPublicTestSuite struct {
	suite.Suite
}

// TestEveryShippedRigUsesTheVocabulary covers the terms every shipped rig
// describes itself with.
//
// Read through the embedded copy rather than off disk, so this counts no
// directories and travels wherever the package does.
func (s *ShippedPublicTestSuite) TestEveryShippedRigUsesTheVocabulary() {
	paths, err := fs.Glob(rigs.FS, filepath.Join("*", "*.yaml"))
	s.Require().NoError(err)
	s.Require().NotEmpty(paths, "no rigs found to check")

	for _, path := range paths {
		s.Run(filepath.Base(path), func() {
			f, err := rigs.FS.Open(path)
			s.Require().NoError(err)

			defer func() { s.Require().NoError(f.Close()) }()

			spec, err := rig.Load(f)
			s.Require().NoError(err)

			for _, u := range compile.CheckCharacter(spec) {
				s.Require().Fail("no such character term",
					"%q. Add it to pkg/sdk/compile/data/character-terms.json "+
						"with a definition, or use one of: %v", u.Term, u.Near)
			}
		})
	}
}

// TestEveryExampleUsesTheVocabulary covers the rigs the docs point at.
func (s *ShippedPublicTestSuite) TestEveryExampleUsesTheVocabulary() {
	paths, err := filepath.Glob(
		filepath.Join("..", "..", "..", "examples", "rigspec", "*.yaml"))
	s.Require().NoError(err)
	s.Require().NotEmpty(paths, "no examples found to check")

	for _, path := range paths {
		s.Run(filepath.Base(path), func() {
			f, err := os.Open(path)
			s.Require().NoError(err)

			defer func() { s.Require().NoError(f.Close()) }()

			spec, err := rig.Load(f)
			s.Require().NoError(err)

			s.Require().Empty(compile.CheckCharacter(spec))
		})
	}
}

func TestShippedPublicTestSuite(t *testing.T) {
	suite.Run(t, new(ShippedPublicTestSuite))
}
