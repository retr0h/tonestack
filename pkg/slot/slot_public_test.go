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

package slot_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/slot"
)

type SlotPublicTestSuite struct {
	suite.Suite
}

func (s *SlotPublicTestSuite) TestLabelsAPosition() {
	for _, tc := range []struct {
		slot int
		want string
	}{
		{0, "01A"},
		{1, "01B"},
		{2, "01C"},
		{3, "02A"},
		{90, "31A"},
		{125, "42C"},
	} {
		s.Require().Equal(tc.want, slot.Label(tc.slot))
	}
}

func (s *SlotPublicTestSuite) TestReadsALabelThePedalShows() {
	for _, tc := range []struct {
		in   string
		want int
	}{
		{"01A", 0},
		{"01C", 2},
		{"31A", 90},
		{"31a", 90 /* whatever case somebody types */},
		{"  02B  ", 4},
		{"42C", 125},
	} {
		got, err := slot.Parse(tc.in)

		s.Require().NoError(err)
		s.Require().Equal(tc.want, got, tc.in)
	}
}

func (s *SlotPublicTestSuite) TestReadsABareIndex() {
	// Scripts count, and a number is what they have. It is an index rather
	// than a bank, which is why it needs no letter.
	got, err := slot.Parse("90")

	s.Require().NoError(err)
	s.Require().Equal(90, got)
}

func (s *SlotPublicTestSuite) TestALabelAndItsIndexAgree() {
	for i := range 126 {
		got, err := slot.Parse(slot.Label(i))

		s.Require().NoError(err)
		s.Require().Equal(i, got, "%s must address slot %d", slot.Label(i), i)
	}
}

func (s *SlotPublicTestSuite) TestRefusesWhatNamesNoSlot() {
	for _, in := range []string{"", "   ", "-1", "01D", "00A", "A", "xxA", "0A"} {
		_, err := slot.Parse(in)

		s.Require().ErrorIs(err, slot.ErrBadSlot, "%q", in)
	}
}

func (s *SlotPublicTestSuite) TestTheFlagTakesEitherForm() {
	var got int

	v := slot.NewValue(&got)

	s.Require().NoError(v.Set("31A"))
	s.Require().Equal(90, got)
	s.Require().Equal("31A", v.String(), "shown the way the pedal shows it")

	s.Require().NoError(v.Set("7"))
	s.Require().Equal(7, got)

	s.Require().Error(v.Set("nowhere"))
	s.Require().Equal("slot", v.Type())
}

func (s *SlotPublicTestSuite) TestTheFlagWithNowhereToWrite() {
	s.Require().Empty((&slot.Value{}).String())
}

func TestSlotPublicTestSuite(t *testing.T) {
	suite.Run(t, new(SlotPublicTestSuite))
}
