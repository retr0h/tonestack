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
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/catalog"
	"github.com/retr0h/tonestack/pkg/chain"
	"github.com/retr0h/tonestack/pkg/preset"
)

// WritePublicTestSuite covers writing a .hlx and putting a chain into one.
type WritePublicTestSuite struct {
	suite.Suite
}

// doc is the fixture preset, read.
func (s *WritePublicTestSuite) doc() *preset.Document {
	f, err := os.Open("testdata/minimal.hlx")
	s.Require().NoError(err)

	defer func() { s.Require().NoError(f.Close()) }()

	d, err := preset.Read(f)
	s.Require().NoError(err)

	return d
}

// TestWrite puts a document back the way it came.
func (s *WritePublicTestSuite) TestWrite() {
	tests := []struct {
		name     string
		to       io.Writer
		contains []string
		err      bool
	}{
		{
			name: "the routing and snapshots a chain says nothing about",
			contains: []string{
				"AppDSPFlow1Input", "snapshot0",
			},
		},
		{
			name: "nowhere to write it",
			to:   &failingWriter{},
			err:  true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var buf bytes.Buffer

			to := tt.to
			if to == nil {
				to = &buf
			}

			err := preset.Write(to, s.doc())

			if tt.err {
				s.Require().Error(err)

				return
			}

			s.Require().NoError(err)

			for _, want := range tt.contains {
				s.Require().Contains(buf.String(), want)
			}
		})
	}
}

// TestWriteSurvivesAReadBack is the rule the whole package exists to keep: a
// preset read and written again is the preset that was read.
func (s *WritePublicTestSuite) TestWriteSurvivesAReadBack() {
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

// TestSetSpec replaces the chain and leaves everything else alone.
func (s *WritePublicTestSuite) TestSetSpec() {
	tests := []struct {
		name string
		doc  *preset.Document
		// a document read from this, for shapes the fixture does not have.
		raw      string
		spec     chain.Chain
		contains []string
		absent   []string
	}{
		{
			name: "the blocks that were on that processor are gone",
			doc:  s.doc(),
			spec: chain.Chain{
				Name: "Replaced",
				Blocks: []chain.Block{{
					Model:   "HD2_AmpBrit2204",
					Params:  map[string]catalog.ParamValue{"Drive": catalog.Float(0.25)},
					Enabled: true,
				}},
			},
			contains: []string{"HD2_AmpBrit2204", "Replaced"},
			absent:   []string{"HD2_CompressorLAStudioComp"},
		},
		{
			name: "a document with no tone at all gets one",
			doc:  &preset.Document{Schema: "L6Preset"},
			spec: chain.Chain{
				Name:   "New",
				Blocks: []chain.Block{{Model: "HD2_AmpX", Enabled: true}},
			},
			contains: []string{"HD2_AmpX"},
		},
		{
			// The entries beside the chain are a device's own, and a chain
			// says nothing about them.
			name: "the routing, which survives a chain replacement",
			raw: `{"schema":"L6Preset","data":{"tone":{"dsp0":{` +
				`"block0":{"@model":"Old","@position":0},` +
				`"split":{"@model":"HD2_Split"}}}}}`,
			spec: chain.Chain{
				Blocks: []chain.Block{{Model: "New", Enabled: true}},
			},
			contains: []string{"HD2_Split"},
			absent:   []string{`"Old"`},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			doc := tt.doc

			if tt.raw != "" {
				got, err := preset.Read(strings.NewReader(tt.raw))
				s.Require().NoError(err)

				doc = got
			}

			s.Require().NoError(doc.SetSpec(tt.spec))

			var buf bytes.Buffer
			s.Require().NoError(preset.Write(&buf, doc))

			for _, want := range tt.contains {
				s.Require().Contains(buf.String(), want)
			}

			for _, gone := range tt.absent {
				s.Require().NotContains(buf.String(), gone)
			}
		})
	}
}

// TestNew builds a preset from a chain and nothing else.
func (s *WritePublicTestSuite) TestNew() {
	zero := catalog.ParamValue{}

	tests := []struct {
		name string
		spec chain.Chain
		err  bool
		says string
	}{
		{
			name: "a chain of one block",
			spec: chain.Chain{
				Name: "From Scratch",
				Blocks: []chain.Block{{
					Model:   "HD2_AmpSVBeastNrm",
					Params:  map[string]catalog.ParamValue{"Drive": catalog.Float(0.53)},
					Enabled: true,
				}},
			},
		},
		{
			// An attribute is the device's to set. A parameter named like
			// one would overwrite it in every preset generated.
			name: "a parameter named like an attribute",
			spec: chain.Chain{
				Blocks: []chain.Block{{
					Model:  "HD2_AmpSVBeastNrm",
					Params: map[string]catalog.ParamValue{"@model": catalog.Enum("nope")},
				}},
			},
			err:  true,
			says: "collides",
		},
		{
			name: "a value with no kind at all",
			spec: chain.Chain{
				Blocks: []chain.Block{{
					Model:  "X",
					Params: map[string]catalog.ParamValue{"Gain": zero},
				}},
			},
			err: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := preset.New(2162694, tt.spec)

			if tt.err {
				s.Require().Error(err)

				if tt.says != "" {
					s.Require().Contains(err.Error(), tt.says)
				}

				return
			}

			s.Require().NoError(err)

			var buf bytes.Buffer
			s.Require().NoError(preset.Write(&buf, got))

			again, err := preset.Read(&buf)
			s.Require().NoError(err)

			back, err := again.Spec()
			s.Require().NoError(err)
			s.Require().Equal(tt.spec.Name, back.Name)
			s.Require().Len(back.Blocks, len(tt.spec.Blocks))
		})
	}
}

type failingWriter struct{}

func (*failingWriter) Write([]byte) (int, error) { return 0, errors.New("boom") }

func TestWritePublicTestSuite(t *testing.T) {
	suite.Run(t, new(WritePublicTestSuite))
}
