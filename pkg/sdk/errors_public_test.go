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
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk"
)

type ErrorsPublicTestSuite struct {
	suite.Suite
}

func (s *ErrorsPublicTestSuite) TestUnknownModelErrorNamesTheProduct() {
	err := &sdk.UnknownModelError{Product: 0xBEEF}

	s.Require().Contains(err.Error(), "0xbeef")
	s.Require().ErrorIs(err, sdk.ErrUnknownModel)
}

func (s *ErrorsPublicTestSuite) TestUnknownModelErrorSurvivesWrapping() {
	err := fmt.Errorf("opening: %w", &sdk.UnknownModelError{Product: 0xBEEF})

	var target *sdk.UnknownModelError
	s.Require().True(errors.As(err, &target))
	s.Require().Equal(uint16(0xBEEF), target.Product)
}

func (s *ErrorsPublicTestSuite) TestSentinelsAreDistinct() {
	s.Require().NotErrorIs(sdk.ErrNoDevice, sdk.ErrUnknownModel)
}

// TestNotAPresetError covers a device answering with something nobody can
// decode, which is how a protocol change becomes visible.
func (s *ErrorsPublicTestSuite) TestNotAPresetError() {
	tests := []struct {
		name   string
		result any
		want   string
	}{
		{
			name:   "a decoded document",
			result: map[any]any{1: "a", 2: "b"},
			want:   "map with 2 keys",
		},
		{
			name:   "a run of bytes",
			result: []byte{1, 2, 3},
			want:   "3 bytes",
		},
		{
			name:   "something else entirely",
			result: 42,
			want:   "int",
		},
		{name: "nothing recognisable at all", want: "<nil>"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			err := &sdk.NotAPresetError{Result: tt.result}

			s.Require().Equal(tt.want, err.Shape())
			s.Require().Contains(err.Error(), tt.want)
			s.Require().ErrorIs(err, sdk.ErrNotAPreset)
		})
	}
}

func TestErrorsPublicTestSuite(t *testing.T) {
	suite.Run(t, new(ErrorsPublicTestSuite))
}
