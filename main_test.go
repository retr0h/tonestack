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

package main

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

// MainTestSuite covers the shape of the repository rather than its behaviour.
type MainTestSuite struct {
	suite.Suite
}

// TestPkgDoesNotImportInternal asserts the boundary the compiler will not.
//
// `internal/` sits at the repository root, so Go permits everything here to
// import it, `pkg/` included. That makes the rule a convention, and a
// convention nothing checks is one somebody breaks by accident: a test
// reaching for a catalog helper is all it takes.
//
// The rule is that `pkg/` holds what something outside this repository would
// call. A package that imports `internal/` cannot be lifted out, so the import
// is the thing that says the code is on the wrong side.
func (s *MainTestSuite) TestPkgDoesNotImportInternal() {
	const internal = "github.com/retr0h/tonestack/internal/"

	fset := token.NewFileSet()

	err := filepath.WalkDir("pkg", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}

		f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}

		for _, i := range f.Imports {
			s.Require().NotContains(
				i.Path.Value, internal,
				"%s imports an internal package, so it cannot be lifted out", path)
		}

		return nil
	})

	s.Require().NoError(err)
}

func TestMainTestSuite(t *testing.T) {
	suite.Run(t, new(MainTestSuite))
}
