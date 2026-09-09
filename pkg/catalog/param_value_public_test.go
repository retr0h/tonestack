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

package catalog_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/catalog"
)

type ParamValuePublicTestSuite struct {
	suite.Suite
}

// TestAccessors covers what a value says it is and what it hands back.
//
// Each case asks a value for all four kinds: the one it holds answers with
// the value, and the other three refuse rather than converting.
func (s *ParamValuePublicTestSuite) TestAccessors() {
	tests := []struct {
		name string
		val  catalog.ParamValue
		kind catalog.ParamType
		want any
	}{
		{
			name: "a float",
			val:  catalog.Float(0.25),
			kind: catalog.ParamFloat,
			want: 0.25,
		},
		{
			name: "an integer",
			val:  catalog.Int(7),
			kind: catalog.ParamInt,
			want: int64(7),
		},
		{
			name: "a switch",
			val:  catalog.Bool(true),
			kind: catalog.ParamBool,
			want: true,
		},
		{
			name: "an enumerated string",
			val:  catalog.Enum("Bright"),
			kind: catalog.ParamEnum,
			want: "Bright",
		},
		{name: "a value with no kind"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.kind, tt.val.Type())

			f, ok := tt.val.Float()
			s.Require().Equal(tt.kind == catalog.ParamFloat, ok)

			if ok {
				s.Require().InDelta(tt.want, f, 1e-9)
			}

			i, ok := tt.val.Int()
			s.Require().Equal(tt.kind == catalog.ParamInt, ok)

			if ok {
				s.Require().Equal(tt.want, i)
			}

			b, ok := tt.val.Bool()
			s.Require().Equal(tt.kind == catalog.ParamBool, ok)

			if ok {
				s.Require().Equal(tt.want, b)
			}

			e, ok := tt.val.Enum()
			s.Require().Equal(tt.kind == catalog.ParamEnum, ok)

			if ok {
				s.Require().Equal(tt.want, e)
			}
		})
	}
}

// TestMarshalJSON writes a value out, and reads it back unchanged.
func (s *ParamValuePublicTestSuite) TestMarshalJSON() {
	tests := []struct {
		name string
		val  catalog.ParamValue
		want string
		err  bool
	}{
		{name: "a float", val: catalog.Float(0.5), want: "0.5"},
		// A whole-numbered float is the case that breaks: encoding/json
		// writes 1.0 as "1", and a literal with no decimal point reads back
		// as an integer. Every float below must survive as a float.
		{name: "a whole float", val: catalog.Float(1), want: "1.0"},
		{name: "a float of zero", val: catalog.Float(0), want: "0.0"},
		{name: "a negative whole float", val: catalog.Float(-2), want: "-2.0"},
		{name: "an integer", val: catalog.Int(3), want: "3"},
		{name: "a switch that is on", val: catalog.Bool(true), want: "true"},
		{name: "one that is off", val: catalog.Bool(false), want: "false"},
		{name: "an enumerated string", val: catalog.Enum("Normal"), want: `"Normal"`},
		{name: "a value with no kind", err: true},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			b, err := json.Marshal(tt.val)

			if tt.err {
				s.Require().ErrorIs(err, catalog.ErrBadParam)

				return
			}

			s.Require().NoError(err)
			s.Require().JSONEq(tt.want, string(b))

			var back catalog.ParamValue
			s.Require().NoError(json.Unmarshal(b, &back))
			s.Require().Equal(tt.val, back)
		})
	}
}

// TestUnmarshalJSON reads a value written by something else.
func (s *ParamValuePublicTestSuite) TestUnmarshalJSON() {
	tests := []struct {
		name string
		in   string
		kind catalog.ParamType
		want any
		err  bool
	}{
		{name: "a literal with no decimal point", in: "5", kind: catalog.ParamInt, want: int64(5)},
		{name: "one with a decimal point", in: "5.0", kind: catalog.ParamFloat, want: 5.0},
		{name: "one in exponent form", in: "5e2", kind: catalog.ParamFloat, want: 500.0},
		{name: "a negative integer", in: "-3", kind: catalog.ParamInt, want: int64(-3)},
		// Regression: catalog defaults are full of whole floats — an amp's
		// Master at 1.0 — and reading one back as an int made every such
		// parameter fail validation against its own declared type.
		{name: "a whole float", in: "1.0", kind: catalog.ParamFloat, want: 1.0},
		{name: "a float of zero", in: "0.0", kind: catalog.ParamFloat, want: 0.0},
		{name: "a negative whole float", in: "-1.0", kind: catalog.ParamFloat, want: -1.0},
		{name: "a large whole float", in: "100.0", kind: catalog.ParamFloat, want: 100.0},
		{name: "a switch", in: "true", kind: catalog.ParamBool, want: true},
		{name: "a string", in: `"Fast"`, kind: catalog.ParamEnum, want: "Fast"},
		{name: "nothing at all", err: true},
		{name: "an object", in: `{"a":1}`, err: true},
		{name: "an array", in: `[1,2]`, err: true},
		{name: "a null", in: `null`, err: true},
		{name: "a number that is not one", in: `1.2.3`, err: true},
		{name: "an integer too large to hold", in: `12345678901234567890123`, err: true},
		{name: "an unterminated string", in: `"abc`, err: true},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var got catalog.ParamValue

			err := got.UnmarshalJSON([]byte(tt.in))

			if tt.err {
				s.Require().Error(err)

				return
			}

			s.Require().NoError(err)
			s.Require().Equal(tt.kind, got.Type())

			switch want := tt.want.(type) {
			case float64:
				f, ok := got.Float()
				s.Require().True(ok)
				s.Require().InDelta(want, f, 1e-9)
			case int64:
				i, ok := got.Int()
				s.Require().True(ok)
				s.Require().Equal(want, i)
			case bool:
				b, ok := got.Bool()
				s.Require().True(ok)
				s.Require().Equal(want, b)
			case string:
				e, ok := got.Enum()
				s.Require().True(ok)
				s.Require().Equal(want, e)
			}
		})
	}
}

// TestString shows a value the way somebody reads it.
func (s *ParamValuePublicTestSuite) TestString() {
	tests := []struct {
		name string
		val  catalog.ParamValue
		want string
	}{
		{name: "a float", val: catalog.Float(0.5), want: "0.5"},
		{name: "a whole float", val: catalog.Float(1), want: "1.0"},
		{name: "an integer", val: catalog.Int(82), want: "82"},
		{name: "a switch", val: catalog.Bool(true), want: "true"},
		{name: "an enumerated string", val: catalog.Enum("Fast"), want: "Fast"},
		{name: "a value with no kind"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.want, tt.val.String())
		})
	}
}

func TestParamValuePublicTestSuite(t *testing.T) {
	suite.Run(t, new(ParamValuePublicTestSuite))
}
