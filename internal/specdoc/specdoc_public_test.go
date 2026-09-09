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

package specdoc_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/internal/specdoc"
	"github.com/retr0h/tonestack/resources/schemas"
)

// SpecdocPublicTestSuite covers the page the contract generates.
type SpecdocPublicTestSuite struct {
	suite.Suite
}

// TestRender turns a contract into a page.
func (s *SpecdocPublicTestSuite) TestRender() {
	tests := []struct {
		name     string
		schema   string
		contains []string
		absent   []string
		err      bool
	}{
		{
			name: "a closed set, listed",
			schema: object(`
        role:
          type: string
          enum: [amp, cab]`),
			contains: []string{"| `role` | string | closed | `amp`, `cab` |"},
		},
		{
			name: "a field checked against the catalog",
			schema: object(`
        gear:
          type: string
          x-lookup: what the catalog maps a model to`),
			contains: []string{
				"| `gear` | string | looked up | what the catalog maps a model to |",
			},
		},
		{
			name: "a field held to a pattern",
			schema: object(`
        id:
          type: string
          pattern: "^[a-z]+$"`),
			contains: []string{"shaped | `^[a-z]+$`"},
		},
		{
			// Prose is a bucket too, and saying so is the point: a reader
			// can tell it from a field that has an answer.
			name: "a field nothing checks",
			schema: object(`
        note:
          type: string`),
			contains: []string{"| `note` | string | open | — |"},
		},
		{
			name: "a number held to a range",
			schema: object(`
        drive:
          type: number
          minimum: 0
          maximum: 1`),
			contains: []string{"shaped | `0` to `1`"},
		},
		{
			name: "a number held from below",
			schema: object(`
        level:
          type: number
          minimum: 0`),
			contains: []string{"shaped | `0` or more"},
		},
		{
			name: "a number held from above",
			schema: object(`
        level:
          type: number
          maximum: 10`),
			contains: []string{"shaped | `10` or less"},
		},
		{
			// Nothing to tabulate, so the constraint is answered here rather
			// than linked to a section nobody emits.
			name: "a field holding a map of numbers",
			schema: `
openapi: 3.0.3
info: { title: t, version: "1" }
paths: {}
components:
  schemas:
    RigSpec:
      type: object
      properties:
        settings:
          $ref: "#/components/schemas/Settings"
    Settings:
      type: object
      additionalProperties: { type: number, minimum: 0, maximum: 1 }
`,
			contains: []string{"| `settings` | map of number | shaped | `0` to `1` |"},
			absent:   []string{"(#settings)"},
		},
		{
			// The question moves to that object's own table rather than
			// being answered twice.
			name: "a field holding another object",
			schema: `
openapi: 3.0.3
info: { title: t, version: "1" }
paths: {}
components:
  schemas:
    RigSpec:
      type: object
      properties:
        subject:
          $ref: "#/components/schemas/Subject"
    Subject:
      type: object
      properties:
        name:
          type: string
`,
			contains: []string{"[Subject](#subject)", "## Subject"},
		},
		{
			name:   "something that is not a contract",
			schema: "not a schema",
			err:    true,
		},
		{
			name: "a contract describing nothing",
			schema: `
openapi: 3.0.3
info: { title: t, version: "1" }
paths: {}
`,
			err: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := specdoc.Render([]byte(tt.schema))

			if tt.err {
				s.Require().Error(err)

				return
			}

			s.Require().NoError(err)

			for _, want := range tt.contains {
				s.Require().Contains(string(got), want)
			}

			for _, unwanted := range tt.absent {
				s.Require().NotContains(string(got), unwanted)
			}
		})
	}
}

// TestEveryLinkPointsAtASection covers a page that refers to itself.
//
// A field holding another object points at that object's table. One holding
// something with no fields to tabulate — Settings is a map of numbers — has no
// table to point at, and linked to a heading nobody emits.
func (s *SpecdocPublicTestSuite) TestEveryLinkPointsAtASection() {
	body, err := specdoc.Render(schemas.RigSpec)
	s.Require().NoError(err)

	page := string(body)

	sections := map[string]bool{}
	for _, m := range regexp.MustCompile(`(?m)^## (\w+)`).FindAllStringSubmatch(page, -1) {
		sections[strings.ToLower(m[1])] = true
	}

	s.Require().NotEmpty(sections)

	for _, m := range regexp.MustCompile(`\]\(#([\w-]+)\)`).FindAllStringSubmatch(page, -1) {
		s.Require().True(sections[m[1]],
			"the page links to #%s and has no such section", m[1])
	}
}

// TestTheShippedPageIsCurrent is what keeps the page honest.
//
// A generated reference that nobody regenerates is a hand-written one with
// extra steps, which is how docs/recipes.md came to describe a format that had
// moved. This fails the moment the contract and the page disagree, in the
// ordinary test run rather than in a step somebody has to remember.
func (s *SpecdocPublicTestSuite) TestTheShippedPageIsCurrent() {
	want, err := specdoc.Render(schemas.RigSpec)
	s.Require().NoError(err)

	path := filepath.Join("..", "..", "docs", "rigspec.md")

	got, err := os.ReadFile(path) //nolint:gosec // a path this repository owns
	s.Require().NoError(err)

	s.Require().Equal(string(want), string(got),
		"docs/rigspec.md is out of date — run `just generate`")
}

// object wraps field definitions in the smallest contract that carries them.
func object(fields string) string {
	return `
openapi: 3.0.3
info: { title: t, version: "1" }
paths: {}
components:
  schemas:
    RigSpec:
      type: object
      properties:` + fields + "\n"
}

func TestSpecdocPublicTestSuite(t *testing.T) {
	suite.Run(t, new(SpecdocPublicTestSuite))
}
