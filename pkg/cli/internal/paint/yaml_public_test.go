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

package paint_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/cli/internal/paint"
)

// YAMLPublicTestSuite covers painting a document without changing it.
type YAMLPublicTestSuite struct {
	suite.Suite
}

// TestYAML paints a document for a sink that takes no colour, which is what a
// buffer and a redirected file both are. The whole point: a rig redirected to
// a file is still the rig, byte for byte. Painting happens only where a
// terminal can show it.
func (s *YAMLPublicTestSuite) TestYAML() {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "a document",
			body: strings.Join([]string{
				"# a rig",
				"schema: RigSpec",
				"chain:",
				"- gear: Ampeg SVT",
				"  role: amp",
				"",
				"target:",
				"  device: HX Stomp",
			}, "\n"),
		},
		{name: "a line that is not a field", body: "just words"},
		{name: "a colour a switch can show", body: "  led: violet"},
		{
			// `auto` is not a colour — the light follows the block.
			name: "a switch following its block",
			body: "  led: auto color",
		},
		{
			// A name this does not know gets no colour rather than a wrong
			// one.
			name: "a colour nobody has",
			body: "  led: chartreuse",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.body, paint.YAML(&bytes.Buffer{}, tt.body))
		})
	}
}

// TestSwatch raises the hue of what it is given, where it can.
func (s *YAMLPublicTestSuite) TestSwatch() {
	tests := []struct {
		name string
		rgb  int
	}{
		{name: "a colour below the range", rgb: -1},
		{name: "a colour above it", rgb: 0x1000000},
		// A switch that lights nothing has no hue to raise.
		{name: "black", rgb: 0},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().Equal("x", paint.Swatch(&bytes.Buffer{}, tt.rgb, "x"))
		})
	}
}

func TestYAMLPublicTestSuite(t *testing.T) {
	suite.Run(t, new(YAMLPublicTestSuite))
}
