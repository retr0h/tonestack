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

package result_test

import (
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/result"
)

type FormatPublicTestSuite struct {
	suite.Suite
}

// TestSet covers taking a format by name, the way a flag hands one over.
func (s *FormatPublicTestSuite) TestSet() {
	tests := []struct {
		name string
		in   string
		want result.Format
		err  bool
	}{
		{name: "a rig", in: "rigspec", want: result.FormatRig},
		{name: "the device's own file", in: "hlx", want: result.FormatPreset},
		{
			// Refused when the flag is parsed, rather than quietly written
			// as a rig once the device is already open.
			name: "a format that does not exist",
			in:   "hlxx",
			want: result.FormatRig,
			err:  true,
		},
		{name: "no name at all", in: "", want: result.FormatRig, err: true},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got := result.FormatRig

			err := got.Set(tt.in)

			if tt.err {
				s.Require().ErrorIs(err, result.ErrUnknownFormat)
				s.Require().ErrorContains(err, "use rigspec or hlx")
			} else {
				s.Require().NoError(err)
			}

			s.Require().Equal(tt.want, got, "a refused name leaves the format alone")
			s.Require().Equal(string(tt.want), got.String())
		})
	}
}

// TestFlag covers the format as a command-line flag.
func (s *FormatPublicTestSuite) TestFlag() {
	tests := []struct {
		name string
		args []string
		want result.Format
		err  bool
	}{
		{name: "left alone", want: result.FormatRig},
		{name: "asked for", args: []string{"--as", "hlx"}, want: result.FormatPreset},
		{name: "misspelled", args: []string{"--as", "yaml"}, want: result.FormatRig, err: true},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			as := result.FormatRig

			flags := pflag.NewFlagSet("export", pflag.ContinueOnError)
			flags.Var(&as, "as", "rigspec for a rig, hlx for the device's own file")

			err := flags.Parse(tt.args)

			if tt.err {
				s.Require().ErrorIs(err, result.ErrUnknownFormat)
			} else {
				s.Require().NoError(err)
			}

			s.Require().Equal(tt.want, as)

			// Typed as a string with the default shown, which is exactly what
			// the usage said when the flag was a plain string.
			s.Require().Contains(flags.FlagUsages(), `--as string`)
			s.Require().Contains(flags.FlagUsages(), `(default "rigspec")`)
		})
	}
}

func TestFormatPublicTestSuite(t *testing.T) {
	suite.Run(t, new(FormatPublicTestSuite))
}
