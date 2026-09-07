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

func (s *ParamValuePublicTestSuite) TestConstructorsSetType() {
	tests := []struct {
		name string
		got  catalog.ParamValue
		want catalog.ParamType
	}{
		{"float", catalog.Float(0.5), catalog.ParamFloat},
		{"int", catalog.Int(3), catalog.ParamInt},
		{"bool", catalog.Bool(true), catalog.ParamBool},
		{"enum", catalog.Enum("Normal"), catalog.ParamEnum},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			s.Require().Equal(tc.want, tc.got.Type())
		})
	}
}

func (s *ParamValuePublicTestSuite) TestAccessorsReturnValueAndTrueOnMatch() {
	f, ok := catalog.Float(0.25).Float()
	s.Require().True(ok)
	s.Require().InDelta(0.25, f, 1e-9)

	i, ok := catalog.Int(7).Int()
	s.Require().True(ok)
	s.Require().Equal(int64(7), i)

	b, ok := catalog.Bool(true).Bool()
	s.Require().True(ok)
	s.Require().True(b)

	e, ok := catalog.Enum("Bright").Enum()
	s.Require().True(ok)
	s.Require().Equal("Bright", e)
}

func (s *ParamValuePublicTestSuite) TestAccessorsReturnFalseOnMismatch() {
	_, ok := catalog.Enum("Bright").Float()
	s.Require().False(ok)

	_, ok = catalog.Float(1).Int()
	s.Require().False(ok)

	_, ok = catalog.Int(1).Bool()
	s.Require().False(ok)

	_, ok = catalog.Bool(true).Enum()
	s.Require().False(ok)
}

func (s *ParamValuePublicTestSuite) TestMarshalRoundTripsEachType() {
	tests := []struct {
		name string
		val  catalog.ParamValue
		json string
	}{
		{"float", catalog.Float(0.5), "0.5"},
		// A whole-numbered float is the case that breaks: encoding/json
		// writes 1.0 as "1", and a literal with no decimal point reads back
		// as an integer. Every float below must survive as a float.
		{"float 1.0", catalog.Float(1.0), "1.0"},
		{"float 0.0", catalog.Float(0.0), "0.0"},
		{"float -2.0", catalog.Float(-2.0), "-2.0"},
		{"int", catalog.Int(3), "3"},
		{"bool true", catalog.Bool(true), "true"},
		{"bool false", catalog.Bool(false), "false"},
		{"enum", catalog.Enum("Normal"), `"Normal"`},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			b, err := json.Marshal(tc.val)
			s.Require().NoError(err)
			s.Require().JSONEq(tc.json, string(b))

			var back catalog.ParamValue
			s.Require().NoError(json.Unmarshal(b, &back))
			s.Require().Equal(tc.val, back)
		})
	}
}

func (s *ParamValuePublicTestSuite) TestUnmarshalDistinguishesIntFromFloat() {
	var i catalog.ParamValue
	s.Require().NoError(json.Unmarshal([]byte("5"), &i))
	s.Require().Equal(catalog.ParamInt, i.Type())

	var f catalog.ParamValue
	s.Require().NoError(json.Unmarshal([]byte("5.0"), &f))
	s.Require().Equal(catalog.ParamFloat, f.Type())

	var e catalog.ParamValue
	s.Require().NoError(json.Unmarshal([]byte("5e2"), &e))
	s.Require().Equal(catalog.ParamFloat, e.Type())
}

func (s *ParamValuePublicTestSuite) TestUnmarshalAcceptsNegativeNumbers() {
	var i catalog.ParamValue
	s.Require().NoError(json.Unmarshal([]byte("-3"), &i))
	s.Require().Equal(catalog.ParamInt, i.Type())

	got, ok := i.Int()
	s.Require().True(ok)
	s.Require().Equal(int64(-3), got)
}

func (s *ParamValuePublicTestSuite) TestWholeNumberedFloatsStayFloats() {
	// Regression: catalog defaults are full of whole floats — an amp's Master
	// at 1.0 — and reading one back as an int made every such parameter fail
	// validation against its own declared type.
	for _, v := range []float64{0, 1, -1, 2, 100} {
		var back catalog.ParamValue

		raw, err := json.Marshal(catalog.Float(v))
		s.Require().NoError(err)
		s.Require().NoError(json.Unmarshal(raw, &back))

		s.Require().Equal(catalog.ParamFloat, back.Type(),
			"%v marshalled as %s and came back the wrong kind", v, raw)

		got, ok := back.Float()
		s.Require().True(ok)
		s.Require().InDelta(v, got, 1e-9)
	}
}

func (s *ParamValuePublicTestSuite) TestMarshalZeroValueIsAnError() {
	var zero catalog.ParamValue

	_, err := json.Marshal(zero)
	s.Require().ErrorIs(err, catalog.ErrBadParam)
}

func (s *ParamValuePublicTestSuite) TestUnmarshalRejectsMalformedInput() {
	tests := []struct {
		name string
		in   string
	}{
		{"empty", ``},
		{"object", `{"a":1}`},
		{"array", `[1,2]`},
		{"null", `null`},
		{"bad float", `1.2.3`},
		{"bad int", `12345678901234567890123`},
		{"unterminated string", `"abc`},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			var v catalog.ParamValue
			s.Require().Error(v.UnmarshalJSON([]byte(tc.in)))
		})
	}
}

func (s *ParamValuePublicTestSuite) TestString() {
	tests := []struct {
		name string
		val  catalog.ParamValue
		want string
	}{
		{"a float", catalog.Float(0.5), "0.5"},
		{"a whole float", catalog.Float(1), "1.0"},
		{"an integer", catalog.Int(82), "82"},
		{"a boolean", catalog.Bool(true), "true"},
		{"an enumerated string", catalog.Enum("Fast"), "Fast"},
		{"a value with no kind", catalog.ParamValue{}, ""},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			s.Require().Equal(tc.want, tc.val.String())
		})
	}
}

func TestParamValuePublicTestSuite(t *testing.T) {
	suite.Run(t, new(ParamValuePublicTestSuite))
}
