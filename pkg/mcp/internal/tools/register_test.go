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

package tools

import (
	"encoding/json"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk"
	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/chain"
	"github.com/retr0h/tonestack/pkg/sdk/rig"
)

// RegisterTestSuite covers building a tool's output schema.
type RegisterTestSuite struct {
	suite.Suite
}

// TestMustOutputSchema covers that a realistic value, marshalled as the SDK
// marshals it, passes the schema the SDK validates it against.
func (s *RegisterTestSuite) TestMustOutputSchema() {
	// A block as a built chain holds it: every kind of parameter value, and
	// Attrs left nil, which marshals to null.
	block := chain.Block{
		Model: "HD2_AmpSVBeastBrt",
		Params: chain.Params{
			"Drive":  catalog.Float(0.41),
			"Ch":     catalog.Int(3),
			"Bright": catalog.Bool(true),
			"Mic":    catalog.Enum("57 Dynamic"),
		},
		Enabled: true,
	}
	version := json.RawMessage(`6`)

	tests := []struct {
		name   string
		schema func() *jsonschema.Schema
		value  any
		panics bool
	}{
		{
			name:   "a built chain",
			schema: func() *jsonschema.Schema { return mustOutputSchema[sdk.Made]() },
			value: sdk.Made{
				Chain: chain.Chain{Name: "Longview", Blocks: []chain.Block{block}},
				Path:  "longview.hlx",
			},
		},
		{
			name:   "a pedal's slots",
			schema: func() *jsonschema.Schema { return mustOutputSchema[sdk.Listing]() },
			value: sdk.Listing{
				Name:  "HX Stomp",
				Slots: []sdk.Held{{Slot: 0, Name: "Longview", Blocks: []chain.Block{block}}},
			},
		},
		{
			name:   "a catalog block",
			schema: func() *jsonschema.Schema { return mustOutputSchema[catalog.Block]() },
			value: catalog.Block{
				ID:       "HD2_AmpSVBeastBrt",
				Name:     "Ampeg SVT Brt",
				Category: catalog.CategoryAmp,
				Params: map[string]catalog.Param{
					"Drive": {
						Key:     "Drive",
						Type:    catalog.ParamFloat,
						Max:     1,
						Default: catalog.Float(0.41),
					},
				},
			},
		},
		{
			// Device state a rig keeps without modelling it: raw JSON behind
			// a pointer, and a map of it.
			name:   "a rig carrying device state",
			schema: func() *jsonschema.Schema { return mustOutputSchema[sdk.Recipe]() },
			value: sdk.Recipe{Rig: rig.Spec{
				ID:         "mike-dirnt",
				Instrument: rig.InstrumentBass,
				Schema:     rig.SchemaName,
				Device: &rig.DeviceState{
					Version: &version,
					Meta:    &map[string]json.RawMessage{"name": json.RawMessage(`"Longview"`)},
				},
			}},
		},
		{
			name:   "a type jsonschema cannot describe",
			schema: func() *jsonschema.Schema { return mustOutputSchema[chan int]() },
			panics: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			if tt.panics {
				s.PanicsWithValue(
					"mustOutputSchema[chan int]: For[chan int](): type chan int is unsupported by jsonschema",
					func() { tt.schema() },
				)
				return
			}

			resolved, err := tt.schema().Resolve(nil)
			s.Require().NoError(err)

			raw, err := json.Marshal(tt.value)
			s.Require().NoError(err)

			var instance any
			s.Require().NoError(json.Unmarshal(raw, &instance))

			s.NoError(resolved.Validate(instance))
		})
	}
}

func TestRegisterTestSuite(t *testing.T) {
	suite.Run(t, new(RegisterTestSuite))
}
