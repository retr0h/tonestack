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

package sdk_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk"
	"github.com/retr0h/tonestack/pkg/sdk/preset"
)

// ReadingPublicTestSuite covers what a caller is handed for one preset.
type ReadingPublicTestSuite struct {
	suite.Suite
}

// TestEmpty covers what says a slot holds no preset.
func (s *ReadingPublicTestSuite) TestEmpty() {
	tests := []struct {
		name string
		in   sdk.Reading
		want bool
	}{
		{
			// A device names every slot, so an untouched one still answers
			// with whatever it shipped with. Only the document says.
			name: "a slot the device named and nobody filled",
			in:   sdk.Reading{Name: "New Preset"},
			want: true,
		},
		{
			name: "a slot holding a preset",
			in:   sdk.Reading{Name: "Mike Dirnt", Doc: &preset.Document{}},
		},
		{
			// A reply nobody could decode is not a slot holding nothing. It
			// is a slot whose contents did not survive the wire, and calling
			// it empty would lose that.
			name: "a reply that was not a preset",
			in: sdk.Reading{
				Name:   "Mike Dirnt",
				Doc:    &preset.Document{},
				Answer: &sdk.Answer{Model: "HX Stomp", Shape: "map with 2 keys"},
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.want, tt.in.Empty())
		})
	}
}

func TestReadingPublicTestSuite(t *testing.T) {
	suite.Run(t, new(ReadingPublicTestSuite))
}
