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

package chain_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/chain"
)

type ValidateParamsPublicTestSuite struct {
	suite.Suite
}

// TestValidateParams checks every knob a chain sets against the catalog.
func (s *ValidateParamsPublicTestSuite) TestValidateParams() {
	tests := []struct {
		name string
		// the model the chain names, an amp the catalog has unless a case
		// says otherwise.
		model catalog.ModelID
		// parameters the block carries beside the ones the fixture declares.
		declared map[string]catalog.Param
		params   map[string]catalog.ParamValue

		err error
		// the parameter the failure must name.
		badKey  string
		errText string
		// run the case twenty times, for a failure that must name the same
		// parameter every run.
		stable bool
	}{
		{
			name: "values in range",
			params: map[string]catalog.ParamValue{
				"Gain": catalog.Float(0.5),
				"Mode": catalog.Enum("Bright"),
			},
		},
		{
			name:   "a float at the bottom of its range",
			params: map[string]catalog.ParamValue{"Gain": catalog.Float(0)},
		},
		{
			name:   "a float at the top of it",
			params: map[string]catalog.ParamValue{"Gain": catalog.Float(1)},
		},
		{
			name: "a switch, which has no range to check",
			declared: map[string]catalog.Param{
				"Bright": {
					Key: "Bright", Type: catalog.ParamBool,
					Default: catalog.Bool(false),
				},
			},
			params: map[string]catalog.ParamValue{"Bright": catalog.Bool(true)},
		},
		{
			name:   "a parameter the block does not have",
			params: map[string]catalog.ParamValue{"Nope": catalog.Float(0.5)},
			err:    catalog.ErrBadParam,
			badKey: "Nope",
		},
		{
			name:    "a float below its range",
			params:  map[string]catalog.ParamValue{"Gain": catalog.Float(-0.01)},
			err:     catalog.ErrBadParam,
			errText: "out of range",
		},
		{
			name:    "a float above it",
			params:  map[string]catalog.ParamValue{"Gain": catalog.Float(1.01)},
			err:     catalog.ErrBadParam,
			errText: "out of range",
		},
		{
			name: "an integer out of range",
			declared: map[string]catalog.Param{
				"Taps": {
					Key: "Taps", Type: catalog.ParamInt,
					Min: 1, Max: 4, Default: catalog.Int(1),
				},
			},
			params: map[string]catalog.ParamValue{"Taps": catalog.Int(9)},
			err:    catalog.ErrBadParam,
		},
		{
			name:    "an enum member nobody declared",
			params:  map[string]catalog.ParamValue{"Mode": catalog.Enum("Sparkle")},
			err:     catalog.ErrBadParam,
			errText: "Sparkle",
		},
		{
			name: "a kind the catalog invented",
			declared: map[string]catalog.Param{
				"Weird": {Key: "Weird", Type: catalog.ParamType("wat")},
			},
			params:  map[string]catalog.ParamValue{"Weird": catalog.Float(1)},
			err:     catalog.ErrBadParam,
			errText: "unknown kind",
		},
		{
			name:    "a float declared and an enum given",
			params:  map[string]catalog.ParamValue{"Gain": catalog.Enum("loud")},
			err:     catalog.ErrBadParam,
			errText: "expected float",
		},
		{
			name: "an integer declared and a float given",
			declared: map[string]catalog.Param{
				"Taps": {Key: "Taps", Type: catalog.ParamInt, Min: 0, Max: 9},
			},
			params:  map[string]catalog.ParamValue{"Taps": catalog.Float(1)},
			err:     catalog.ErrBadParam,
			errText: "expected",
		},
		{
			name: "a switch declared and an integer given",
			declared: map[string]catalog.Param{
				"Bright": {Key: "Bright", Type: catalog.ParamBool},
			},
			params:  map[string]catalog.ParamValue{"Bright": catalog.Int(1)},
			err:     catalog.ErrBadParam,
			errText: "expected",
		},
		{
			name:    "an enum declared and a switch given",
			params:  map[string]catalog.ParamValue{"Mode": catalog.Bool(true)},
			err:     catalog.ErrBadParam,
			errText: "expected",
		},
		{
			name:  "a model the catalog does not have",
			model: "HD2_Nope",
			err:   chain.ErrUnknownBlock,
		},
		{
			name: "two parameters the block does not have",
			params: map[string]catalog.ParamValue{
				"Zebra": catalog.Float(0.5),
				"Alpha": catalog.Float(0.5),
			},
			err:    catalog.ErrBadParam,
			badKey: "Alpha",
			stable: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			blk := testAmp()
			for key, p := range tt.declared {
				blk.Params[key] = p
			}

			model := tt.model
			if model == "" {
				model = "HD2_AmpTest"
			}

			spec := chain.Chain{
				Blocks: []chain.Block{{Model: model, Params: tt.params}},
			}

			runs := 1
			if tt.stable {
				runs = 20
			}

			for range runs {
				err := chain.ValidateParams(newCatalog(blk), spec)

				if tt.err == nil {
					s.Require().NoError(err)

					continue
				}

				s.Require().ErrorIs(err, tt.err)

				if tt.errText != "" {
					s.Require().Contains(err.Error(), tt.errText)
				}

				if tt.badKey == "" {
					continue
				}

				var target *catalog.BadParamError
				s.Require().True(errors.As(err, &target))
				s.Require().Equal(tt.badKey, target.Key,
					"and it must name the same one every run")
			}
		})
	}
}

func TestValidateParamsPublicTestSuite(t *testing.T) {
	suite.Run(t, new(ValidateParamsPublicTestSuite))
}
