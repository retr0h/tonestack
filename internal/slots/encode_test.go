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

package slots

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/internal/catalogview"
	"github.com/retr0h/tonestack/pkg/catalog"
	"github.com/retr0h/tonestack/pkg/chain"
	"github.com/retr0h/tonestack/pkg/preset"
	"github.com/retr0h/tonestack/pkg/sdk/wire"
)

// EncodeTestSuite covers turning a preset back into what a device lays out.
type EncodeTestSuite struct {
	suite.Suite

	cat *catalog.Catalog
}

func (s *EncodeTestSuite) SetupSuite() {
	cat, err := catalogview.Open("")
	s.Require().NoError(err)

	s.cat = cat
}

// capture returns one slot as an HX Stomp actually sent it.
func (s *EncodeTestSuite) capture(name string) []byte {
	raw, err := os.ReadFile(
		filepath.Join("..", "..", "pkg", "sdk", "wire", "testdata", name))
	s.Require().NoError(err)

	return raw
}

// asPreset rebuilds what the reading commands produce for a device's answer.
func (s *EncodeTestSuite) asPreset(
	name string,
	got wire.DevicePreset,
) *preset.Document {
	c, err := chainOf(name, got, s.cat)
	s.Require().NoError(err)

	doc, err := preset.Blank()
	s.Require().NoError(err)

	doc.Data.Device = s.cat.DeviceID
	s.Require().NoError(doc.SetSpec(c))

	// A cabinet an amp carries is not in the chain. A preset keeps it beside
	// the routing, which is where the reading path puts it and where the
	// writing path looks for it.
	cabs := map[string]json.RawMessage{}
	pairedCabs(cabs, got, s.cat)

	for key, body := range cabs {
		doc.Data.Tone[processorKey][strings.TrimPrefix(key, processorKey+".")] = body
	}

	return doc
}

// TestAPresetSurvivesGoingBackToTheDevice is the claim this file exists for.
//
// Every capture is read the way `presets show` reads one, written back the
// way an import writes one, and read again. A block that comes back changed
// is a tone somebody would hear go wrong.
func (s *EncodeTestSuite) TestAPresetSurvivesGoingBackToTheDevice() {
	for _, name := range []string{"preset.bin", "switches.bin", "empty.bin"} {
		s.Run(name, func() {
			was, err := wire.DecodePreset(s.capture(name))
			s.Require().NoError(err)

			blocks, err := placementsOf(s.asPreset(name, was), s.cat)
			s.Require().NoError(err)

			out, err := wire.Blank()
			s.Require().NoError(err)
			s.Require().NoError(wire.PlaceAsWritten(out, blocks))

			back, err := wire.DecodePreset(out.Encode())
			s.Require().NoError(err)

			s.Require().Equal(was.Blocks, back.Blocks)
		})
	}
}

