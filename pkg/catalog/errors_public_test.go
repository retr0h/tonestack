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

package catalog_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/catalog"
)

type ErrorsPublicTestSuite struct {
	suite.Suite
}

func (s *ErrorsPublicTestSuite) TestBadParamErrorReportsBlockKeyAndReason() {
	err := &catalog.BadParamError{Model: "HD2_AmpX", Key: "Gain", Reason: "out of range"}

	s.Require().Contains(err.Error(), "HD2_AmpX")
	s.Require().Contains(err.Error(), "Gain")
	s.Require().Contains(err.Error(), "out of range")
}

func (s *ErrorsPublicTestSuite) TestBadParamErrorUnwrapsToSentinel() {
	err := &catalog.BadParamError{Model: "HD2_AmpX", Key: "Gain", Reason: "nope"}

	s.Require().ErrorIs(err, catalog.ErrBadParam)
}

func (s *ErrorsPublicTestSuite) TestBadParamErrorSurvivesWrapping() {
	err := fmt.Errorf("validating: %w",
		&catalog.BadParamError{Model: "HD2_AmpX", Key: "Gain", Reason: "nope"})

	s.Require().ErrorIs(err, catalog.ErrBadParam)

	var target *catalog.BadParamError
	s.Require().True(errors.As(err, &target))
	s.Require().Equal("Gain", target.Key)
}

func TestErrorsPublicTestSuite(t *testing.T) {
	suite.Run(t, new(ErrorsPublicTestSuite))
}
