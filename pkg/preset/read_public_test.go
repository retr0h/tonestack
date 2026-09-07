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
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/catalog"
	"github.com/retr0h/tonestack/pkg/chain"
	"github.com/retr0h/tonestack/pkg/preset"
)

type ReadPublicTestSuite struct {
	suite.Suite
}

func (s *ReadPublicTestSuite) doc() *preset.Document {
	f, err := os.Open("testdata/minimal.hlx")
	s.Require().NoError(err)

	defer func() { s.Require().NoError(f.Close()) }()

	d, err := preset.Read(f)
	s.Require().NoError(err)

	return d
}

func (s *ReadPublicTestSuite) TestReadDecodesTheEnvelope() {
	d := s.doc()

	s.Require().Equal("L6Preset", d.Schema)
	s.Require().Equal(6, d.Version)
	s.Require().Equal(2162694, d.Data.Device)
	s.Require().Equal("Test Preset", d.Data.Meta.Name)
}

func (s *ReadPublicTestSuite) TestSpecOrdersBlocksByProcessorThenPosition() {
	spec, err := s.doc().Spec()

	s.Require().NoError(err)
	s.Require().Len(spec.Blocks, 3)
	s.Require().Equal(catalog.ModelID("HD2_CompressorLAStudioComp"), spec.Blocks[0].Model)
	s.Require().Equal(catalog.ModelID("HD2_AmpSVBeastNrm"), spec.Blocks[1].Model)
	s.Require().Equal(catalog.ModelID("HD2_ReverbSpring"), spec.Blocks[2].Model)
	s.Require().Equal(0, spec.Blocks[0].DSP)
	s.Require().Equal(1, spec.Blocks[2].DSP)
}

func (s *ReadPublicTestSuite) TestSpecSkipsRoutingEntries() {
	spec, _ := s.doc().Spec()

	for _, b := range spec.Blocks {
		s.Require().NotContains(string(b.Model), "AppDSPFlow",
			"inputs and outputs are routing, not links in a chain")
	}
}

func (s *ReadPublicTestSuite) TestSpecKeepsEachParameterKind() {
	spec, _ := s.doc().Spec()
	comp := spec.Blocks[0]

	f, ok := comp.Params["Gain"].Float()
	s.Require().True(ok)
	s.Require().InDelta(0.68, f, 1e-9)

	b, ok := comp.Params["Type"].Bool()
	s.Require().True(ok)
	s.Require().True(b)

	amp := spec.Blocks[1]
	i, ok := amp.Params["Taps"].Int()
	s.Require().True(ok)
	s.Require().Equal(int64(2), i)
}

func (s *ReadPublicTestSuite) TestSpecDoesNotTreatAttributesAsParameters() {
	spec, _ := s.doc().Spec()

	for _, b := range spec.Blocks {
		for k := range b.Params {
			s.Require().False(strings.HasPrefix(k, "@"), "attribute %q leaked into params", k)
		}
	}
}

func (s *ReadPublicTestSuite) TestSpecCarriesEnabledState() {
	spec, _ := s.doc().Spec()

	s.Require().True(spec.Blocks[0].Enabled)
	s.Require().False(spec.Blocks[2].Enabled, "a bypassed block stays bypassed")
}

func (s *ReadPublicTestSuite) TestReadRejectsSomethingThatIsNotAPreset() {
	f, err := os.Open("testdata/notapreset.hlx")
	s.Require().NoError(err)

	defer func() { s.Require().NoError(f.Close()) }()

	_, err = preset.Read(f)
	s.Require().ErrorIs(err, preset.ErrNotAPreset)
}

func (s *ReadPublicTestSuite) TestReadRejectsMalformedJSON() {
	_, err := preset.Read(strings.NewReader("{not json"))

	s.Require().Error(err)
}

func (s *ReadPublicTestSuite) TestReadReportsAMissingSchemaField() {
	_, err := preset.Read(strings.NewReader(`{"version":6}`))

	s.Require().ErrorIs(err, preset.ErrNotAPreset)
	s.Require().Contains(err.Error(), "no schema field")
}

