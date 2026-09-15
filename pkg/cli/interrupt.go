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

package cli

import (
	"fmt"
	"io"
	"os"
	"syscall"
)

// What an interrupt says while a command holds the device.
//
// Each starts on a line of its own, because a terminal echoes ^C without
// ending the line it was on.
const (
	interruptStopping = "finishing with the pedal and letting it go, " +
		"so it is left in a safe state…"
	interruptStill = "still finishing with the pedal; press Ctrl-C again " +
		"to quit now, at the pedal's risk"
	interruptQuit = "quitting without letting the pedal go; if it stops " +
		"answering, unplug its power and plug it back in"
)

// Interrupts answers the interrupts a running command receives, and returns
// once signals is closed.
//
// The first stops the command. A device command does not end there: a write
// that has started finishes, because half a message stalls the pedal, and the
// session then tells the device it is over, which waits on the device. So a
// command holding the device says what it is waiting for, and says it again on
// the second interrupt. The third ends the program where it stands, which is
// the pedal's risk rather than this program's, and so it is the only one that
// does.
//
// A command that is not holding the device ends quickly once stopped, and has
// nothing to explain, so nothing is printed for it.
func Interrupts(
	signals <-chan os.Signal,
	w io.Writer,
	device Holder,
	process Process,
) {
	count := 0

	for sig := range signals {
		count++

		switch count {
		case 1:
			say(w, device, interruptStopping)
			process.Stop()
		case 2:
			say(w, device, interruptStill)
		default:
			say(w, device, interruptQuit)
			process.Exit(exitCode(sig))

			return
		}
	}
}

// say writes line to w on a line of its own, if the device is held.
func say(
	w io.Writer,
	device Holder,
	line string,
) {
	if !device.Held() {
		return
	}

	// Nothing useful is left to do with an error writing to stderr.
	_, _ = fmt.Fprintf(w, "\n%s\n", line)
}

// exitCode is what a shell reports for a program a signal ended: 128 plus the
// signal's number, so 130 for Ctrl-C and 143 for SIGTERM. A signal with no
// number is a plain failure.
func exitCode(
	sig os.Signal,
) int {
	if n, ok := sig.(syscall.Signal); ok {
		return 128 + int(n)
	}

	return 1
}
