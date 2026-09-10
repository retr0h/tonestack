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

package wire

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type DecodeTestSuite struct {
	suite.Suite
}

func (s *DecodeTestSuite) TestAsUintAcceptsEveryShapeADecoderProduces() {
	tests := []struct {
		name string
		in   any
		want uint64
		ok   bool
	}{
		{"uint64", uint64(7), 7, true},
		{"uint32", uint32(7), 7, true},
		{"uint16", uint16(7), 7, true},
		{"uint8", uint8(7), 7, true},
		{"int64", int64(7), 7, true},
		{"int32", int32(7), 7, true},
		{"int16", int16(7), 7, true},
		{"int8", int8(7), 7, true},
		{"int", 7, 7, true},
		{"a negative value, which is not unsigned", int8(-1), 0, false},
		{"something else entirely", "seven", 0, false},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			got, ok := AsUint(tc.in)

			s.Require().Equal(tc.ok, ok)

			if tc.ok {
				s.Require().Equal(tc.want, got)
			}
		})
	}
}

func (s *DecodeTestSuite) TestAsIntAcceptsEveryShapeADecoderProduces() {
	tests := []struct {
		name string
		in   any
		want int64
		ok   bool
	}{
		{"int64", int64(-3), -3, true},
		{"int32", int32(-3), -3, true},
		{"int16", int16(-3), -3, true},
		{"int8", int8(-3), -3, true},
		{"int", -3, -3, true},
		{"an unsigned value", uint16(3), 3, true},
		{"something else entirely", "three", 0, false},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			got, ok := AsInt(tc.in)

			s.Require().Equal(tc.ok, ok)

			if tc.ok {
				s.Require().Equal(tc.want, got)
			}
		})
	}
}

func (s *DecodeTestSuite) TestAsStringDropsTheTerminator() {
	// Line 6's strings are C strings whose declared length counts the
	// trailing NUL, so a name arrives one byte longer than it reads.
	tests := []struct {
		name string
		in   any
		want string
		ok   bool
	}{
		{"a terminated string", "Creep\x00", "Creep", true},
		{"one that is not terminated", "Creep", "Creep", true},
		{"nothing but a terminator", "\x00", "", true},
		{"something that is not a string", 7, "", false},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			got, ok := AsString(tc.in)

			s.Require().Equal(tc.ok, ok)
			s.Require().Equal(tc.want, got)
		})
	}
}

func TestDecodeTestSuite(t *testing.T) {
	suite.Run(t, new(DecodeTestSuite))
}
