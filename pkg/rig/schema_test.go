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

	"github.com/retr0h/tonestack/pkg/rig"
	"github.com/retr0h/tonestack/pkg/rig/gen"
	"github.com/retr0h/tonestack/resources/schemas"
)

// SchemaTestSuite covers reading the contract a rig is checked against.
//
// The one this binary ships cannot fail — the Go types are generated from the
// same document, so a broken one would have failed generation first. What is
// worth covering is that it says so rather than silently accepting anything.
type SchemaTestSuite struct {
	suite.Suite
}

func (s *SchemaTestSuite) TestTheShippedContractLoads() {
	got, err := rig.LoadSchema(schemas.RigSpec)

	s.Require().NoError(err)
	s.Require().NotNil(got)
}

func (s *SchemaTestSuite) TestReportsADocumentItCannotRead() {
	_, err := rig.LoadSchema([]byte("not a schema"))

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "RigSpec schema")
}

func (s *SchemaTestSuite) TestReportsADocumentDescribingNoRig() {
	_, err := rig.LoadSchema([]byte(`
openapi: 3.0.3
info: { title: Something Else, version: "1.0.0" }
paths: {}
components:
  schemas:
    NotARig: { type: object }
`))

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "describes no RigSpec")
}

func (s *SchemaTestSuite) TestARigCannotBeCheckedWithoutAContract() {
	// Nothing reaches this in a shipped binary. It is here so that if the
	// contract ever could not be read, a rig would be reported as unchecked
	// rather than passed as valid.
	restore := *rig.Contract
	defer func() { *rig.Contract = restore }()

	boom := errors.New("no contract")
	*rig.Contract = func() (*openapi3.Schema, error) { return nil, boom }

	err := rig.Validate(gen.RigSpec{})

	s.Require().ErrorIs(err, boom)
}

func (s *SchemaTestSuite) TestAFailureWithNothingToPointAt() {
	// A document that is not an object at all fails the contract as a whole
	// rather than at a field, so there is nothing to name but the contract.
	err := rig.Against("not a rig")

	s.Require().ErrorIs(err, rig.ErrInvalid)
	s.Require().Contains(err.Error(), "RigSpec")
}

func (s *SchemaTestSuite) TestSeveralFailuresReportTheFirst() {
	// A rig missing everything fails in more ways than a person can act on
	// at once. The first is the one worth showing.
	err := rig.Against(map[string]any{})

	s.Require().ErrorIs(err, rig.ErrInvalid)
}

func (s *SchemaTestSuite) TestAFailureOfSomeOtherKind() {
	// The library reports a failed field today. If it ever reports something
	// else, that has to reach somebody rather than be swallowed.
	err := rig.Invalid(errors.New("something else went wrong"))

	s.Require().ErrorIs(err, rig.ErrInvalid)
	s.Require().Contains(err.Error(), "something else went wrong")
}

func (s *SchemaTestSuite) TestAFailureThatGivesNoReason() {
	err := rig.Invalid(&openapi3.SchemaError{
		Schema: openapi3.NewStringSchema(),
		Value:  1,
	})

	s.Require().ErrorIs(err, rig.ErrInvalid)
	s.Require().NotEmpty(err.Error())
}

func TestSchemaTestSuite(t *testing.T) {
	suite.Run(t, new(SchemaTestSuite))
}
