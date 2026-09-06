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

package catalog_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/catalog"
)

type TypesPublicTestSuite struct {
	suite.Suite
}

func (s *TypesPublicTestSuite) TestTrustedAcceptsEverythingButAssumed() {
	for _, p := range []catalog.Provenance{
		catalog.ProvOfficial,
		catalog.ProvMeasured,
		catalog.ProvObserved,
		catalog.ProvInherited,
	} {
		s.Require().True(p.Trusted(), "%s should be trusted", p)
	}
}

func (s *TypesPublicTestSuite) TestTrustedRejectsAssumed() {
	s.Require().False(catalog.ProvAssumed.Trusted())
}

func (s *TypesPublicTestSuite) TestProvenanceValuesAreDistinct() {
	all := []catalog.Provenance{
		catalog.ProvOfficial,
		catalog.ProvMeasured,
		catalog.ProvObserved,
		catalog.ProvInherited,
		catalog.ProvAssumed,
	}

	seen := make(map[catalog.Provenance]bool, len(all))
	for _, p := range all {
		s.Require().False(seen[p], "duplicate provenance %q", p)
		seen[p] = true
	}
}

func TestTypesPublicTestSuite(t *testing.T) {
	suite.Run(t, new(TypesPublicTestSuite))
}
