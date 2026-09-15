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

package cli_test

import (
	"bytes"
	"os"
	"syscall"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/retr0h/tonestack/pkg/cli"
	"github.com/retr0h/tonestack/pkg/cli/mocks"
)

type InterruptPublicTestSuite struct {
	suite.Suite
}

// unnumbered is a signal with no number, as a platform that does not number
// its signals would send. Written by hand because os.Signal is the standard
// library's interface.
type unnumbered string

func (u unnumbered) String() string { return string(u) }

func (unnumbered) Signal() {}

// TestInterrupts covers what each interrupt says and does.
func (s *InterruptPublicTestSuite) TestInterrupts() {
	const (
		stopping = "\nfinishing with the pedal and letting it go, " +
			"so it is left in a safe state…\n"
		still = "\nstill finishing with the pedal; press Ctrl-C again " +
			"to quit now, at the pedal's risk\n"
		quit = "\nquitting without letting the pedal go; if it stops " +
			"answering, unplug its power and plug it back in\n"
	)

	three := []os.Signal{os.Interrupt, os.Interrupt, os.Interrupt}

	tests := []struct {
		name    string
		signals []os.Signal
		held    bool
		want    string
		stops   int
		// exit is the code the program ends with; zero means it does not end.
		exit int
	}{
		{name: "never interrupted", held: true},
		{
			name:    "once, holding the device",
			signals: three[:1],
			held:    true,
			want:    stopping,
			stops:   1,
		},
		{
			// The second stops nothing more: the command is already stopped.
			name:    "twice, holding the device",
			signals: three[:2],
			held:    true,
			want:    stopping + still,
			stops:   1,
		},
		{
			name:    "three times, holding the device",
			signals: three,
			held:    true,
			want:    stopping + still + quit,
			stops:   1,
			exit:    130,
		},
		{
			// Nothing after the third is read, because nothing after it runs:
			// the code is the third's, not the SIGTERM behind it.
			name:    "a fourth never arrives",
			signals: []os.Signal{os.Interrupt, os.Interrupt, os.Interrupt, syscall.SIGTERM},
			held:    true,
			want:    stopping + still + quit,
			stops:   1,
			exit:    130,
		},
		{
			name:    "SIGTERM ends it as SIGTERM",
			signals: []os.Signal{syscall.SIGTERM, syscall.SIGTERM, syscall.SIGTERM},
			held:    true,
			want:    stopping + still + quit,
			stops:   1,
			exit:    143,
		},
		{
			// A command that never went to the device has nothing to explain.
			name:    "three times, not holding the device",
			signals: three,
			stops:   1,
			exit:    130,
		},
		{
			name:    "a signal with no number",
			signals: []os.Signal{unnumbered("a"), unnumbered("b"), unnumbered("c")},
			held:    true,
			want:    stopping + still + quit,
			stops:   1,
			exit:    1,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctrl := gomock.NewController(s.T())

			device := mocks.NewMockHolder(ctrl)
			device.EXPECT().Held().Return(tt.held).AnyTimes()

			process := mocks.NewMockProcess(ctrl)
			process.EXPECT().Stop().Times(tt.stops)

			if tt.exit != 0 {
				process.EXPECT().Exit(tt.exit)
			}

			signals := make(chan os.Signal, len(tt.signals))
			for _, sig := range tt.signals {
				signals <- sig
			}

			close(signals)

			var out bytes.Buffer

			cli.Interrupts(signals, &out, device, process)

			s.Equal(tt.want, out.String())
		})
	}
}

func TestInterruptPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(InterruptPublicTestSuite))
}