// TestPlacementsOfReportsWhatItCannotWrite covers the ways a preset can name
// something a device has no number for.
func (s *EncodeTestSuite) TestPlacementsOfReportsWhatItCannotWrite() {
	tests := []struct {
		name string
		cat  *catalog.Catalog
		tone map[string]json.RawMessage
		want string
	}{
		{
			name: "a catalog with no model table",
			cat:  &catalog.Catalog{},
			want: "catalog generate",
		},
		{
			name: "a model the table does not carry",
			tone: map[string]json.RawMessage{
				"block0": json.RawMessage(
					`{"@model":"HD2_NoSuchThing","@position":1,"@enabled":true}`),
			},
			want: "does not carry",
		},
		{
			name: "an amp naming a cabinet the preset does not hold",
			tone: map[string]json.RawMessage{
				"block0": json.RawMessage(
					`{"@model":"HD2_AmpTucknGo","@position":1,"@enabled":true,` +
						`"@cab":"cab7"}`),
			},
			want: "which the preset does not hold",
		},
		{
			name: "a cabinet entry that is not an entry",
			tone: map[string]json.RawMessage{
				"block0": json.RawMessage(
					`{"@model":"HD2_AmpTucknGo","@position":1,"@enabled":true,` +
						`"@cab":"cab0"}`),
				"cab0": json.RawMessage(`5`),
			},
			want: "reading cabinet",
		},
		{
			name: "a cabinet naming no model",
			tone: map[string]json.RawMessage{
				"block0": json.RawMessage(
					`{"@model":"HD2_AmpTucknGo","@position":1,"@enabled":true,` +
						`"@cab":"cab0"}`),
				"cab0": json.RawMessage(`{"Level":1}`),
			},
			want: "names no model",
		},
		{
			name: "a cabinet the table does not carry",
			tone: map[string]json.RawMessage{
				"block0": json.RawMessage(
					`{"@model":"HD2_AmpTucknGo","@position":1,"@enabled":true,` +
						`"@cab":"cab0"}`),
				"cab0": json.RawMessage(`{"@model":"HD2_NoSuchCab"}`),
			},
			want: "does not carry",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			cat := tt.cat
			if cat == nil {
				cat = s.cat
			}

			doc, err := preset.Blank()
			s.Require().NoError(err)

			for key, body := range tt.tone {
				doc.Data.Tone[processorKey][key] = body
			}

			_, err = placementsOf(doc, cat)

			s.Require().Error(err)
			s.Require().Contains(err.Error(), tt.want)
		})
	}
}

// TestPlacementsOfReportsAnUnreadablePreset covers a document whose chain
// cannot be read at all.
func (s *EncodeTestSuite) TestPlacementsOfReportsAnUnreadablePreset() {
	doc, err := preset.Blank()
	s.Require().NoError(err)

	doc.Data.Tone[processorKey]["block0"] = json.RawMessage(`"not a block"`)

	_, err = placementsOf(doc, s.cat)
	s.Require().Error(err)
}

// TestValuesFollowTheCatalogsWord covers the typing JSON cannot carry.
func (s *EncodeTestSuite) TestValuesFollowTheCatalogsWord() {
	sym := catalog.Symbol{Params: []string{"a", "b", "c", "d", "e"}}
	types := map[string]catalog.Param{
		"a": {Type: catalog.ParamFloat},
		"b": {Type: catalog.ParamInt},
		"c": {Type: catalog.ParamBool},
		"d": {Type: catalog.ParamEnum},
	}

	tests := []struct {
		name   string
		params map[string]catalog.ParamValue
		want   []any
	}{
		{
			name: "a whole number the catalog calls a fraction",
			params: map[string]catalog.ParamValue{
				"a": catalog.Int(3),
				"b": catalog.Float(4),
				"c": catalog.Bool(true),
				"d": catalog.Float(2),
			},
			want: []any{float64(3), int64(4), true, int64(2), float64(0)},
		},
		{
			name:   "a preset that carries none of them",
			params: nil,
			want:   []any{float64(0), int64(0), false, int64(0), float64(0)},
		},
		{
			name: "values already of the kind the catalog states",
			params: map[string]catalog.ParamValue{
				"a": catalog.Float(0.25),
				"b": catalog.Int(7),
				"c": catalog.Bool(false),
				"d": catalog.Int(1),
				"e": catalog.Bool(true),
			},
			want: []any{0.25, int64(7), false, int64(1), true},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.want, valuesOf(sym, tt.params, types, nil))
		})
	}
}

// TestValuesOfAModelWithNoParameters covers a block the table names but
// gives nothing to set.
func (s *EncodeTestSuite) TestValuesOfAModelWithNoParameters() {
	s.Require().Nil(valuesOf(catalog.Symbol{}, nil, nil, nil))
}

