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

package catalog

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Float returns a ParamValue holding v.
func Float(v float64) ParamValue { return ParamValue{typ: ParamFloat, f: v} }

// Int returns a ParamValue holding v.
func Int(v int64) ParamValue { return ParamValue{typ: ParamInt, i: v} }

// Bool returns a ParamValue holding v.
func Bool(v bool) ParamValue { return ParamValue{typ: ParamBool, b: v} }

// Enum returns a ParamValue holding the enumerated member v.
func Enum(v string) ParamValue { return ParamValue{typ: ParamEnum, s: v} }

// Type reports the kind held, or the empty ParamType for a zero value.
func (v ParamValue) Type() ParamType { return v.typ }

// Float returns the float held and whether the value holds one.
func (v ParamValue) Float() (value float64, ok bool) {
	return v.f, v.typ == ParamFloat
}

// Int returns the integer held and whether the value holds one.
func (v ParamValue) Int() (value int64, ok bool) {
	return v.i, v.typ == ParamInt
}

// Bool returns the boolean held and whether the value holds one.
func (v ParamValue) Bool() (value bool, ok bool) {
	return v.b, v.typ == ParamBool
}

// Enum returns the enumerated member held and whether the value holds one.
func (v ParamValue) Enum() (value string, ok bool) {
	return v.s, v.typ == ParamEnum
}

// MarshalJSON writes the value in its own kind. A zero ParamValue is an error.
// String renders the value the way the device would show it.
//
// A ParamValue holds one of four kinds, and a caller that only wants to print
// it should not have to ask which. Marshalling is the same rendering, so it
// is reused rather than duplicated.
func (v ParamValue) String() string {
	raw, err := v.MarshalJSON()
	if err != nil {
		return ""
	}

	return strings.Trim(string(raw), `"`)
}

func (v ParamValue) MarshalJSON() ([]byte, error) {
	switch v.typ {
	case ParamFloat:
		// Always emit a decimal point. encoding/json writes 1.0 as "1", and
		// UnmarshalJSON reads a literal without a point as an integer — so a
		// whole-numbered float would not survive its own round trip.
		out := strconv.FormatFloat(v.f, 'f', -1, 64)
		if !strings.ContainsAny(out, ".eE") {
			out += ".0"
		}

		return []byte(out), nil
	case ParamInt:
		return json.Marshal(v.i)
	case ParamBool:
		return json.Marshal(v.b)
	case ParamEnum:
		return json.Marshal(v.s)
	default:
		return nil, fmt.Errorf(
			"%w: zero ParamValue has no kind",
			ErrBadParam,
		)
	}
}

// UnmarshalJSON infers the kind from the JSON literal. A number whose literal
// carries no '.', 'e' or 'E' is an integer; any other number is a float. JSON
// alone cannot distinguish 5 from 5.0 once decoded, so the literal decides.
func (v *ParamValue) UnmarshalJSON(b []byte) error {
	s := strings.TrimSpace(string(b))
	if s == "" {
		return fmt.Errorf("%w: empty value", ErrBadParam)
	}

	switch {
	case s == "true", s == "false":
		v.typ, v.b = ParamBool, s == "true"

		return nil
	case s[0] == '"':
		var str string
		if err := json.Unmarshal(b, &str); err != nil {
			return fmt.Errorf("%w: %w", ErrBadParam, err)
		}

		v.typ, v.s = ParamEnum, str

		return nil
	case s[0] == '-' || (s[0] >= '0' && s[0] <= '9'):
		return v.unmarshalNumber(b, s)
	default:
		return fmt.Errorf(
			"%w: %q is not a parameter value",
			ErrBadParam,
			s,
		)
	}
}

func (v *ParamValue) unmarshalNumber(b []byte, lit string) error {
	if !strings.ContainsAny(lit, ".eE") {
		var i int64
		if err := json.Unmarshal(b, &i); err != nil {
			return fmt.Errorf("%w: %w", ErrBadParam, err)
		}

		v.typ, v.i = ParamInt, i

		return nil
	}

	var f float64
	if err := json.Unmarshal(b, &f); err != nil {
		return fmt.Errorf("%w: %w", ErrBadParam, err)
	}

	v.typ, v.f = ParamFloat, f

	return nil
}
