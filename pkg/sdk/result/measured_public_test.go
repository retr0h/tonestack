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

package result_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/corpus"
	"github.com/retr0h/tonestack/pkg/sdk/result"
)

// MeasuredPublicTestSuite covers what a caller is handed for the corpus.
type MeasuredPublicTestSuite struct {
	suite.Suite
}

// TestAboutOne covers telling the two questions apart.
//
// Both come out of the same measurements, and which was asked decides what
// there is to say about them, so the answer has to carry it rather than leave
// somebody to infer it from which fields are set.
func (s *MeasuredPublicTestSuite) TestAboutOne() {
	tests := []struct {
		name string
		in   result.Measured
		want bool
	}{
		{
			name: "one model was asked about",
			in: result.Measured{
				Stats: &corpus.Stats{},
				Model: "HD2_AmpSVBeastNrm",
			},
			want: true,
		},
		{
			// The grammar of a chain, which is about no model in particular.
			name: "the grammar was asked for",
			in:   result.Measured{Stats: &corpus.Stats{}, Instrument: "bass"},
		},
		{
			name: "the grammar of everything",
			in:   result.Measured{Stats: &corpus.Stats{}},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.want, tt.in.AboutOne())
		})
	}
}

func TestMeasuredPublicTestSuite(t *testing.T) {
	suite.Run(t, new(MeasuredPublicTestSuite))
}
