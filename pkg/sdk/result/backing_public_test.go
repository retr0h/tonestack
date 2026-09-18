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

	"github.com/retr0h/tonestack/pkg/sdk/result"
)

// BackingPublicTestSuite covers what a caller is handed for a rig's records.
type BackingPublicTestSuite struct {
	suite.Suite
}

// TestStated covers telling a rig that can be checked from one that cannot.
//
// A rig with no years is not a rig whose records are fine. It is a rig
// nothing can hold its records to, and the two read differently.
func (s *BackingPublicTestSuite) TestStated() {
	tests := []struct {
		name string
		in   result.Backing
		want bool
	}{
		{
			name: "a rig that says which years it describes",
			in:   result.Backing{From: 1975, To: 1978},
			want: true,
		},
		{
			name: "a rig that applied for one year",
			in:   result.Backing{From: 2004, To: 2004},
			want: true,
		},
		{
			name: "a rig that says none",
		},
		{
			// Half an era is not an era: a rig that says when it started and
			// not when it stopped cannot refuse a record from 2019.
			name: "a rig that states one end",
			in:   result.Backing{From: 1975},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.want, tt.in.Stated())
		})
	}
}

// TestOutside counts the records made when the rig was not what it says.
func (s *BackingPublicTestSuite) TestOutside() {
	tests := []struct {
		name string
		in   result.Backing
		want int
	}{
		{
			name: "every record in era",
			in: result.Backing{Records: []result.Record{
				{Track: "anthem", Year: 1975},
				{Track: "la-villa-strangiato", Year: 1978},
			}},
		},
		{
			// The case worth acting on: some of the evidence describes gear
			// the rig does not name, and the rest of it is fine.
			name: "one of three",
			in: result.Backing{Records: []result.Record{
				{Track: "bombtrack", Year: 1992},
				{Track: "take-the-power-back", Year: 1992},
				{Track: "sleep-now-in-the-fire", Year: 1999, Outside: true},
			}},
			want: 1,
		},
		{
			name: "all of them",
			in: result.Backing{Records: []result.Record{
				{Track: "aeroplane", Year: 1995, Outside: true},
				{Track: "suck-my-kiss", Year: 1991, Outside: true},
			}},
			want: 2,
		},
		{
			name: "a rig nobody has measured",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.want, tt.in.Outside())
		})
	}
}

func TestBackingPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(BackingPublicTestSuite))
}
