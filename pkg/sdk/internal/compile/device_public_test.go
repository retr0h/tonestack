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

package compile_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/internal/compile"
	"github.com/retr0h/tonestack/pkg/sdk/preset"
	"github.com/retr0h/tonestack/pkg/sdk/rig"
)

// DevicePublicTestSuite covers what a hand-edited rig can put in the section
// a device wrote.
//
// Everything under `device` is carried verbatim, which means a person can
// type anything into it. Building a preset out of nonsense must leave the
// preset alone rather than fail or write nonsense through.
type DevicePublicTestSuite struct {
	suite.Suite

	cat *catalog.Catalog
}

func (s *DevicePublicTestSuite) SetupSuite() {
	var err error

	s.cat, err = catalog.BuiltIn()
	s.Require().NoError(err)
}

// rig returns a buildable rig carrying the given device state.
func (s *DevicePublicTestSuite) rig(state *rig.DeviceState) rig.Spec {
	return rig.Spec{
		Schema:     rig.SchemaName,
		ID:         "test",
		Subject:    rig.Subject{Kind: rig.KindSound, Name: "Test"},
		Instrument: rig.InstrumentBass,
		Chain: []rig.ChainEntry{
			{Role: rig.RoleAmp, Gear: "Ampeg SVT"},
		},
		Device: state,
	}
}

// TestLowerDeviceState builds a preset out of what a rig carries under
// `device`.
func (s *DevicePublicTestSuite) TestLowerDeviceState() {
	tests := []struct {
		name string
		// one entry of the tone section, or one of the routing.
		tone    map[string]json.RawMessage
		routing map[string]json.RawMessage

		contains string
		absent   string
	}{
		{
			// A person edited the file and put a string where a device wrote
			// a map. Dropping that one entry beats refusing to build the rest
			// of the rig.
			name: "a tone entry that is not an object",
			tone: map[string]json.RawMessage{
				"controller": json.RawMessage(`"nonsense"`),
			},
			absent: "nonsense",
		},
		{
			name: "a tone entry that will not read",
			tone: map[string]json.RawMessage{
				"controller": json.RawMessage(`[1, 2]`),
			},
			absent: `"controller": [`,
		},
		{
			// Routing is keyed by processor and entry — "dsp0.inputA". A key
			// with no processor names nowhere to put it.
			name: "routing naming no processor",
			routing: map[string]json.RawMessage{
				"inputA": json.RawMessage(`{"@model":"X"}`),
			},
			absent: `"@model":"X"`,
		},
		{
			name: "routing reaching a processor the preset lacks",
			routing: map[string]json.RawMessage{
				"dsp7.inputA": json.RawMessage(`{"@model":"HD2_AppDSPFlow1Input"}`),
			},
			contains: "dsp7",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			doc, err := preset.Blank()
			s.Require().NoError(err)

			state := &rig.DeviceState{}

			if tt.tone != nil {
				state.Tone = &tt.tone
			}

			if tt.routing != nil {
				state.Routing = &tt.routing
			}

			s.Require().NoError(compile.Lower(doc, s.rig(state), s.cat))

			var out bytes.Buffer
			s.Require().NoError(preset.Write(&out, doc))

			if tt.contains != "" {
				s.Require().Contains(out.String(), tt.contains)
			}

			if tt.absent != "" {
				s.Require().NotContains(out.String(), tt.absent)
			}
		})
	}
}

func TestDevicePublicTestSuite(t *testing.T) {
	suite.Run(t, new(DevicePublicTestSuite))
}
