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

package presets_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/internal/fileslots"
	"github.com/retr0h/tonestack/pkg/sdk/internal/presets"
	presetmocks "github.com/retr0h/tonestack/pkg/sdk/internal/presets/mocks"
	"github.com/retr0h/tonestack/pkg/sdk/preset"
	"github.com/retr0h/tonestack/pkg/sdk/result"
	"github.com/retr0h/tonestack/pkg/sdk/rig"
	"github.com/retr0h/tonestack/pkg/sdk/slot"
)

type CompilePublicTestSuite struct {
	suite.Suite
}

// slotFixture is a file the slot flows test against, which a rig is exported
// out of here.
func slotFixture(
	name string,
) string {
	return filepath.Join("..", "fileslots", "testdata", name)
}

// catalogs hands over the catalog at path, however often it is asked.
func (s *CompilePublicTestSuite) catalogs(
	path string,
) *presetmocks.MockCatalogs {
	c := presetmocks.NewMockCatalogs(gomock.NewController(s.T()))
	c.EXPECT().Catalog(gomock.Any()).DoAndReturn(
		func(context.Context) (*catalog.Catalog, error) { return catalog.Open(path) },
	).AnyTimes()

	return c
}

// handWritten returns a rig nobody lifted from a preset: gear and nothing
// else, which is what somebody typing one produces.
func (s *CompilePublicTestSuite) handWritten(
	dir string,
) string {
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
func (s *CompilePublicTestSuite) exported(
	dir string,
) string {
	out := filepath.Join(dir, "rig.yaml")

	_, err := (&fileslots.Flows{Catalogs: s.catalogs(slotFixture("catalog.json"))}).
		Export(context.Background(),
			slotFixture("setlist.hls"), slot.Address{}, out, result.FormatRig,
			result.ReplaceExisting)
	s.Require().NoError(err)

	return out
}

// unknownGear writes a valid rig naming gear no catalog carries.
func (s *CompilePublicTestSuite) unknownGear(
	dir string,
) string {
	path := filepath.Join(dir, "unknown.yaml")
	s.Require().NoError(os.WriteFile(path, []byte(
		"schema: RigSpec\nid: unknown\nsubject: {kind: sound, name: Unknown}\n"+
			"instrument: guitar\nchain:\n  - {role: amp, gear: Nonesuch 900}\n"),
		0o600))

	return path
}

// emptyChain writes a rig with no chain, which the contract refuses.
func (s *CompilePublicTestSuite) emptyChain(
	dir string,
) string {
	path := filepath.Join(dir, "bad.yaml")
	s.Require().NoError(os.WriteFile(path, []byte(
		"schema: RigSpec\nid: x\nsubject: {kind: artist, name: X}\n"+
			"instrument: bass\nchain: []\n"), 0o600))

	return path
}

// TestCompile covers turning a rig into a preset.
func (s *CompilePublicTestSuite) TestCompile() {
	tests := []struct {
		name string
		ctx  context.Context
		// which rig to build: one exported from a slot unless a case says
		// otherwise.
		rig string
		// a preset to write the chain into, rather than an untouched one.
		template string
		catalog  string
		out      string

		// how many blocks the built preset must report.
		blocks int
		// what the written preset must say, and must not.
		contains []string
		absent   []string
		// the chain must come back out of what was written.
		loadable bool

		// a compiler that leaves the preset unable to encode.
		unencodable bool

		err     error
		errText string
	}{
		{
			name:     "a rig lifted off a slot",
			blocks:   3,
			loadable: true,
		},
		{
			// A device expects inputs, outputs, a split and a join around a
			// chain. 98.6% of real presets carry them, and one assembled from
			// nothing carries none, so a compiled preset is written into an
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
			template: slotFixture("preset.hlx"),
			absent:   []string{"controller"},
		},
		{
			name:     "a typed rig, written into a template",
			rig:      "hand-written",
			template: slotFixture("preset.hlx"),
			// A rig nobody lifted carries no state, so the template's is
			// kept.
			contains: []string{"controller"},
		},
		{name: "a rig that is not there", rig: slotFixture("nope.yaml"), errText: "opening"},
		{
			name:    "a file that is not a rig",
			rig:     slotFixture("setlist.hls"),
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
			errText: "emulates \"Nonesuch 900\"",
		},
		{
			name:     "a template that is not there",
			template: slotFixture("nope.hlx"),
			errText:  "opening",
		},
		{
			name:     "a template that is not a preset",
			template: slotFixture("notapreset.hlx"),
			errText:  "reading",
		},
		{
			name:    "a catalog that is not there",
			catalog: slotFixture("nope.json"),
			errText: "catalog",
		},
		{
			// Reported, rather than a file holding nothing where a preset
			// was meant to be.
			name:        "a preset that will not encode",
			unencodable: true,
			errText:     "encoding preset",
		},
		{
			name:    "a destination directory that is not there",
			out:     filepath.Join("no", "out.hlx"),
			errText: "writing",
		},
		{
			name:    "a caller who stopped waiting",
			ctx:     cancelledContext(),
			errText: context.Canceled.Error(),
		},
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

			cat := slotFixture("catalog.json")
			if tt.catalog != "" {
				cat = tt.catalog
			}

			deps := presets.Deps{Catalogs: s.catalogs(cat)}

			if tt.unencodable {
				compiler := presetmocks.NewMockCompiler(gomock.NewController(s.T()))
				compiler.EXPECT().
					Lower(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(doc *preset.Document, _ rig.Spec, _ *catalog.Catalog) error {
						doc.Meta = json.RawMessage("{")

						return nil
					})

				deps.Compiler = compiler
			}

			ctx := tt.ctx
			if ctx == nil {
				ctx = context.Background()
			}

			got, err := presets.Compile(ctx, presets.CompileOptions{
				Deps:         deps,
				RigPath:      rigPath,
				TemplatePath: tt.template,
				OutputPath:   out,
			})

			if tt.err != nil || tt.errText != "" {
				s.Require().Error(err)
				s.Require().NoFileExists(out)

				if tt.err != nil {
					s.Require().ErrorIs(err, tt.err)
				}

				if tt.errText != "" {
					s.Require().ErrorContains(err, tt.errText)
				}

				return
			}

			s.Require().NoError(err)
			s.Require().Equal(out, got.Path)
			s.Require().NotEmpty(got.Name)

			if tt.blocks != 0 {
				s.Require().Equal(tt.blocks, got.Blocks)
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

func TestCompilePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(CompilePublicTestSuite))
}
