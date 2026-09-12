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

package catalogen_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/internal/catalogen"
)

// StalePublicTestSuite covers the catalog going out of date quietly.
type StalePublicTestSuite struct {
	suite.Suite
}

// TestTheCatalogMatchesTheInstalledRelease covers the one mistake nothing
// else here can catch.
//
// The catalog embedded in this binary is generated from a licensed HX Edit
// installation, which is why it cannot be a `go:generate` directive: most
// people working on this do not have the application, and a commit gate that
// tried to read it would fail for all of them.
//
// That leaves "remember to run `just catalog`" as the mechanism, and nobody
// remembers. Update HX Edit, forget, and the embedded catalog goes on
// describing the old release — every test still passes, because every test
// reads the same stale file.
//
// So this runs only where the mistake can be made. No HX Edit, nothing to
// compare against, skip. HX Edit present and disagreeing with what the
// catalog says it came from, fail and say which.
func (s *StalePublicTestSuite) TestTheCatalogMatchesTheInstalledRelease() {
	installed := catalogen.SourceFor(catalogen.DefaultResourcesDir)
	if installed == "" {
		s.T().Skip("no HX Edit installed; nothing to check the catalog against")
	}

	built, err := catalog.BuiltIn()
	s.Require().NoError(err)

	s.Require().Equal(installed, built.Source,
		"%s is installed and the embedded catalog came from %q — run `just catalog`",
		installed, built.Source)
}

func TestStalePublicTestSuite(t *testing.T) {
	suite.Run(t, new(StalePublicTestSuite))
}
