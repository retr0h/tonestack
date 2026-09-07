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

package preset_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/chain"
	"github.com/retr0h/tonestack/pkg/preset"
)

// A preset this package rewrote must differ from the original only where
// somebody asked it to. Everything here is a way that silently failed to be
// true.
type FidelityPublicTestSuite struct {
	suite.Suite
}

// roundTrip reads a preset and writes it straight back.
func (s *FidelityPublicTestSuite) roundTrip(in string) string {
	doc, err := preset.Read(bytes.NewReader([]byte(in)))
	s.Require().NoError(err)

	var out bytes.Buffer
	s.Require().NoError(preset.Write(&out, doc))

	return s.canonical(out.Bytes())
}

func (s *FidelityPublicTestSuite) canonical(raw []byte) string {
	var v any
	s.Require().NoError(json.Unmarshal(raw, &v))

	out, err := json.Marshal(v)
	s.Require().NoError(err)

	return string(out)
}

func (s *FidelityPublicTestSuite) TestKeepsWhatItDoesNotModel() {
	tests := []struct {
		name string
		in   string
	}{
		{
			"metadata nobody documented",
			`{"schema":"L6Preset","version":6,"data":{"device":2162694,
			  "device_version":0,"meta":{"name":"X","song":"Longview",
			  "band":"Green Day","author":"someone","tnid":7},"tone":{}}}`,
		},
		{
			"an empty string, which is not the same as an absent field",
			`{"schema":"L6Preset","version":6,"data":{"device":2162694,
			  "device_version":0,"meta":{"name":"X","build_sha":""},"tone":{}}}`,
		},
		{
			"a version written as a string",
			`{"schema":"L6Preset","version":6,"data":{"device":2162694,
			  "device_version":"0","meta":{"name":"X"},"tone":{}}}`,
		},
		{
			"a version written as a decimal string",
			`{"schema":"L6Preset","version":6,"data":{"device":2162694,
			  "device_version":"0.00","meta":{"name":"X"},"tone":{}}}`,
		},
		{
			"block attributes this package has no opinion about",
			`{"schema":"L6Preset","version":6,"data":{"device":2162694,
			  "device_version":0,"meta":{"name":"X"},"tone":{"dsp0":{"block0":{
			  "@model":"HD2_AmpTest","@position":0,"@enabled":true,"@path":0,
			  "@stereo":false,"@type":7,"@trails":true,
			  "@no_snapshot_bypass":false,"Drive":0.5}}}}}`,
		},
		{
			"a block whose key and position disagree",
			`{"schema":"L6Preset","version":6,"data":{"device":2162694,
			  "device_version":0,"meta":{"name":"X"},"tone":{"dsp0":{"block5":{
			  "@model":"HD2_AmpTest","@position":6,"@enabled":true}}}}}`,
		},
		{
			"a switch, which is not a number",
			`{"schema":"L6Preset","version":6,"data":{"device":2162694,
			  "device_version":0,"meta":{"name":"X"},"tone":{"dsp0":{"block0":{
			  "@model":"HD2_AmpTest","@position":0,"@enabled":true,
			  "TempoSync1":true,"Voicing":"Modern"}}}}}`,
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			s.Require().Equal(s.canonical([]byte(tc.in)), s.roundTrip(tc.in))
		})
	}
}

func (s *FidelityPublicTestSuite) TestRefusesWhatItCannotRead() {
	tests := []struct {
		name    string
		in      string
		message string
	}{
		{
			"metadata that is not an object",
			`{"schema":"L6Preset","version":6,"data":{"device":2162694,
			  "meta":"nope","tone":{}}}`,
			"decoding preset metadata",
		},
		{
			"a name that is not a string",
			`{"schema":"L6Preset","version":6,"data":{"device":2162694,
			  "meta":{"name":7},"tone":{}}}`,
			"decoding preset name",
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			_, err := preset.Read(bytes.NewReader([]byte(tc.in)))

			s.Require().Error(err)
			s.Require().Contains(err.Error(), tc.message)
		})
	}
}

func (s *FidelityPublicTestSuite) TestWritingAChainKeepsItsAttributes() {
	// Building a preset from a chain is a different path from rewriting one
	// that was read, and it has to carry the same things.
	doc, err := preset.Read(bytes.NewReader([]byte(
		`{"schema":"L6Preset","version":6,"data":{"device":2162694,
		  "device_version":0,"meta":{"name":"X"},"tone":{}}}`)))
	s.Require().NoError(err)

	s.Require().NoError(doc.SetSpec(chain.Chain{
		Name: "X",
		Blocks: []chain.Block{{
			Model: "HD2_AmpTest", Pos: 5, Enabled: true,
			Attrs: map[string]json.RawMessage{
				"@position": json.RawMessage("6"),
				"@trails":   json.RawMessage("true"),
				// An attribute colliding with one the chain models is the
				// chain's to decide, not the attribute's.
				"@enabled": json.RawMessage("false"),
			},
		}},
	}))

	var out bytes.Buffer
	s.Require().NoError(preset.Write(&out, doc))

	got := out.String()
	s.Require().Contains(got, `"block5"`, "the key comes from the position field")
	s.Require().Contains(got, `"@position": 6`, "the attribute is what was carried")
	s.Require().Contains(got, `"@trails": true`)
	s.Require().Contains(got, `"@enabled": true`, "the chain decides what it models")
}

func (s *FidelityPublicTestSuite) TestRefusesABlockKeyThatIsNotNumbered() {
	doc, err := preset.Read(bytes.NewReader([]byte(
		`{"schema":"L6Preset","version":6,"data":{"device":2162694,
		  "meta":{"name":"X"},"tone":{"dsp0":{"blockX":{
		  "@model":"HD2_AmpTest","@position":0}}}}}`)))
	s.Require().NoError(err)

	_, err = doc.Spec()

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "not numbered")
}

func TestFidelityPublicTestSuite(t *testing.T) {
	suite.Run(t, new(FidelityPublicTestSuite))
}