// TestMicOf covers the value a cabinet sends past its named ones.
func (s *EncodeTestSuite) TestMicOf() {
	tests := []struct {
		name   string
		fields map[string]json.RawMessage
		want   []any
	}{
		{
			name:   "a cabinet naming its microphone",
			fields: map[string]json.RawMessage{cabMic: json.RawMessage(`11`)},
			want:   []any{int64(11)},
		},
		{
			name:   "one that names none",
			fields: nil,
		},
		{
			name:   "one whose microphone is not a value",
			fields: map[string]json.RawMessage{cabMic: json.RawMessage(`{}`)},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.want, micOf(tt.fields))
		})
	}
}

// TestClassOf covers what a block tells the device it is.
func (s *EncodeTestSuite) TestClassOf() {
	tests := []struct {
		name   string
		model  catalog.ModelID
		paired bool
		want   int
	}{
		{
			name:  "an amp on its own",
			model: "HD2_AmpTucknGo",
			want:  wire.ClassAmp,
		},
		{
			name:   "an amp carrying its cabinet",
			model:  "HD2_AmpTucknGo",
			paired: true,
			want:   wire.ClassAmpCab,
		},
		{
			name:  "a cabinet",
			model: "HD2_Cab1x15TucknGo",
			want:  wire.ClassCab,
		},
		{
			name:  "an effect",
			model: "HD2_DistTeemah",
			want:  wire.ClassEffect,
		},
		{
			name:  "a model the catalog does not carry",
			model: "HD2_NoSuchThing",
			want:  wire.ClassEffect,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.want, classOf(tt.model, s.cat, tt.paired))
		})
	}
}

// TestCabNameOf covers both places a preset can name a block's cabinet.
func (s *EncodeTestSuite) TestCabNameOf() {
	tests := []struct {
		name  string
		block chain.Block
		want  string
		found bool
	}{
		{
			name: "a chain built here, where it is a parameter",
			block: chain.Block{
				Params: chain.Params{attrCab: catalog.Enum("cab0")},
			},
			want:  "cab0",
			found: true,
		},
		{
			name: "one read back out of a preset, where it is an attribute",
			block: chain.Block{
				Attrs: map[string]json.RawMessage{attrCab: json.RawMessage(`"cab1"`)},
			},
			want:  "cab1",
			found: true,
		},
		{
			name:  "a block carrying no cabinet",
			block: chain.Block{},
		},
		{
			name: "one whose cabinet is not a name",
			block: chain.Block{
				Attrs: map[string]json.RawMessage{attrCab: json.RawMessage(`7`)},
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, ok := cabNameOf(tt.block)

			s.Require().Equal(tt.found, ok)
			s.Require().Equal(tt.want, got)
		})
	}
}

// TestParamsOfJSON covers reading a cabinet entry's settings.
func (s *EncodeTestSuite) TestParamsOfJSON() {
	sym := catalog.Symbol{Params: []string{"Level", "LowCut"}}

	tests := []struct {
		name   string
		fields map[string]json.RawMessage
		want   map[string]catalog.ParamValue
	}{
		{
			name: "an entry carrying both",
			fields: map[string]json.RawMessage{
				"Level":  json.RawMessage(`0.5`),
				"LowCut": json.RawMessage(`20`),
			},
			want: map[string]catalog.ParamValue{
				"Level":  catalog.Float(0.5),
				"LowCut": catalog.Int(20),
			},
		},
		{
			name:   "one carrying neither",
			fields: nil,
			want:   map[string]catalog.ParamValue{},
		},
		{
			name: "one whose value is not a value",
			fields: map[string]json.RawMessage{
				"Level": json.RawMessage(`{}`),
			},
			want: map[string]catalog.ParamValue{},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.want, paramsOfJSON(sym, tt.fields))
		})
	}
}

// TestTypesOf covers what the catalog says a model's parameters hold.
func (s *EncodeTestSuite) TestTypesOf() {
	s.Require().NotEmpty(typesOf("HD2_AmpTucknGo", s.cat))
	s.Require().Nil(typesOf("HD2_NoSuchThing", s.cat))
}

func TestEncodeTestSuite(t *testing.T) {
	suite.Run(t, new(EncodeTestSuite))
}
