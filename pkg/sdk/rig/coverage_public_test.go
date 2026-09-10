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

package rig_test

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"

	"github.com/retr0h/tonestack/pkg/sdk/rig"
	"github.com/stretchr/testify/suite"
	"sigs.k8s.io/yaml"
)

// CoveragePublicTestSuite holds the contract to the rigs that demonstrate it.
//
// A field nobody has ever written is a field nobody has checked the shape of,
// and the only description of it is prose in a schema most people never open.
// Thirty-three of them were in that state when this was written.
type CoveragePublicTestSuite struct {
	suite.Suite
}

// exempt names the fields no rig here can honestly carry, and why.
//
// A reason rather than a list, so that adding to it is a decision somebody
// has to defend rather than a way past a failing test.
var exempt = map[string]string{
	"extends": "records that one rig departs from another. Every rig here " +
		"is a sibling rather than a departure, and `recipes new --from` " +
		"writes the field, so exercising it would mean inventing a rig to " +
		"have something to extend.",
}

// TestEveryFieldAppearsInARig walks the contract and finds each field written
// down somewhere a person can read it.
func (s *CoveragePublicTestSuite) TestEveryFieldAppearsInARig() {
	declared := s.declared()
	s.Require().NotEmpty(declared)

	written := s.written()
	s.Require().NotEmpty(written)

	missing := []string(nil)

	for _, field := range declared {
		if written[field] {
			continue
		}

		if _, ok := exempt[field]; ok {
			continue
		}

		missing = append(missing, field)
	}

	s.Require().Empty(missing,
		"no rig under examples/rigspec or resources/recipes writes these. "+
			"Add one to a rig, or add it to exempt with a reason: %v", missing)
}

// declared returns every property the contract names, however deep.
func (s *CoveragePublicTestSuite) declared() []string {
	doc, err := openapi3.NewLoader().LoadFromData(rig.Schema)
	s.Require().NoError(err)

	out := map[string]bool{}

	var walk func(schema *openapi3.SchemaRef, depth int)

	walk = func(schema *openapi3.SchemaRef, depth int) {
		// A rig nests a handful of levels. The bound is against a contract
		// that grows a cycle rather than against the one there is.
		if schema == nil || schema.Value == nil || depth > 8 {
			return
		}

		for name, sub := range schema.Value.Properties {
			out[name] = true

			walk(sub, depth+1)
		}

		walk(schema.Value.Items, depth+1)
	}

	walk(doc.Components.Schemas["RigSpec"], 0)

	names := make([]string, 0, len(out))
	for name := range out {
		names = append(names, name)
	}

	sort.Strings(names)

	return names
}

// written returns every key any rig in this repository uses.
//
// Both the examples and the rigs that ship, because both are documents
// somebody reads and copies, and a field exercised in a real rig is better
// evidence than one exercised in a demonstration.
func (s *CoveragePublicTestSuite) written() map[string]bool {
	out := map[string]bool{}

	for _, pattern := range [][]string{
		{"..", "..", "..", "examples", "rigspec", "*.yaml"},
		{"..", "..", "..", "resources", "recipes", "*", "*.yaml"},
	} {
		paths, err := filepath.Glob(filepath.Join(pattern...))
		s.Require().NoError(err)
		s.Require().NotEmpty(paths)

		for _, path := range paths {
			raw, err := os.ReadFile(path) //nolint:gosec // a path this test chose
			s.Require().NoError(err)

			var body any
			s.Require().NoError(yaml.Unmarshal(raw, &body))

			s.keys(body, out)
		}
	}

	return out
}

// keys records every mapping key in a decoded document.
func (s *CoveragePublicTestSuite) keys(value any, out map[string]bool) {
	switch typed := value.(type) {
	case map[string]any:
		for key, sub := range typed {
			out[key] = true

			s.keys(sub, out)
		}
	case []any:
		for _, sub := range typed {
			s.keys(sub, out)
		}
	}
}

func TestCoveragePublicTestSuite(t *testing.T) {
	suite.Run(t, new(CoveragePublicTestSuite))
}
