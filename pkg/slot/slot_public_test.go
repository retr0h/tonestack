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

// TestLabel names a position the way the pedal prints it.
func (s *SlotPublicTestSuite) TestLabel() {
	tests := []struct {
		name string
		slot int
		want string
	}{
		{name: "the first", slot: 0, want: "01A"},
		{name: "the second in a bank", slot: 1, want: "01B"},
		{name: "the third", slot: 2, want: "01C"},
		{name: "the first of the next bank", slot: 3, want: "02A"},
		{name: "one well into the setlist", slot: 90, want: "31A"},
		{name: "the last", slot: 125, want: "42C"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.want, slot.Label(tt.slot))
		})
	}
}

// TestParse reads whatever somebody types.
func (s *SlotPublicTestSuite) TestParse() {
	tests := []struct {
		name string
		in   string
		want int
		err  bool
	}{
		{name: "a label the pedal shows", in: "01A", want: 0},
		{name: "the third in a bank", in: "01C", want: 2},
		{name: "one further in", in: "31A", want: 90},
		{name: "whatever case somebody types", in: "31a", want: 90},
		{name: "one somebody pasted with spaces", in: "  02B  ", want: 4},
		{name: "the last", in: "42C", want: 125},
		{
			// Scripts count, and a number is what they have. It is an index
			// rather than a bank, which is why it needs no letter.
			name: "a bare index",
			in:   "90",
			want: 90,
		},
		{name: "nothing at all", in: "", err: true},
		{name: "only spaces", in: "   ", err: true},
		{name: "a negative index", in: "-1", err: true},
		{name: "a letter no bank has", in: "01D", err: true},
		{name: "a bank below the first", in: "00A", err: true},
		{name: "a letter with no bank", in: "A", err: true},
		{name: "a bank that is not a number", in: "xxA", err: true},
		{name: "a bank of one digit", in: "0A", err: true},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := slot.Parse(tt.in)

			if tt.err {
				s.Require().ErrorIs(err, slot.ErrBadSlot)

				return
			}

			s.Require().NoError(err)
			s.Require().Equal(tt.want, got)
		})
	}
}

// TestALabelAndItsIndexAgree is a property of the pair rather than a case of
// either. Every slot a device has, both ways round.
func (s *SlotPublicTestSuite) TestALabelAndItsIndexAgree() {
	for i := range 126 {
		got, err := slot.Parse(slot.Label(i))

		s.Require().NoError(err)
		s.Require().Equal(i, got, "%s must address slot %d", slot.Label(i), i)
	}
}

// TestValue covers the flag, which takes either form.
func (s *SlotPublicTestSuite) TestValue() {
	tests := []struct {
		name  string
		set   string
		want  int
		shown string
		err   bool
	}{
		{
			name:  "a label",
			set:   "31A",
			want:  90,
			shown: "31A",
		},
		{
			name:  "an index, shown the way the pedal shows it",
			set:   "7",
			want:  7,
			shown: "03B",
		},
		{
			name: "something that names no slot",
			set:  "nowhere",
			err:  true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var got int

			v := slot.NewValue(&got)

			if tt.err {
				s.Require().Error(v.Set(tt.set))

				return
			}

			s.Require().NoError(v.Set(tt.set))
			s.Require().Equal(tt.want, got)
			s.Require().Equal(tt.shown, v.String())
			s.Require().Equal("slot", v.Type())
		})
	}
}

// TestValueWithNowhereToWrite covers a flag nobody wired up.
func (s *SlotPublicTestSuite) TestValueWithNowhereToWrite() {
	s.Require().Empty((&slot.Value{}).String())
}

func TestSlotPublicTestSuite(t *testing.T) {
	suite.Run(t, new(SlotPublicTestSuite))
}
