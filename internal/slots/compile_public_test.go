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

package slots_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/internal/slots"
	"github.com/retr0h/tonestack/pkg/preset"
	"github.com/retr0h/tonestack/pkg/rig"
)

type CompilePublicTestSuite struct {
	suite.Suite
}

// handWritten returns a rig nobody lifted from a preset: gear and nothing
// else, which is what somebody typing one produces.
func (s *CompilePublicTestSuite) handWritten(dir string) string {
	out := filepath.Join(dir, "typed.yaml")

	s.Require().NoError(os.WriteFile(out, []byte(`schema: RigSpec
version: 2
id: typed
subject: { kind: sound, name: Typed }
instrument: bass
chain:
  - { role: amp, gear: Ampeg SVT }
`), 0o600))

	return out
}

// exported writes a slot out as a rig and returns where it went.
func (s *CompilePublicTestSuite) exported(dir string) string {
	out := filepath.Join(dir, "rig.yaml")

	s.Require().NoError(slots.Export(&bytes.Buffer{}, slots.ExportOptions{
		Path: fixture("setlist.hls"), Slot: 0, OutputPath: out,
		CatalogPath: catalogPath(),
	}))

	return out
}

// unknownGear writes a valid rig naming gear no catalog carries.
func (s *CompilePublicTestSuite) unknownGear(dir string) string {
	path := filepath.Join(dir, "unknown.yaml")
	s.Require().NoError(os.WriteFile(path, []byte(
		"schema: RigSpec\nid: unknown\nsubject: {kind: sound, name: Unknown}\n"+
			"instrument: guitar\nchain:\n  - {role: amp, gear: Nonesuch 900}\n"),
		0o600))

	return path
}

// emptyChain writes a rig with no chain, which the contract refuses.
func (s *CompilePublicTestSuite) emptyChain(dir string) string {
	path := filepath.Join(dir, "bad.yaml")
	s.Require().NoError(os.WriteFile(path, []byte(
		"schema: RigSpec\nid: x\nsubject: {kind: artist, name: X}\n"+
			"instrument: bass\nchain: []\n"), 0o600))

	return path
}

// TestCompile turns a rig into a preset.
func (s *CompilePublicTestSuite) TestCompile() {
	tests := []struct {
		name string
		// which rig to build: one exported from a slot unless a case says
		// otherwise.
		rig string
		// a preset to write the chain into, rather than an untouched one.
		template string
		catalog  string
		out      string
		deaf     bool

		logs []string
		// what the written preset must say, and must not.
		contains []string
		absent   []string
		// the chain must come back out of what was written.
		loadable bool

		err     error
		errText string
	}{
		{
			name:     "a rig lifted off a slot",
			logs:     []string{"in the chain"},
			loadable: true,
		},
		{
			// A device expects inputs, outputs, a split and a join around a
			// chain. 98.6% of real presets carry them, and one assembled from
			// nothing carries none — so a compiled preset is written into an
			// untouched one.
			name:     "a rig somebody typed",
			rig:      "hand-written",
			contains: []string{"inputA", "outputA", "split", "join", "snapshot0"},
		},
		{
			// A lifted rig carries what the preset it came from carried, and
			// that wins over whatever the preset being written into holds.
			// Otherwise a rig shared with somebody else would rebuild with a
			// stranger's routing.
			name:     "a lifted rig, written into somebody else's preset",
			template: fixture("preset.hlx"),
			absent:   []string{"controller"},
		},
		{
			name:     "a typed rig, written into a template",
			rig:      "hand-written",
			template: fixture("preset.hlx"),
			// A rig nobody lifted carries no state, so the template's is
			// kept.
			contains: []string{"controller"},
		},
		{
			name:    "a rig that is not there",
			rig:     fixture("nope.yaml"),
			errText: "opening",
		},
		{
			name:    "a file that is not a rig",
			rig:     fixture("setlist.hls"),
			errText: "not a valid rig",
		},
		{
			name:    "a rig that does not meet its own contract",
			rig:     "empty chain",
			err:     rig.ErrInvalid,
			errText: "chain",
		},
		{
			name:    "a rig naming gear this device does not model",
			rig:     "unknown gear",
			errText: "nothing on this device is",
		},
		{
			name:     "a template that is not there",
			template: fixture("nope.hlx"),
			errText:  "opening",
		},
		{
			name:     "a template that is not a preset",
			template: fixture("notapreset.hlx"),
			errText:  "reading",
		},
		{
			name:    "a catalog that is not there",
			catalog: fixture("nope.json"),
			errText: "catalog",
		},
		{
			name:    "a destination directory that is not there",
			out:     filepath.Join("no", "out.hlx"),
			errText: "writing",
		},
		{name: "a writer that fails", deaf: true},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			dir := s.T().TempDir()

			out := filepath.Join(dir, "out.hlx")
			if tt.out != "" {
				out = filepath.Join(dir, tt.out)
			}

			var rigPath string

			switch tt.rig {
			case "":
				rigPath = s.exported(dir)
			case "hand-written":
				rigPath = s.handWritten(dir)
			case "unknown gear":
				rigPath = s.unknownGear(dir)
			case "empty chain":
				rigPath = s.emptyChain(dir)
			default:
				rigPath = tt.rig
			}

			catalog := catalogPath()
			if tt.catalog != "" {
				catalog = tt.catalog
			}

			var log bytes.Buffer

			w := io.Writer(&log)
			if tt.deaf {
				w = &failingWriter{}
			}

			err := slots.Compile(w, slots.CompileOptions{
				RigPath: rigPath, OutputPath: out, CatalogPath: catalog,
				TemplatePath: tt.template,
			})

			if tt.err != nil || tt.errText != "" || tt.deaf {
				s.Require().Error(err)

				if tt.err != nil {
					s.Require().ErrorIs(err, tt.err)
				}

				if tt.errText != "" {
					s.Require().Contains(err.Error(), tt.errText)
				}

				return
			}

			s.Require().NoError(err)

			for _, want := range tt.logs {
				s.Require().Contains(log.String(), want)
			}

			raw, err := os.ReadFile(out) //nolint:gosec // a path this test chose
			s.Require().NoError(err)

			for _, want := range tt.contains {
				s.Require().Contains(string(raw), want)
			}

			for _, unwanted := range tt.absent {
				s.Require().NotContains(string(raw), unwanted)
			}

			if !tt.loadable {
				return
			}

			doc, err := preset.Read(bytes.NewReader(raw))
			s.Require().NoError(err)

			c, err := doc.Spec()
			s.Require().NoError(err)
			s.Require().NotEmpty(c.Blocks)
		})
	}
}

func TestCompilePublicTestSuite(t *testing.T) {
	suite.Run(t, new(CompilePublicTestSuite))
}
