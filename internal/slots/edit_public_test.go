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
	"github.com/retr0h/tonestack/pkg/setlist"
)

type EditPublicTestSuite struct {
	suite.Suite
}

// reread loads a setlist this suite just wrote.
func (s *EditPublicTestSuite) reread(path string) *setlist.Document {
	f, err := os.Open(path) //nolint:gosec // a path this test chose
	s.Require().NoError(err)

	defer func() { s.Require().NoError(f.Close()) }()

	doc, err := setlist.Read(f)
	s.Require().NoError(err)

	return doc
}

func (s *EditPublicTestSuite) opts(out string) slots.EditOptions {
	return slots.EditOptions{
		Path: fixture("setlist.hls"), FromSlot: 0, ToSlot: 1, OutputPath: out,
	}
}

func (s *EditPublicTestSuite) TestCopyLeavesTheSourceAlone() {
	out := filepath.Join(s.T().TempDir(), "out.hls")

	var log bytes.Buffer
	s.Require().NoError(slots.Copy(&log, s.opts(out)))

	doc := s.reread(out)
	s.Require().Equal("First", doc.Setlists[0].Slots[0].Meta.Name)
	s.Require().Equal("First", doc.Setlists[0].Slots[1].Meta.Name)
	s.Require().Contains(log.String(), "copied")
	s.Require().Contains(log.String(), "01A")
	s.Require().Contains(log.String(), "01B")
}

func (s *EditPublicTestSuite) TestSwapExchangesBothSlots() {
	out := filepath.Join(s.T().TempDir(), "out.hls")

	var log bytes.Buffer
	s.Require().NoError(slots.Swap(&log, s.opts(out)))

	doc := s.reread(out)
	s.Require().Equal("Second", doc.Setlists[0].Slots[0].Meta.Name)
	s.Require().Equal("First", doc.Setlists[0].Slots[1].Meta.Name)
	s.Require().Contains(log.String(), "swapped")
}

func (s *EditPublicTestSuite) TestEditsReportProblems() {
	dir := s.T().TempDir()

	tests := []struct {
		name    string
		mutate  func(*slots.EditOptions)
		message string
	}{
		{
			"a file that is not there",
			func(o *slots.EditOptions) { o.Path = fixture("nope.hls") },
			"opening",
		},
		{
			"a source that is not there",
			func(o *slots.EditOptions) { o.FromSlot = 99 },
			"no such slot",
		},
		{
			"a destination that is not there",
			func(o *slots.EditOptions) { o.ToSlot = 99 },
			"no such slot",
		},
		{
			"a destination directory that is not there",
			func(o *slots.EditOptions) { o.OutputPath = filepath.Join(dir, "no", "out.hls") },
			"writing",
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			opts := s.opts(filepath.Join(dir, "out.hls"))
			tc.mutate(&opts)

			err := slots.Copy(&bytes.Buffer{}, opts)

			s.Require().Error(err)
			s.Require().Contains(err.Error(), tc.message)
		})
	}
}

func (s *EditPublicTestSuite) TestEditReportsAWriterThatFails() {
	out := filepath.Join(s.T().TempDir(), "out.hls")

	s.Require().Error(slots.Copy(&failingWriter{}, s.opts(out)))
}

func (s *EditPublicTestSuite) TestExportWritesARigByDefault() {
	// A rig is what this project speaks, and the format that reads on other
	// hardware. The device's own file is a faithful copy, which is a
	// different thing and has to be asked for.
	out := filepath.Join(s.T().TempDir(), "one.yaml")

	var log bytes.Buffer
	s.Require().NoError(slots.Export(&log, slots.ExportOptions{
		Path: fixture("setlist.hls"), Slot: 0, OutputPath: out,
		CatalogPath: catalogPath(),
	}))

	raw, err := os.ReadFile(out) //nolint:gosec // a path this test chose
	s.Require().NoError(err)
	s.Require().Contains(string(raw), "schema: RigSpec")
	s.Require().Contains(string(raw), "gear:")
	s.Require().Contains(string(raw), "models:",
		"a lifted rig records the exact model, since a name does not identify one")
}

func (s *EditPublicTestSuite) TestExportCanWriteTheDevicesOwnFile() {
	out := filepath.Join(s.T().TempDir(), "one.hlx")

	var log bytes.Buffer
	s.Require().NoError(slots.Export(&log, slots.ExportOptions{
		Path: fixture("setlist.hls"), Slot: 0, OutputPath: out,
		As: slots.FormatPreset,
	}))

	var show bytes.Buffer
	s.Require().NoError(slots.Show(&show, slots.ShowOptions{
		File: out, CatalogPath: catalogPath(),
	}))

	s.Require().Contains(show.String(), "First")
	s.Require().Contains(show.String(), "Ampeg SVT")
	s.Require().Contains(log.String(), "wrote")
}

