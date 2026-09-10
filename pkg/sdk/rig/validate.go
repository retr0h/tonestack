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

// Package rig validates a RigSpec against the contract it declares.
//
// The types in gen/ are generated from resources/schemas/rigspec.openapi.yaml, and
// generation gives them shape but not rules: nothing stops a required field
// being empty or an enumeration holding a word that is not in it.
//
// Those rules are enforced by checking a rig against that same document,
// rather than against a second copy of it written in Go. Two copies drift: a
// constraint added to the schema becomes a type nothing enforces, and one
// removed becomes a check nothing asked for.
package rig

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/getkin/kin-openapi/openapi3"

	"github.com/retr0h/tonestack/pkg/sdk/rig/gen"
	"github.com/retr0h/tonestack/resources/schemas"
)

// ErrInvalid reports a rig that does not meet its own contract.
var ErrInvalid = errors.New("not a valid rig")

// InvalidError says which part of a rig is wrong.
type InvalidError struct {
	// Field is the path to the offending value, as it appears in the file.
	Field string
	// Reason says what is wrong with it.
	Reason string
}

func (e *InvalidError) Error() string {
	return fmt.Sprintf("not a valid rig: %s %s", e.Field, e.Reason)
}

func (*InvalidError) Unwrap() error { return ErrInvalid }

// Validate reports whether a rig meets the contract in the schema.
func Validate(s gen.RigSpec) error {
	// Through JSON, because that is the shape a schema describes. The tags on
	// the generated types map one to the other, and they came from the same
	// document as the rules.
	// A rig can carry raw JSON it was handed — the state a device wrote —
	// and something that is not JSON cannot be checked against anything.
	body, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("reading the rig: %w", err)
	}

	// What Marshal produced is JSON, so reading it back cannot fail.
	var document any
	_ = json.Unmarshal(body, &document)

	return against(document)
}

// schemaName is the contract a rig is checked against.
const schemaName = "RigSpec"

// contract returns the schema, parsed once.
//
// Parsing an OpenAPI document is not cheap and the document never changes, so
// it happens on the first rig checked and not again.
var contract = sync.OnceValues(func() (*openapi3.Schema, error) {
	return load(schemas.RigSpec)
})

// load reads the contract out of an OpenAPI document.
func load(raw []byte) (*openapi3.Schema, error) {
	doc, err := openapi3.NewLoader().LoadFromData(raw)
	if err != nil {
		return nil, fmt.Errorf("reading the %s schema: %w", schemaName, err)
	}

	ref, ok := doc.Components.Schemas[schemaName]
	if !ok {
		return nil, fmt.Errorf("the schema describes no %s", schemaName)
	}

	return ref.Value, nil
}

// against checks a document against the schema.
//
// The schema is the contract, so it is what does the checking. Writing the
// same rules a second time in Go is how the two drift: a constraint added to
// the schema would be a type nothing enforced, and one removed would be a
// check nothing had asked for.
func against(document any) error {
	schema, err := contract()
	if err != nil {
		return err
	}

	if err := schema.VisitJSON(document); err != nil {
		return invalid(err)
	}

	return nil
}

// invalid turns a schema failure into one that names the field.
//
// A rig is hand-written, so the field is the useful half of the message.
func invalid(err error) error {
	var schemaErr *openapi3.SchemaError
	if !errors.As(err, &schemaErr) {
		return &InvalidError{Field: schemaName, Reason: err.Error()}
	}

	return &InvalidError{
		Field:  fieldOf(schemaErr),
		Reason: reasonOf(schemaErr),
	}
}

// fieldOf names what failed, the way the file writes it.
//
// A schema points at a value with a JSON pointer, where every step is a name
// — `chain.0.role`. Somebody looking at their own YAML sees a list, so the
// steps that are positions are written as ones: `chain[0].role`.
func fieldOf(err *openapi3.SchemaError) string {
	path := err.JSONPointer()
	if len(path) == 0 {
		return schemaName
	}

	var out strings.Builder

	for _, step := range path {
		if _, err := strconv.Atoi(step); err == nil {
			fmt.Fprintf(&out, "[%s]", step)

			continue
		}

		if out.Len() > 0 {
			out.WriteString(".")
		}

		out.WriteString(step)
	}

	return out.String()
}

// reasonOf says what was wrong with it, without repeating the field.
func reasonOf(err *openapi3.SchemaError) string {
	if err.Reason != "" {
		return err.Reason
	}

	return err.Error()
}