func (s *ReadPublicTestSuite) TestDeviceVersionToleratesAString() {
	for _, doc := range []string{
		`{"schema":"L6Preset","data":{"device_version":57737216}}`,
		`{"schema":"L6Preset","data":{"device_version":"57737216"}}`,
		`{"schema":"L6Preset","data":{"device_version":"0.00"}}`,
		`{"schema":"L6Preset","data":{"device_version":""}}`,
	} {
		_, err := preset.Read(strings.NewReader(doc))
		s.Require().NoError(err, doc)
	}
}

func (s *ReadPublicTestSuite) TestDeviceVersionRejectsNonsense() {
	_, err := preset.Read(strings.NewReader(
		`{"schema":"L6Preset","data":{"device_version":"not a version"}}`))

	s.Require().Error(err)
}

func (s *ReadPublicTestSuite) TestRoundTripPreservesTheChain() {
	d := s.doc()
	before, err := d.Spec()
	s.Require().NoError(err)

	var buf bytes.Buffer
	s.Require().NoError(preset.Write(&buf, d))

	again, err := preset.Read(&buf)
	s.Require().NoError(err)

	after, err := again.Spec()
	s.Require().NoError(err)
	s.Require().Equal(before, after)
}

func (s *ReadPublicTestSuite) TestRoundTripKeepsRoutingAndSnapshots() {
	d := s.doc()

	var buf bytes.Buffer
	s.Require().NoError(preset.Write(&buf, d))

	s.Require().Contains(buf.String(), "AppDSPFlow1Input", "routing must survive")
	s.Require().Contains(buf.String(), "snapshot0", "snapshots must survive")
}

func (s *ReadPublicTestSuite) TestSetSpecReplacesTheChain() {
	d := s.doc()

	s.Require().NoError(d.SetSpec(chain.Chain{
		Name: "Replaced",
		Blocks: []chain.Block{{
			Model:   "HD2_AmpBrit2204",
			Params:  map[string]catalog.ParamValue{"Drive": catalog.Float(0.25)},
			DSP:     0,
			Pos:     0,
			Enabled: true,
		}},
	}))

	got, err := d.Spec()
	s.Require().NoError(err)
	s.Require().Equal("Replaced", got.Name)

	var onDSP0 int

	for _, b := range got.Blocks {
		if b.DSP == 0 {
			onDSP0++

			s.Require().Equal(catalog.ModelID("HD2_AmpBrit2204"), b.Model)
		}
	}

	s.Require().Equal(1, onDSP0, "the old blocks on that processor are gone")
}

func (s *ReadPublicTestSuite) TestNewBuildsAPresetFromNothing() {
	d, err := preset.New(2162694, chain.Chain{
		Name: "From Scratch",
		Blocks: []chain.Block{{
			Model:   "HD2_AmpSVBeastNrm",
			Params:  map[string]catalog.ParamValue{"Drive": catalog.Float(0.53)},
			Pos:     0,
			Enabled: true,
		}},
	})
	s.Require().NoError(err)

	var buf bytes.Buffer
	s.Require().NoError(preset.Write(&buf, d))

	again, err := preset.Read(&buf)
	s.Require().NoError(err)

	spec, err := again.Spec()
	s.Require().NoError(err)
	s.Require().Equal("From Scratch", spec.Name)
	s.Require().Len(spec.Blocks, 1)
	s.Require().Equal(catalog.ModelID("HD2_AmpSVBeastNrm"), spec.Blocks[0].Model)
}

func (s *ReadPublicTestSuite) TestEncodingRefusesAParameterNamedLikeAnAttribute() {
	_, err := preset.New(2162694, chain.Chain{
		Blocks: []chain.Block{{
			Model:  "HD2_AmpSVBeastNrm",
			Params: map[string]catalog.ParamValue{"@model": catalog.Enum("nope")},
		}},
	})

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "collides")
}

func TestReadPublicTestSuite(t *testing.T) {
	suite.Run(t, new(ReadPublicTestSuite))
}
