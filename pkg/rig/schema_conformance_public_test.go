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
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/catalog"
	"github.com/retr0h/tonestack/pkg/rig"
)

// schemaPath points at the repository's canonical RigSpec definition.
const schemaPath = "../../schemas/rigspec.schema.json"

// SchemaConformancePublicTestSuite pins the Go types to the published schema.
// Without it the schema is documentation that can silently drift from the code
// it claims to describe.
type SchemaConformancePublicTestSuite struct {
	suite.Suite

	schema *jsonschema.Schema
}

func (s *SchemaConformancePublicTestSuite) SetupSuite() {
	f, err := os.Open(schemaPath)
	s.Require().
		NoError(err, "canonical schema must be readable at %s", schemaPath)

	defer func() { s.Require().NoError(f.Close()) }()

	doc, err := jsonschema.UnmarshalJSON(f)
	s.Require().NoError(err)

	c := jsonschema.NewCompiler()
	s.Require().NoError(c.AddResource("rigspec.schema.json", doc))

	s.schema, err = c.Compile("rigspec.schema.json")
	s.Require().NoError(err)
}

// validateMarshalled marshals v and validates the result against the schema.
func (s *SchemaConformancePublicTestSuite) validateMarshalled(v any) error {
	s.T().Helper()

	b, err := json.Marshal(v)
	s.Require().NoError(err)

	inst, err := jsonschema.UnmarshalJSON(bytes.NewReader(b))
	s.Require().NoError(err)

	return s.schema.Validate(inst)
}

func (s *SchemaConformancePublicTestSuite) TestAMinimalSpecConforms() {
	spec := rig.Spec{
		Name:   "Minimal",
		Origin: rig.OriginCurated,
		Blocks: []rig.SpecBlock{{
			Model:   "HD2_AmpTest",
			DSP:     0,
			Pos:     0,
			Enabled: true,
		}},
	}

	s.Require().NoError(s.validateMarshalled(spec))
}

func (s *SchemaConformancePublicTestSuite) TestEveryParamKindConforms() {
	spec := rig.Spec{
		Name:   "All Kinds",
		Origin: rig.OriginLLM,
		Blocks: []rig.SpecBlock{{
			Model: "HD2_AmpTest",
			Params: map[string]catalog.ParamValue{
				"Gain":   catalog.Float(0.5),
				"Taps":   catalog.Int(3),
				"Bright": catalog.Bool(true),
				"Mode":   catalog.Enum("Normal"),
			},
			DSP:     0,
			Pos:     0,
			Enabled: true,
		}},
	}

	s.Require().NoError(s.validateMarshalled(spec))
}

// TestSnapshotIndicesMarshalAsStringKeys pins the one place the Go type and the
// JSON shape genuinely differ: Go keys snapshot overrides by int, JSON keys
// every object by string. The schema declares the string form, so a change to
// the Go type that broke this would be caught here.
func (s *SchemaConformancePublicTestSuite) TestSnapshotIndicesMarshalAsStringKeys() {
	spec := rig.Spec{
		Name:   "With Snapshot",
		Origin: rig.OriginAudio,
		Blocks: []rig.SpecBlock{
			{Model: "HD2_AmpTest", DSP: 0, Pos: 0, Enabled: true},
		},
		Snapshots: []rig.Snapshot{{
			Name: "Lead",
			Overrides: map[int]map[string]catalog.ParamValue{
				0: {"Gain": catalog.Float(0.9)},
			},
		}},
	}

	b, err := json.Marshal(spec)
	s.Require().NoError(err)
	s.Require().
		Contains(string(b), `"0":`, "snapshot indices must marshal as string keys")

	s.Require().NoError(s.validateMarshalled(spec))
}

func (s *SchemaConformancePublicTestSuite) TestSchemaRejectsARigWithNoBlocks() {
	inst, err := jsonschema.UnmarshalJSON(bytes.NewReader(
		[]byte(`{"name":"Empty","origin":"curated","blocks":[]}`)))
	s.Require().NoError(err)

	s.Require().Error(s.schema.Validate(inst))
}

func (s *SchemaConformancePublicTestSuite) TestSchemaRejectsAnUnknownOrigin() {
	inst, err := jsonschema.UnmarshalJSON(bytes.NewReader([]byte(
		`{"name":"X","origin":"vibes","blocks":[{"model":"A","dsp":0,"pos":0,"enabled":true}]}`,
	)))
	s.Require().NoError(err)

	s.Require().Error(s.schema.Validate(inst))
}

func (s *SchemaConformancePublicTestSuite) TestSchemaRejectsAnUnknownProperty() {
	inst, err := jsonschema.UnmarshalJSON(bytes.NewReader([]byte(
		`{"name":"X","origin":"curated","vibe":"warm",` +
			`"blocks":[{"model":"A","dsp":0,"pos":0,"enabled":true}]}`)))
	s.Require().NoError(err)

	s.Require().Error(s.schema.Validate(inst))
}

func TestSchemaConformancePublicTestSuite(t *testing.T) {
	suite.Run(t, new(SchemaConformancePublicTestSuite))
}
