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

package lift_test

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/internal/lift"
	"github.com/retr0h/tonestack/pkg/catalog"
	"github.com/retr0h/tonestack/pkg/preset"
	riggen "github.com/retr0h/tonestack/pkg/rig/gen"
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
	f, err := os.Open("../../schemas/hx-stomp.catalog.json")
	s.Require().NoError(err)

	defer func() { s.Require().NoError(f.Close()) }()

	s.cat, err = catalog.Load(f)
	s.Require().NoError(err)
}

// rig returns a buildable rig carrying the given device state.
func (s *DevicePublicTestSuite) rig(state *riggen.DeviceState) riggen.RigSpec {
	return riggen.RigSpec{
		Schema:     riggen.RigSpecSchemaRigSpec,
		ID:         "test",
		Subject:    riggen.Subject{Kind: riggen.KindSound, Name: "Test"},
		Instrument: riggen.InstrumentBass,
		Chain: []riggen.ChainEntry{
			{Role: riggen.RoleAmp, Gear: "Ampeg SVT"},
		},
		Device: state,
	}
}

func (s *DevicePublicTestSuite) TestAToneEntryThatIsNotAnObjectIsIgnored() {
	// A person edited the file and put a string where a device wrote a map.
	// Dropping that one entry beats refusing to build the rest of the rig.
	doc, err := preset.Blank()
	s.Require().NoError(err)

	tone := map[string]json.RawMessage{"controller": json.RawMessage(`"nonsense"`)}
	s.Require().NoError(lift.Lower(doc, s.rig(&riggen.DeviceState{Tone: &tone}), s.cat))

	var out bytes.Buffer
	s.Require().NoError(preset.Write(&out, doc))
	s.Require().NotContains(out.String(), "nonsense")
}

func (s *DevicePublicTestSuite) TestRoutingNotNamingAProcessorIsIgnored() {
	// Routing is keyed by processor and entry — "dsp0.inputA". A key with no
	// processor names nowhere to put it.
	doc, err := preset.Blank()
	s.Require().NoError(err)

	routing := map[string]json.RawMessage{"inputA": json.RawMessage(`{"@model":"X"}`)}
	s.Require().NoError(lift.Lower(doc, s.rig(&riggen.DeviceState{Routing: &routing}), s.cat))

	var out bytes.Buffer
	s.Require().NoError(preset.Write(&out, doc))
	s.Require().NotContains(out.String(), `"@model":"X"`)
}

func (s *DevicePublicTestSuite) TestRoutingReachesAProcessorThePresetLacks() {
	doc, err := preset.Blank()
	s.Require().NoError(err)

	routing := map[string]json.RawMessage{
		"dsp7.inputA": json.RawMessage(`{"@model":"HD2_AppDSPFlow1Input"}`),
	}
	s.Require().NoError(lift.Lower(doc, s.rig(&riggen.DeviceState{Routing: &routing}), s.cat))

	var out bytes.Buffer
	s.Require().NoError(preset.Write(&out, doc))
	s.Require().Contains(out.String(), "dsp7")
}

func TestDevicePublicTestSuite(t *testing.T) {
	suite.Run(t, new(DevicePublicTestSuite))
}
