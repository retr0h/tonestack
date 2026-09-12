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
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/cli"
)

type HelpPublicTestSuite struct {
	suite.Suite
}

// TestRender lays out a help page.
func (s *HelpPublicTestSuite) TestRender() {
	tests := []struct {
		name     string
		help     cli.Help
		contains []string
		absent   []string
	}{
		{
			name: "every section there is",
			help: cli.Help{
				Name:        "tonestack presets make",
				Description: "Build a preset.\n\nFrom a recipe.",
				Usage:       "tonestack presets make [flags]",
				Commands:    []cli.Item{{Name: "list", Description: "list them"}},
				Flags:       []cli.Item{{Name: "--id string", Description: "which one"}},
				Footer:      "Run --help for more.",
			},
			contains: []string{
				"tonestack presets make", "Build a preset.", "USAGE",
				"COMMANDS", "list", "FLAGS", "--id string", "Run --help for more.",
			},
		},
		{
			name:     "a command with neither subcommands nor flags",
			help:     cli.Help{Usage: "tonestack"},
			contains: []string{"USAGE"},
			absent:   []string{"COMMANDS", "FLAGS"},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var out bytes.Buffer

			s.Require().NoError(tt.help.Render(&out))

			for _, want := range tt.contains {
				s.Require().Contains(out.String(), want)
			}

			for _, unwanted := range tt.absent {
				s.Require().NotContains(out.String(), unwanted)
			}
		})
	}
}

// TestRenderReportsAWriterThatFails covers a page nobody can read. Each
// section is a separate write, so a writer failing at any point must surface
// rather than leaving a half-rendered page and a success. Ten writes make a
// complete page: title, description, three headings with a row each, the
// footer, and the closing newline.
func (s *HelpPublicTestSuite) TestRenderReportsAWriterThatFails() {
	full := cli.Help{
		Name:        "n",
		Description: "d",
		Usage:       "u",
		Commands:    []cli.Item{{Name: "c"}},
		Flags:       []cli.Item{{Name: "f"}},
		Footer:      "foot",
	}

	for i := range 10 {
		s.Run(fmt.Sprintf("after %d writes", i), func() {
			s.Require().Error(full.Render(&failAfter{n: i}))
		})
	}
}

func TestHelpPublicTestSuite(t *testing.T) {
	suite.Run(t, new(HelpPublicTestSuite))
}

// failAfter fails once it has accepted n writes, so a page written in parts
// can be failed at each seam.
type failAfter struct {
	n int
	c int
}

func (w *failAfter) Write(p []byte) (int, error) {
	w.c++
	if w.c > w.n {
		return 0, errors.New("boom")
	}

	return len(p), nil
}
