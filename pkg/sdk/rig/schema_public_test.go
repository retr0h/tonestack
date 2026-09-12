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
	"errors"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/rig"
)

// SchemaPublicTestSuite covers reading the contract a rig is checked against.
//
// The one this binary ships cannot fail — the Go types are generated from the
// same document, so a broken one would have failed generation first. What is
// worth covering is that it says so rather than silently accepting anything.
type SchemaPublicTestSuite struct {
	suite.Suite
}

// TestLoadSchema reads the contract out of an OpenAPI document.
func (s *SchemaPublicTestSuite) TestLoadSchema() {
	tests := []struct {
		name    string
		doc     []byte
		errText string
	}{
		{name: "the contract this binary ships", doc: rig.Schema},
		{
			name:    "a document it cannot read",
			doc:     []byte("not a schema"),
			errText: "RigSpec schema",
		},
		{
			name: "a document describing no rig",
			doc: []byte(`
openapi: 3.0.3
info: { title: Something Else, version: "1.0.0" }
paths: {}
components:
  schemas:
    NotARig: { type: object }
`),
			errText: "describes no RigSpec",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := rig.LoadSchema(tt.doc)

			if tt.errText != "" {
				s.Require().Error(err)
				s.Require().Contains(err.Error(), tt.errText)

				return
			}

			s.Require().NoError(err)
			s.Require().NotNil(got)
		})
	}
}

// TestValidate covers a rig checked against a contract nobody can read.
//
// Nothing reaches this in a shipped binary. It is here so that if the contract
// ever could not be read, a rig would be reported as unchecked rather than
// passed as valid.
func (s *SchemaPublicTestSuite) TestValidate() {
	restore := *rig.Contract
	defer func() { *rig.Contract = restore }()

	boom := errors.New("no contract")
	*rig.Contract = func() (*openapi3.Schema, error) { return nil, boom }

	s.Require().ErrorIs(rig.Validate(rig.Spec{}), boom)
}

// TestAgainst checks a document against the contract.
func (s *SchemaPublicTestSuite) TestAgainst() {
	tests := []struct {
		name     string
		doc      any
		contains string
	}{
		{
			// A document that is not an object at all fails the contract as a
			// whole rather than at a field, so there is nothing to name but
			// the contract.
			name:     "a failure with nothing to point at",
			doc:      "not a rig",
			contains: "RigSpec",
		},
		{
			// A rig missing everything fails in more ways than a person can
			// act on at once. The first is the one worth showing.
			name: "several failures at once",
			doc:  map[string]any{},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			err := rig.Against(tt.doc)

			s.Require().ErrorIs(err, rig.ErrInvalid)

			if tt.contains != "" {
				s.Require().Contains(err.Error(), tt.contains)
			}
		})
	}
}

// TestInvalid says what went wrong, whatever the library handed it.
func (s *SchemaPublicTestSuite) TestInvalid() {
	tests := []struct {
		name     string
		in       error
		contains string
	}{
		{
			// The library reports a failed field today. If it ever reports
			// something else, that has to reach somebody rather than be
			// swallowed.
			name:     "a failure of some other kind",
			in:       errors.New("something else went wrong"),
			contains: "something else went wrong",
		},
		{
			name: "a failure that gives no reason",
			in: &openapi3.SchemaError{
				Schema: openapi3.NewStringSchema(),
				Value:  1,
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			err := rig.Invalid(tt.in)

			s.Require().ErrorIs(err, rig.ErrInvalid)
			s.Require().NotEmpty(err.Error())

			if tt.contains != "" {
				s.Require().Contains(err.Error(), tt.contains)
			}
		})
	}
}

func TestSchemaTestSuite(t *testing.T) {
	suite.Run(t, new(SchemaPublicTestSuite))
}
