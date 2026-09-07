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
// Package recipe reads curated knowledge about how a sound is built.
//
// The shape of a recipe is generated from schemas/recipe.openapi.yaml into
// pkg/recipe/gen. This package holds what cannot be generated: reading a file,
// and the checks a JSON Schema expresses but Go's type system does not.
package recipe

import (
	"fmt"
	"io"
	"regexp"

	"sigs.k8s.io/yaml"

	"github.com/retr0h/tonestack/pkg/recipe/gen"
)

// idPattern is the identifier form the contract requires: lowercase words
// joined by single hyphens.
var idPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// Load reads one recipe and checks it against the contract.
//
// Decoding alone is not enough. Go accepts any string where an enumerated
// value is wanted, so `kind: banjo` would unmarshal happily; the checks here
// are what the type system cannot express.
func Load(r io.Reader) (*gen.Recipe, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading recipe: %w", err)
	}

	var rec gen.Recipe
	if err := yaml.Unmarshal(raw, &rec); err != nil {
		return nil, fmt.Errorf("decoding recipe: %w", err)
	}

	if err := Validate(&rec); err != nil {
		return nil, err
	}

	return &rec, nil
}

// Validate reports the first way a recipe fails the contract.
func Validate(r *gen.Recipe) error {
	if !idPattern.MatchString(r.ID) {
		return &InvalidError{r.ID, "id", "must be lowercase words joined by hyphens"}
	}

	if r.Name == "" {
		return &InvalidError{r.ID, "name", "must not be empty"}
	}

	if !r.Kind.Valid() {
		return &InvalidError{r.ID, "kind", fmt.Sprintf("%q is not artist or genre", r.Kind)}
	}

	if !r.InstrumentType.Valid() {
		return &InvalidError{
			r.ID, "instrument_type",
			fmt.Sprintf("%q is not guitar or bass", r.InstrumentType),
		}
	}

	if r.Rig.Amp == "" {
		return &InvalidError{r.ID, "rig.amp", "must name real-world gear"}
	}

	if r.Variants != nil {
		for _, v := range *r.Variants {
			if err := validateVariant(r.ID, v); err != nil {
				return err
			}
		}
	}

	return validateProvenance(r.ID, r.Provenance)
}

// validateVariant reports a variant that does not satisfy the contract.
func validateVariant(id string, v gen.Variant) error {
	if !idPattern.MatchString(v.ID) {
		return &InvalidError{
			id, "variants." + v.ID + ".id",
			"must be lowercase words joined by hyphens",
		}
	}

	if v.Name == "" {
		return &InvalidError{id, "variants." + v.ID + ".name", "must not be empty"}
	}

	return nil
}

// validateProvenance reports a provenance that does not satisfy the contract.
func validateProvenance(id string, p gen.Provenance) error {
	if !p.Source.Valid() {
		return &InvalidError{
			id, "provenance.source",
			fmt.Sprintf("%q is not llm, curated or cited", p.Source),
		}
	}

	if !p.Confidence.Valid() {
		return &InvalidError{
			id, "provenance.confidence",
			fmt.Sprintf("%q is not low, medium or high", p.Confidence),
		}
	}

	return nil
}

// Trusted reports whether a recipe's gear identification was confirmed by a
// person rather than asserted by a model.
func Trusted(p gen.Provenance) bool {
	return p.Source == gen.SourceCurated || p.Source == gen.SourceCited
}
