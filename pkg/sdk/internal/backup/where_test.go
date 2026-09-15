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

package backup

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
)

// WhereTestSuite covers where a backup goes when nobody says.
type WhereTestSuite struct {
	suite.Suite
}

// TestWhere covers the directory a backup goes into.
func (s *WhereTestSuite) TestWhere() {
	tests := []struct {
		name    string
		named   string
		state   string
		noHome  bool
		want    []string
		errText string
	}{
		{
			name:  "somewhere the caller named",
			named: filepath.Join("some", "where"),
			want:  []string{filepath.Join("some", "where")},
		},
		{
			name:  "the state directory, when there is one",
			state: filepath.Join("xdg", "state"),
			want:  []string{"xdg", "state", "tonestack", "presets"},
		},
		{
			name: "under the home directory, when there is not",
			want: []string{".local", "state", "tonestack", "presets"},
		},
		{
			name:    "nowhere to call home",
			noHome:  true,
			errText: "finding somewhere to keep a backup",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.T().Setenv("XDG_STATE_HOME", tt.state)

			if tt.noHome {
				s.T().Setenv("HOME", "")
			}

			got, err := where(tt.named)

			if tt.errText != "" {
				s.Require().ErrorContains(err, tt.errText)

				return
			}

			s.Require().NoError(err)

			for _, want := range tt.want {
				s.Require().Contains(got, want)
			}
		})
	}
}

func TestWhereTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(WhereTestSuite))
}
