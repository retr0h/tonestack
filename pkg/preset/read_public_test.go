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
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/catalog"
	"github.com/retr0h/tonestack/pkg/chain"
	"github.com/retr0h/tonestack/pkg/preset"
)

// ReadPublicTestSuite covers reading a .hlx and the chain inside it.
type ReadPublicTestSuite struct {
	suite.Suite
}

// doc is the fixture preset, read.
func (s *ReadPublicTestSuite) doc() *preset.Document {
	f, err := os.Open("testdata/minimal.hlx")
	s.Require().NoError(err)

	defer func() { s.Require().NoError(f.Close()) }()

	d, err := preset.Read(f)
	s.Require().NoError(err)

	return d
}

// read is a shorthand for the many documents written inline below.
func (s *ReadPublicTestSuite) read(doc string) (*preset.Document, error) {
	return preset.Read(strings.NewReader(doc))
}

// TestRead decodes the envelope around a preset, and refuses what is not one.
func (s *ReadPublicTestSuite) TestRead() {
	tests := []struct {
		name string
		doc  string
		file string
		err  bool
		is   error
		says string
	}{
		{
			// A device always writes a number. Hand-made templates in the
			// wild do not, and refusing to read one because of a field
			// nothing depends on would be pedantry rather than correctness.
			name: "a device version written as a number",
			doc:  `{"schema":"L6Preset","data":{"device_version":57737216}}`,
		},
		{
			name: "one written as a string",
			doc:  `{"schema":"L6Preset","data":{"device_version":"57737216"}}`,
		},
		{
			name: "one written with a decimal point",
			doc:  `{"schema":"L6Preset","data":{"device_version":"0.00"}}`,
		},
		{
			name: "one left empty",
			doc:  `{"schema":"L6Preset","data":{"device_version":""}}`,
		},
		{
			name: "one that is not a version at all",
			doc:  `{"schema":"L6Preset","data":{"device_version":"not a version"}}`,
			err:  true,
		},
		{
			name: "one of neither kind",
			doc:  `{"schema":"L6Preset","data":{"device_version":{"nested":1}}}`,
			err:  true,
			says: "neither a number nor a string",
		},
		{
			name: "something that is not JSON",
			doc:  "{not json",
			err:  true,
		},
		{
			name: "a document naming no schema",
			doc:  `{"version":6}`,
			err:  true,
			is:   preset.ErrNotAPreset,
			says: "no schema field",
		},
		{
			name: "a setlist rather than a preset",
			file: "testdata/notapreset.hlx",
			err:  true,
			is:   preset.ErrNotAPreset,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var (
				got *preset.Document
				err error
			)

			if tt.file != "" {
				f, openErr := os.Open(tt.file)
				s.Require().NoError(openErr)

				defer func() { s.Require().NoError(f.Close()) }()

				got, err = preset.Read(f)
			} else {
				got, err = s.read(tt.doc)
			}

			if !tt.err {
				s.Require().NoError(err)
				s.Require().NotNil(got)

				return
			}

			s.Require().Error(err)

			if tt.is != nil {
				s.Require().ErrorIs(err, tt.is)
			}

			if tt.says != "" {
				s.Require().Contains(err.Error(), tt.says)
			}
		})
	}
}

// TestReadDecodesTheEnvelope covers the fields around the chain, which are
// one document rather than a set of cases.
func (s *ReadPublicTestSuite) TestReadDecodesTheEnvelope() {
	d := s.doc()

	s.Require().Equal("L6Preset", d.Schema)
	s.Require().Equal(6, d.Version)
	s.Require().Equal(2162694, d.Data.Device)
	s.Require().Equal("Test Preset", d.Data.Meta.Name)
}

// TestSpec extracts the signal chain a preset describes.
func (s *ReadPublicTestSuite) TestSpec() {
	tests := []struct {
		name  string
		doc   string
		check func(chain.Chain)
		says  string
	}{
		{
			name: "an entry naming no model is not a block",
			doc:  `{"schema":"L6Preset","data":{"tone":{"dsp0":{"block0":{"Gain":0.5}}}}}`,
			check: func(c chain.Chain) {
				s.Require().Empty(c.Blocks)
			},
		},
		{
			name: "an attribute nothing models is not a parameter",
			doc: `{"schema":"L6Preset","data":{"tone":{"dsp0":{"block0":` +
				`{"@model":"X","@position":0,"@no_snapshot_bypass":false,"Gain":0.5}}}}}`,
			check: func(c chain.Chain) {
				s.Require().Len(c.Blocks[0].Params, 1)
			},
		},
		{
			name: "a processor key nothing can number",
			doc:  `{"schema":"L6Preset","data":{"tone":{"dspX":{}}}}`,
			says: "dspX",
		},
		{
			name: "a block that is not an object",
			doc:  `{"schema":"L6Preset","data":{"tone":{"dsp0":{"block0":"nope"}}}}`,
			says: "block0",
		},
		{
			name: "a parameter of no kind at all",
			doc: `{"schema":"L6Preset","data":{"tone":{"dsp0":{"block0":` +
				`{"@model":"X","Gain":{"nested":1}}}}}}`,
			says: "Gain",
		},
		{
			name: "an attribute of the wrong kind",
			doc: `{"schema":"L6Preset","data":{"tone":{"dsp0":{"block0":` +
				`{"@model":"X","@position":"first"}}}}}`,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			d, err := s.read(tt.doc)
			s.Require().NoError(err)

			got, err := d.Spec()

			if tt.check == nil {
				s.Require().Error(err)

				if tt.says != "" {
					s.Require().Contains(err.Error(), tt.says)
				}

				return
			}

			s.Require().NoError(err)
			tt.check(got)
		})
	}
}

// TestSpecReadsTheFixture asks the same chain several questions, so it is one
// method rather than one per question.
func (s *ReadPublicTestSuite) TestSpecReadsTheFixture() {
	got, err := s.doc().Spec()
	s.Require().NoError(err)

	// Ordered by processor, then by position within it.
	s.Require().Len(got.Blocks, 3)
	s.Require().Equal(catalog.ModelID("HD2_CompressorLAStudioComp"), got.Blocks[0].Model)
	s.Require().Equal(catalog.ModelID("HD2_AmpSVBeastNrm"), got.Blocks[1].Model)
	s.Require().Equal(catalog.ModelID("HD2_ReverbSpring"), got.Blocks[2].Model)
	s.Require().Equal(0, got.Blocks[0].DSP)
	s.Require().Equal(1, got.Blocks[2].DSP)

	// A bypassed block stays bypassed.
	s.Require().True(got.Blocks[0].Enabled)
	s.Require().False(got.Blocks[2].Enabled)

	for _, b := range got.Blocks {
		s.Require().NotContains(string(b.Model), "AppDSPFlow",
			"inputs and outputs are routing, not links in a chain")

		for k := range b.Params {
			s.Require().False(strings.HasPrefix(k, "@"),
				"attribute %q leaked into params", k)
		}
	}

	// Every kind a parameter can be, kept as it arrived.
	f, ok := got.Blocks[0].Params["Gain"].Float()
	s.Require().True(ok)
	s.Require().InDelta(0.68, f, 1e-9)

	b, ok := got.Blocks[0].Params["Type"].Bool()
	s.Require().True(ok)
	s.Require().True(b)

	i, ok := got.Blocks[1].Params["Taps"].Int()
	s.Require().True(ok)
	s.Require().Equal(int64(2), i)
}

func TestReadPublicTestSuite(t *testing.T) {
	suite.Run(t, new(ReadPublicTestSuite))
}
