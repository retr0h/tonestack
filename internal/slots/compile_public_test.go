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

// exported writes a slot out as a rig and returns where it went.
func (s *CompilePublicTestSuite) exported(dir string) string {
	out := filepath.Join(dir, "rig.yaml")

	s.Require().NoError(slots.Export(&bytes.Buffer{}, slots.ExportOptions{
		Path: fixture("setlist.hls"), Slot: 0, OutputPath: out,
		CatalogPath: catalogPath(),
	}))

	return out
}

func (s *CompilePublicTestSuite) TestARigBecomesAPresetAndBack() {
	dir := s.T().TempDir()
	out := filepath.Join(dir, "out.hlx")

	var log bytes.Buffer
	s.Require().NoError(slots.Compile(&log, slots.CompileOptions{
		RigPath: s.exported(dir), OutputPath: out, CatalogPath: catalogPath(),
	}))

	s.Require().Contains(log.String(), "in the chain")

	f, err := os.Open(out) //nolint:gosec // a path this test chose
	s.Require().NoError(err)

	defer func() { s.Require().NoError(f.Close()) }()

	doc, err := preset.Read(f)
	s.Require().NoError(err)

	c, err := doc.Spec()
	s.Require().NoError(err)
	s.Require().NotEmpty(c.Blocks)
}

func (s *CompilePublicTestSuite) TestTheResultCarriesWhatADeviceExpects() {
	// A device expects inputs, outputs, a split and a join around a chain.
	// 98.6% of real presets carry them, and one assembled from nothing
	// carries none — so a compiled preset is written into an untouched one.
	dir := s.T().TempDir()
	out := filepath.Join(dir, "out.hlx")

	s.Require().NoError(slots.Compile(&bytes.Buffer{}, slots.CompileOptions{
		RigPath: s.exported(dir), OutputPath: out, CatalogPath: catalogPath(),
	}))

	raw, err := os.ReadFile(out) //nolint:gosec // a path this test chose
	s.Require().NoError(err)

	for _, want := range []string{"inputA", "outputA", "split", "join", "snapshot0"} {
		s.Require().Contains(string(raw), want)
	}
}

func (s *CompilePublicTestSuite) TestATemplateIsWrittenInto() {
	dir := s.T().TempDir()
	out := filepath.Join(dir, "out.hlx")

	s.Require().NoError(slots.Compile(&bytes.Buffer{}, slots.CompileOptions{
		RigPath: s.exported(dir), OutputPath: out, CatalogPath: catalogPath(),
		TemplatePath: fixture("preset.hlx"),
	}))

	raw, err := os.ReadFile(out) //nolint:gosec // a path this test chose
	s.Require().NoError(err)
	s.Require().Contains(string(raw), "controller",
		"whatever the template held that a rig does not model is still there")
}

func (s *CompilePublicTestSuite) TestReportsProblems() {
	dir := s.T().TempDir()

	tests := []struct {
		name    string
		mutate  func(*slots.CompileOptions)
		message string
	}{
		{
			"a rig that is not there",
			func(o *slots.CompileOptions) { o.RigPath = fixture("nope.yaml") },
			"opening",
		},
		{
			"a file that is not a rig",
			func(o *slots.CompileOptions) { o.RigPath = fixture("setlist.hls") },
			"not a valid rig",
		},
		{
			"a template that is not there",
			func(o *slots.CompileOptions) { o.TemplatePath = fixture("nope.hlx") },
			"opening",
		},
		{
			"a template that is not a preset",
			func(o *slots.CompileOptions) { o.TemplatePath = fixture("notapreset.hlx") },
			"reading",
		},
		{
			"a catalog that is not there",
			func(o *slots.CompileOptions) { o.CatalogPath = fixture("nope.json") },
			"catalog",
		},
		{
			"a rig naming gear this device does not model",
			func(o *slots.CompileOptions) { o.RigPath = s.unknownGear(dir) },
			"nothing on this device is",
		},
		{
			"a destination directory that is not there",
			func(o *slots.CompileOptions) {
				o.OutputPath = filepath.Join(dir, "no", "out.hlx")
			},
			"writing",
		},
	}

	rigPath := s.exported(dir)

	for _, tc := range tests {
		s.Run(tc.name, func() {
			o := slots.CompileOptions{
				RigPath:     rigPath,
				OutputPath:  filepath.Join(dir, "out.hlx"),
				CatalogPath: catalogPath(),
			}
			tc.mutate(&o)

			err := slots.Compile(&bytes.Buffer{}, o)

			s.Require().Error(err)
			s.Require().Contains(err.Error(), tc.message)
		})
	}
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

func (s *CompilePublicTestSuite) TestReportsAWriterThatFails() {
	dir := s.T().TempDir()

	s.Require().Error(slots.Compile(&failingWriter{}, slots.CompileOptions{
		RigPath:     s.exported(dir),
		OutputPath:  filepath.Join(dir, "out.hlx"),
		CatalogPath: catalogPath(),
	}))
}

func (s *CompilePublicTestSuite) TestARigMustMeetItsOwnContract() {
	dir := s.T().TempDir()
	bad := filepath.Join(dir, "bad.yaml")
	s.Require().NoError(os.WriteFile(bad, []byte(
		"schema: RigSpec\nid: x\nsubject: {kind: artist, name: X}\n"+
			"instrument: bass\nchain: []\n"), 0o600))

	err := slots.Compile(&bytes.Buffer{}, slots.CompileOptions{
		RigPath: bad, OutputPath: filepath.Join(dir, "out.hlx"),
		CatalogPath: catalogPath(),
	})

	s.Require().ErrorIs(err, rig.ErrInvalid)
}

func TestCompilePublicTestSuite(t *testing.T) {
	suite.Run(t, new(CompilePublicTestSuite))
}
