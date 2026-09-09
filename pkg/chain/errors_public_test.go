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

package chain_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/chain"
)

type ErrorsPublicTestSuite struct {
	suite.Suite
}

// TestUnknownBlockError names the model nothing carries.
func (s *ErrorsPublicTestSuite) TestUnknownBlockError() {
	tests := []struct {
		name string
		err  error
	}{
		{name: "on its own", err: &chain.UnknownBlockError{Model: "HD2_Nope"}},
		{
			// The model has to survive the wrapping every layer adds, or a
			// caller cannot say which block it was.
			name: "wrapped by a caller",
			err: fmt.Errorf("resolving: %w",
				&chain.UnknownBlockError{Model: "HD2_Nope"}),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().ErrorIs(tt.err, chain.ErrUnknownBlock)
			s.Require().Contains(tt.err.Error(), "HD2_Nope")

			var target *chain.UnknownBlockError
			s.Require().True(errors.As(tt.err, &target))
			s.Require().Equal("HD2_Nope", string(target.Model))
		})
	}
}

func (s *ErrorsPublicTestSuite) TestOverBudgetErrorNamesChipAndCost() {
	err := &chain.OverBudgetError{Chip: 1, Cost: 1.2, Ceiling: 0.95}

	s.Require().Contains(err.Error(), "chip 1")
	s.Require().ErrorIs(err, chain.ErrOverBudget)
}

func (s *ErrorsPublicTestSuite) TestTopologyErrorCarriesReason() {
	err := &chain.TopologyError{Reason: "too many blocks"}

	s.Require().Contains(err.Error(), "too many blocks")
	s.Require().ErrorIs(err, chain.ErrBadTopology)
}

func (s *ErrorsPublicTestSuite) TestSentinelsAreDistinct() {
	all := []error{chain.ErrUnknownBlock, chain.ErrOverBudget, chain.ErrBadTopology}

	for i, a := range all {
		for j, b := range all {
			if i != j {
				s.Require().NotErrorIs(a, b)
			}
		}
	}
}

func TestErrorsPublicTestSuite(t *testing.T) {
	suite.Run(t, new(ErrorsPublicTestSuite))
}
