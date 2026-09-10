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

// Package slot addresses a position in a setlist the way the hardware does.
//
// A device labels its slots 01A through 42C, and that is what a player reads
// off the pedal. Everything underneath counts from zero. Both are the same
// position and neither is going away, so this is the one place that converts.
package slot

import (
	"fmt"
	"strconv"
	"strings"
)

// perBank is how many presets share a bank letter.
const perBank = 3

// Label renders a position the way the hardware labels it — 01A through 42C.
func Label(slot int) string {
	return fmt.Sprintf("%02d%c", slot/perBank+1, rune('A'+slot%perBank))
}

// parse reads either form: a label the pedal shows, or a bare index.
//
// A label is what somebody has in front of them, and it is what this project
// prints, so refusing it would mean printing addresses nothing accepts. A bare
// number stays valid because scripts count.
func parse(s string) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("%w: no slot given", ErrBadSlot)
	}

	// A bare number is an index, counted from zero, and needs no decoding.
	if n, err := strconv.Atoi(s); err == nil {
		if n < 0 {
			return 0, fmt.Errorf("%w: %q is before the first slot", ErrBadSlot, s)
		}

		return n, nil
	}

	bank, letter := s[:len(s)-1], strings.ToUpper(s[len(s)-1:])

	// Unsigned, so a letter before A wraps rather than going negative and is
	// caught by the same test.
	offset := int(letter[0] - 'A')
	if offset >= perBank {
		return 0, fmt.Errorf(
			"%w: %q — a bank runs A to %c", ErrBadSlot, s, 'A'+perBank-1)
	}

	n, err := strconv.Atoi(bank)
	if err != nil || n < 1 {
		return 0, fmt.Errorf(
			"%w: %q — expected a label like 31A, or a number from zero",
			ErrBadSlot, s)
	}

	return (n-1)*perBank + offset, nil
}
