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

package recipes_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/internal/recipes"
)

// StorePublicTestSuite covers finding a rig as something a caller can stand
// in for.
//
// Find is the package-level function of the same name, so what is asserted
// here is that reaching it through the value agrees with calling it directly.
type StorePublicTestSuite struct {
	suite.Suite
}

// TestFind covers looking up a rig through the value.
func (s *StorePublicTestSuite) TestFind() {
	tests := []struct {
		name string
		dir  string
		id   string
		ok   bool
	}{
		{name: "one the directory holds", dir: "testdata-good", id: "mike-dirnt", ok: true},
		{name: "one nothing holds", dir: "testdata-good", id: "nobody"},
		{name: "a directory that is not there", dir: "nowhere", id: "mike-dirnt"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := recipes.Store{}.Find(tt.dir, tt.id)

			if !tt.ok {
				s.Require().Error(err)

				return
			}

			want, err := recipes.Find(tt.dir, tt.id)
			s.Require().NoError(err)
			s.Require().Equal(want, got)
		})
	}
}

func TestStorePublicTestSuite(t *testing.T) {
	suite.Run(t, new(StorePublicTestSuite))
}
