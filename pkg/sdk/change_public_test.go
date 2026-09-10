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
)

// ChangePublicTestSuite covers what a caller is handed for a write.
type ChangePublicTestSuite struct {
	suite.Suite
}

// TestOnDevice covers telling a write to hardware from a write to a file.
func (s *ChangePublicTestSuite) TestOnDevice() {
	tests := []struct {
		name string
		in   sdk.Change
		want bool
	}{
		{
			// A device is written in place. There is no file, which is what
			// makes the kept backup the only way back.
			name: "a slot on an attached device",
			in:   sdk.Change{Action: sdk.Copied, To: sdk.At{Slot: 3}},
			want: true,
		},
		{
			name: "a setlist written to a new file",
			in: sdk.Change{
				Action: sdk.Copied,
				To:     sdk.At{Slot: 3},
				Path:   "/tmp/out.hls",
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.want, tt.in.OnDevice())
		})
	}
}

func TestChangePublicTestSuite(t *testing.T) {
	suite.Run(t, new(ChangePublicTestSuite))
}
