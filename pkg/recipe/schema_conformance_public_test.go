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
package recipe_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/stretchr/testify/suite"
	"sigs.k8s.io/yaml"

	"github.com/retr0h/tonestack/pkg/recipe"
)

// schemaPath is the published Recipe contract.
const schemaPath = "../../schemas/recipe.schema.json"

// recipesDir holds every shipped recipe.
const recipesDir = "../../recipes"

// SchemaConformancePublicTestSuite pins the shipped recipes to the contract
// this project publishes.
//
// The Go loader checks what the type system cannot, but it is not the schema:
// a reader in another language validates against the JSON Schema, and if our
// own data does not satisfy it then the contract is a lie. This is the only
// place the two are compared.
type SchemaConformancePublicTestSuite struct {
	suite.Suite

	schema *jsonschema.Schema
}

func (s *SchemaConformancePublicTestSuite) SetupSuite() {
	f, err := os.Open(schemaPath)
	s.Require().NoError(err)

	defer func() { s.Require().NoError(f.Close()) }()

	doc, err := jsonschema.UnmarshalJSON(f)
	s.Require().NoError(err)

	c := jsonschema.NewCompiler()
	s.Require().NoError(c.AddResource("recipe.schema.json", doc))

	s.schema, err = c.Compile("recipe.schema.json")
	s.Require().NoError(err)
}

func (s *SchemaConformancePublicTestSuite) TestEveryShippedRecipeConforms() {
	paths, err := filepath.Glob(filepath.Join(recipesDir, "*", "*.yaml"))
	s.Require().NoError(err)
	s.Require().NotEmpty(paths, "no recipes found to check")

	for _, path := range paths {
		s.Run(filepath.Base(path), func() {
			raw, err := os.ReadFile(path)
			s.Require().NoError(err)

			asJSON, err := yaml.YAMLToJSON(raw)
			s.Require().NoError(err)

			inst, err := jsonschema.UnmarshalJSON(bytes.NewReader(asJSON))
			s.Require().NoError(err)
			s.Require().NoError(s.schema.Validate(inst))

			// The loader must accept everything the schema does.
			r, err := recipe.Load(bytes.NewReader(raw))
			s.Require().NoError(err)

			// The filename must match the identifier, so a recipe can be
			// found by name without opening every file.
			s.Require().Equal(
				r.ID+".yaml", filepath.Base(path),
				"filename must match id")
		})
	}
}

func (s *SchemaConformancePublicTestSuite) TestSchemaRejectsWhatTheLoaderRejects() {
	bad := map[string]any{
		"id": "x", "kind": "banjo", "name": "X",
		"instrument_type": "bass",
		"rig":             map[string]any{"amp": "Ampeg SVT"},
		"provenance":      map[string]any{"source": "llm", "confidence": "low"},
	}

	raw, err := json.Marshal(bad)
	s.Require().NoError(err)

	inst, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
	s.Require().NoError(err)
	s.Require().Error(s.schema.Validate(inst), "schema must reject an unknown kind")
}

func TestSchemaConformancePublicTestSuite(t *testing.T) {
	suite.Run(t, new(SchemaConformancePublicTestSuite))
}
