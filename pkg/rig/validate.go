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
// The types in gen/ are generated from schemas/rigspec.openapi.yaml, and
// generation gives them shape but not rules: nothing stops a required field
// being empty or an enumeration holding a word that is not in it. This is
// where the document's own constraints are enforced, so a rig that has been
// read, written or lifted can be asserted to still be one.
package rig

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/retr0h/tonestack/pkg/rig/gen"
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

// idPattern is the shape the schema states for an identifier.
var idPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// Validate reports whether a rig meets the contract in the schema.
//
// Checked in the order that gives the most useful first failure: the document
// says what it is, then what it is about, then what it holds.
func Validate(s gen.RigSpec) error {
	if !s.Schema.Valid() {
		return &InvalidError{
			Field:  "schema",
			Reason: fmt.Sprintf("is %q, which is not a rig", s.Schema),
		}
	}

	if !idPattern.MatchString(s.ID) {
		return &InvalidError{
			Field:  "id",
			Reason: fmt.Sprintf("%q is not lower case words joined by hyphens", s.ID),
		}
	}

	if err := validateSubject(s.Subject); err != nil {
		return err
	}

	if !s.Instrument.Valid() {
		return &InvalidError{
			Field:  "instrument",
			Reason: fmt.Sprintf("is %q, which is not guitar or bass", s.Instrument),
		}
	}

	if len(s.Chain) == 0 {
		return &InvalidError{Field: "chain", Reason: "holds nothing"}
	}

	return validateChain(s)
}

// validateSubject checks who or what a rig is attributed to.
func validateSubject(sub gen.Subject) error {
	if !sub.Kind.Valid() {
		return &InvalidError{
			Field:  "subject.kind",
			Reason: fmt.Sprintf("is %q, which is not a kind of subject", sub.Kind),
		}
	}

	if strings.TrimSpace(sub.Name) == "" {
		return &InvalidError{Field: "subject.name", Reason: "is empty"}
	}

	return nil
}

// validateChain checks every piece of gear and what is claimed about it.
func validateChain(s gen.RigSpec) error {
	for i, entry := range s.Chain {
		at := fmt.Sprintf("chain[%d]", i)

		if !entry.Role.Valid() {
			return &InvalidError{
				Field:  at + ".role",
				Reason: fmt.Sprintf("is %q, which is not a role", entry.Role),
			}
		}

		if strings.TrimSpace(entry.Gear) == "" {
			return &InvalidError{Field: at + ".gear", Reason: "is empty"}
		}

		if err := validateSettings(at, entry.Settings); err != nil {
			return err
		}

		if err := validateConfidence(at, entry.Confidence); err != nil {
			return err
		}

		if err := validateEvidence(at, entry.Evidence); err != nil {
			return err
		}
	}

	return validateMutations(s.Mutations)
}

// validateSettings checks the musical layer stays inside its stated range.
//
// A setting outside nought to one is not a value any device accepts, and it
// means the rig was written against a different idea of what these are.
func validateSettings(at string, settings *gen.Settings) error {
	if settings == nil {
		return nil
	}

	for key, v := range *settings {
		if v < 0 || v > 1 {
			return &InvalidError{
				Field:  at + ".settings." + key,
				Reason: fmt.Sprintf("is %g, outside nought to one", v),
			}
		}
	}

	return nil
}

// validateConfidence checks a stated confidence is one the schema knows.
func validateConfidence(at string, c *gen.Confidence) error {
	if c == nil || c.Valid() {
		return nil
	}

	return &InvalidError{
		Field:  at + ".confidence",
		Reason: fmt.Sprintf("is %q, which is not low, medium or high", *c),
	}
}

// validateEvidence checks each claim says how it came to be believed.
func validateEvidence(at string, evidence *[]gen.Evidence) error {
	if evidence == nil {
		return nil
	}

	for i, e := range *evidence {
		if !e.Kind.Valid() {
			return &InvalidError{
				Field:  fmt.Sprintf("%s.evidence[%d].kind", at, i),
				Reason: fmt.Sprintf("is %q, which is not a kind of evidence", e.Kind),
			}
		}
	}

	return nil
}

// validateMutations checks each round of correction says what was asked for.
//
// An entry with no ask is not a record of anything: the words somebody used
// are the part that cannot be reconstructed later.
func validateMutations(mutations *[]gen.Mutation) error {
	if mutations == nil {
		return nil
	}

	for i, m := range *mutations {
		if strings.TrimSpace(m.Ask) == "" {
			return &InvalidError{
				Field:  fmt.Sprintf("mutations[%d].ask", i),
				Reason: "is empty, so the entry records no request",
			}
		}
	}

	return nil
}