func (s *EditPublicTestSuite) TestExportReportsProblems() {
	dir := s.T().TempDir()

	tests := []struct {
		name    string
		opts    slots.ExportOptions
		message string
	}{
		{
			"a file that is not there",
			slots.ExportOptions{Path: fixture("nope.hls"), OutputPath: dir + "/x.hlx"},
			"opening",
		},
		{
			"a slot that is not there",
			slots.ExportOptions{
				Path: fixture("setlist.hls"), Slot: 99, OutputPath: dir + "/x.hlx",
				As: slots.FormatPreset,
			},
			"no such slot",
		},
		{
			"a destination directory that is not there",
			slots.ExportOptions{
				Path: fixture("setlist.hls"), OutputPath: filepath.Join(dir, "no", "x.hlx"),
				As: slots.FormatPreset,
			},
			"writing",
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			err := slots.Export(&bytes.Buffer{}, tc.opts)

			s.Require().Error(err)
			s.Require().Contains(err.Error(), tc.message)
		})
	}
}

func (s *EditPublicTestSuite) TestExportReportsProblemsWritingARig() {
	dir := s.T().TempDir()

	tests := []struct {
		name    string
		mutate  func(*slots.ExportOptions)
		message string
	}{
		{
			"a catalog that is not there",
			func(o *slots.ExportOptions) { o.CatalogPath = fixture("nope.json") },
			"catalog",
		},
		{
			"a slot holding nothing, which is not a rig",
			func(o *slots.ExportOptions) { o.Slot = 2 },
			"chain minimum number of items is 1",
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			o := slots.ExportOptions{
				Path: fixture("setlist.hls"), Slot: 0,
				OutputPath: filepath.Join(dir, "x.yaml"), CatalogPath: catalogPath(),
			}
			tc.mutate(&o)

			err := slots.Export(&bytes.Buffer{}, o)

			s.Require().Error(err)
			s.Require().Contains(err.Error(), tc.message)
		})
	}
}

func (s *EditPublicTestSuite) TestExportReportsAWriterThatFails() {
	s.Require().Error(slots.Export(&failingWriter{}, slots.ExportOptions{
		Path:       fixture("setlist.hls"),
		OutputPath: filepath.Join(s.T().TempDir(), "x.hlx"),
		As:         slots.FormatPreset,
	}))
}

func (s *EditPublicTestSuite) TestImportPlacesAPreset() {
	dir := s.T().TempDir()
	pre := filepath.Join(dir, "one.hlx")

	s.Require().NoError(slots.Export(&bytes.Buffer{}, slots.ExportOptions{
		Path: fixture("setlist.hls"), Slot: 0, OutputPath: pre,
		As: slots.FormatPreset,
	}))

	out := filepath.Join(dir, "out.hls")

	var log bytes.Buffer
	s.Require().NoError(slots.Import(&log, slots.ImportOptions{
		Path: fixture("setlist.hls"), File: pre, Slot: 1, OutputPath: out,
	}))

	doc := s.reread(out)
	s.Require().Equal("First", doc.Setlists[0].Slots[1].Meta.Name)
	s.Require().Contains(log.String(), "replaced")
	s.Require().Contains(log.String(), "Second")
}

func (s *EditPublicTestSuite) TestImportWarnsAboutAnotherDevice() {
	out := filepath.Join(s.T().TempDir(), "out.hls")

	var log bytes.Buffer
	s.Require().NoError(slots.Import(&log, slots.ImportOptions{
		Path: fixture("setlist.hls"), File: fixture("otherdevice.hlx"),
		Slot: 1, OutputPath: out,
	}))

	s.Require().Contains(log.String(), "different device")
}

func (s *EditPublicTestSuite) TestImportReportsProblems() {
	dir := s.T().TempDir()
	good := slots.ImportOptions{
		Path: fixture("setlist.hls"), File: fixture("preset.hlx"),
		OutputPath: filepath.Join(dir, "out.hls"),
	}

	tests := []struct {
		name    string
		mutate  func(*slots.ImportOptions)
		message string
	}{
		{
			"a setlist that is not there",
			func(o *slots.ImportOptions) { o.Path = fixture("nope.hls") },
			"opening",
		},
		{
			"a preset that is not there",
			func(o *slots.ImportOptions) { o.File = fixture("nope.hlx") },
			"opening",
		},
		{
			"a file that is not a preset",
			func(o *slots.ImportOptions) { o.File = fixture("notapreset.hlx") },
			"not a preset",
		},
		{
			"a slot that is not there",
			func(o *slots.ImportOptions) { o.Slot = 99 },
			"no such slot",
		},
		{
			"a destination directory that is not there",
			func(o *slots.ImportOptions) { o.OutputPath = filepath.Join(dir, "no", "o.hls") },
			"writing",
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			opts := good
			tc.mutate(&opts)

			err := slots.Import(&bytes.Buffer{}, opts)

			s.Require().Error(err)
			s.Require().Contains(err.Error(), tc.message)
		})
	}
}

func (s *EditPublicTestSuite) TestImportReportsAWriterThatFails() {
	dir := s.T().TempDir()

	tests := []struct {
		name string
		w    interface{ Write([]byte) (int, error) }
		file string
	}{
		{"with no warning", &failingWriter{}, fixture("preset.hlx")},
		{"on the warning", &failingWriter{}, fixture("otherdevice.hlx")},
		{"after the warning", &oneGoodWrite{}, fixture("otherdevice.hlx")},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			err := slots.Import(tc.w, slots.ImportOptions{
				Path: fixture("setlist.hls"), File: tc.file, Slot: 1,
				OutputPath: filepath.Join(dir, "out.hls"),
			})

			s.Require().Error(err)
		})
	}
}

func TestEditPublicTestSuite(t *testing.T) {
	suite.Run(t, new(EditPublicTestSuite))
}
