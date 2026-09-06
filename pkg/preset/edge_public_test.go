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
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/catalog"
	"github.com/retr0h/tonestack/pkg/preset"
	"github.com/retr0h/tonestack/pkg/rig"
)

type EdgePublicTestSuite struct {
	suite.Suite
}

// read is a shorthand for the many malformed documents below.
func (s *EdgePublicTestSuite) read(doc string) (*preset.Document, error) {
	return preset.Read(strings.NewReader(doc))
}

func (s *EdgePublicTestSuite) TestNotAPresetErrorNamesTheSchemaFound() {
	err := &preset.NotAPresetError{Schema: "L6Setlist"}

	s.Require().Contains(err.Error(), "L6Setlist")
	s.Require().ErrorIs(err, preset.ErrNotAPreset)
}

func (s *EdgePublicTestSuite) TestSpecReportsAnUnreadableProcessorKey() {
	d, err := s.read(`{"schema":"L6Preset","data":{"tone":{"dspX":{}}}}`)
	s.Require().NoError(err)

	_, err = d.Spec()
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "dspX")
}

func (s *EdgePublicTestSuite) TestSpecReportsAMalformedBlock() {
	d, err := s.read(`{"schema":"L6Preset","data":{"tone":{"dsp0":{"block0":"nope"}}}}`)
	s.Require().NoError(err)

	_, err = d.Spec()
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "block0")
}

func (s *EdgePublicTestSuite) TestSpecReportsAnUndecodableParameter() {
	d, err := s.read(
		`{"schema":"L6Preset","data":{"tone":{"dsp0":{"block0":` +
			`{"@model":"X","Gain":{"nested":1}}}}}}`)
	s.Require().NoError(err)

	_, err = d.Spec()
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "Gain")
}

func (s *EdgePublicTestSuite) TestSpecReportsAMalformedAttribute() {
	d, err := s.read(
		`{"schema":"L6Preset","data":{"tone":{"dsp0":{"block0":` +
			`{"@model":"X","@position":"first"}}}}}`)
	s.Require().NoError(err)

	_, err = d.Spec()
	s.Require().Error(err)
}

func (s *EdgePublicTestSuite) TestSpecSkipsABlockWithNoModel() {
	d, err := s.read(`{"schema":"L6Preset","data":{"tone":{"dsp0":{"block0":{"Gain":0.5}}}}}`)
	s.Require().NoError(err)

	spec, err := d.Spec()
	s.Require().NoError(err)
	s.Require().Empty(spec.Blocks, "an entry naming no model is not a block")
}

func (s *EdgePublicTestSuite) TestSpecIgnoresAnAttributeItDoesNotModel() {
	d, err := s.read(
		`{"schema":"L6Preset","data":{"tone":{"dsp0":{"block0":` +
			`{"@model":"X","@position":0,"@no_snapshot_bypass":false,"Gain":0.5}}}}}`)
	s.Require().NoError(err)

	spec, err := d.Spec()
	s.Require().NoError(err)
	s.Require().Len(spec.Blocks[0].Params, 1, "an unmodelled attribute is not a parameter")
}

func (s *EdgePublicTestSuite) TestDeviceVersionRejectsAValueOfNeitherKind() {
	_, err := s.read(`{"schema":"L6Preset","data":{"device_version":{"nested":1}}}`)

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "neither a number nor a string")
}

func (s *EdgePublicTestSuite) TestWriteReportsAFailingWriter() {
	d, err := preset.New(2162694, rig.Spec{})
	s.Require().NoError(err)

	s.Require().Error(preset.Write(&failingWriter{}, d))
}

func (s *EdgePublicTestSuite) TestSetSpecOnADocumentWithNoTone() {
	d := &preset.Document{Schema: "L6Preset"}

	s.Require().NoError(d.SetSpec(rig.Spec{
		Name:   "New",
		Blocks: []rig.SpecBlock{{Model: "HD2_AmpX", Pos: 0, Enabled: true}},
	}))

	spec, err := d.Spec()
	s.Require().NoError(err)
	s.Require().Len(spec.Blocks, 1)
}

func (s *EdgePublicTestSuite) TestSetSpecKeepsNonBlockEntries() {
	d, err := s.read(
		`{"schema":"L6Preset","data":{"tone":{"dsp0":{` +
			`"block0":{"@model":"Old","@position":0},"split":{"@model":"HD2_Split"}}}}}`)
	s.Require().NoError(err)

	s.Require().NoError(d.SetSpec(rig.Spec{
		Blocks: []rig.SpecBlock{{Model: "New", Pos: 0, Enabled: true}},
	}))

	var buf bytes.Buffer
	s.Require().NoError(preset.Write(&buf, d))
	s.Require().Contains(buf.String(), "HD2_Split", "routing survives a chain replacement")
	s.Require().NotContains(buf.String(), `"Old"`)
}

func (s *EdgePublicTestSuite) TestEncodingRejectsAZeroParamValue() {
	var zero catalog.ParamValue

	_, err := preset.New(2162694, rig.Spec{
		Blocks: []rig.SpecBlock{{Model: "X", Params: map[string]catalog.ParamValue{"Gain": zero}}},
	})

	s.Require().Error(err, "a value with no kind must not be written")
}

type failingWriter struct{}

func (*failingWriter) Write([]byte) (int, error) { return 0, errors.New("boom") }

func TestEdgePublicTestSuite(t *testing.T) {
	suite.Run(t, new(EdgePublicTestSuite))
}
