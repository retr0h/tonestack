//go:build unix

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

package atomicfile_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/internal/atomicfile"
)

// AtomicfilePublicTestSuite covers writing a file whole or not at all.
//
// Unix only, because the one failure no directory can arrange, a write that
// stops partway, is arranged here with a file size limit.
type AtomicfilePublicTestSuite struct {
	suite.Suite
}

// writer is either of the two functions under test.
type writer func(string, []byte, os.FileMode) error

// setup is where a case writes, and what is already there.
type setup struct {
	// a file already at the path.
	existing bool
	// the mode that file has. Zero means 0600.
	mode os.FileMode
	// a directory already at the path, which nothing can be renamed over.
	directory bool
	// a parent that cannot be written into.
	readOnly bool
	// a parent that can be written into but not opened, so it cannot be
	// synced.
	writeOnly bool
	// a parent that is not there.
	missing bool
	// a file size limit, so the write stops partway.
	limited bool
}

// arrange builds a case's directory and returns the path to write.
func (s *AtomicfilePublicTestSuite) arrange(
	tt setup,
) string {
	dir := s.T().TempDir()
	path := filepath.Join(dir, "preset.hlx")

	mode := tt.mode
	if mode == 0 {
		mode = 0o600
	}

	if tt.existing {
		s.Require().NoError(os.WriteFile(path, []byte("old"), 0o600))
		s.Require().NoError(os.Chmod(path, mode))
	}

	switch {
	case tt.directory:
		s.Require().NoError(os.MkdirAll(filepath.Join(path, "full"), 0o750))
	case tt.readOnly:
		s.Require().NoError(os.Chmod(dir, 0o500))
		s.T().Cleanup(func() { s.Require().NoError(os.Chmod(dir, 0o700)) })
	case tt.writeOnly:
		s.Require().NoError(os.Chmod(dir, 0o300))
		s.T().Cleanup(func() { s.Require().NoError(os.Chmod(dir, 0o700)) })
	case tt.missing:
		path = filepath.Join(dir, "no", "such", "preset.hlx")
	}

	return path
}

// run calls one writer, under a file size limit when a case asks for one.
//
// The limit is lifted before anything else happens, because it applies to
// the whole process, the test binary's own output included. A directory that
// could not be opened is opened up again afterwards, so what is in it can be
// checked.
func (s *AtomicfilePublicTestSuite) run(
	write writer,
	path string,
	tt setup,
) error {
	if tt.writeOnly {
		defer func() { s.Require().NoError(os.Chmod(filepath.Dir(path), 0o700)) }()
	}

	if !tt.limited {
		return write(path, []byte("new"), 0o600)
	}

	var was syscall.Rlimit

	s.Require().NoError(syscall.Getrlimit(syscall.RLIMIT_FSIZE, &was))
	s.Require().NoError(syscall.Setrlimit(syscall.RLIMIT_FSIZE,
		&syscall.Rlimit{Cur: 1, Max: was.Max}))

	err := write(path, []byte("new"), 0o600)

	s.Require().NoError(syscall.Setrlimit(syscall.RLIMIT_FSIZE, &was))

	return err
}

// leftovers names anything in the directory other than the file itself.
func (s *AtomicfilePublicTestSuite) leftovers(
	path string,
) []string {
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		return nil
	}

	out := []string(nil)

	for _, e := range entries {
		if e.Name() != filepath.Base(path) {
			out = append(out, e.Name())
		}
	}

	return out
}

// TestWrite covers putting a file in place, replacing what was there.
func (s *AtomicfilePublicTestSuite) TestWrite() {
	tests := []struct {
		name string
		setup
		// the mode the written file must have. Zero means 0600.
		wantMode os.FileMode
		errText  string
	}{
		{name: "a new file"},
		{
			// Replacing is the point of Write. Somebody saving an edited
			// setlist over the one they opened expects exactly that.
			name:  "a file already there",
			setup: setup{existing: true},
		},
		{
			// Somebody who opened their setlist up to a group expects it to
			// stay that way after an edit.
			name:     "a file already there with its own mode",
			setup:    setup{existing: true, mode: 0o640},
			wantMode: 0o640,
		},
		{
			name:    "a directory where the file would go",
			setup:   setup{directory: true},
			errText: "writing",
		},
		{
			name:    "a directory it cannot write into",
			setup:   setup{readOnly: true},
			errText: "writing",
		},
		{
			// The file is in place, but nothing says its name survives a
			// crash, and that is said rather than assumed.
			name:    "a directory it cannot sync",
			setup:   setup{writeOnly: true},
			errText: "syncing its directory",
		},
		{
			name:    "a directory that is not there",
			setup:   setup{missing: true},
			errText: "writing",
		},
		{
			// A disk that fills partway through. What was there stays, and
			// no half-written file is left beside it.
			name:    "a write that stops partway",
			setup:   setup{existing: true, limited: true},
			errText: "file too large",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			path := s.arrange(tt.setup)

			err := s.run(atomicfile.Write, path, tt.setup)

			s.Require().Empty(s.leftovers(path), "no temporary file is left behind")

			if tt.errText != "" {
				s.Require().ErrorContains(err, tt.errText)

				if tt.existing {
					got, readErr := os.ReadFile(path) //nolint:gosec // a path this test chose
					s.Require().NoError(readErr)
					s.Require().Equal("old", string(got))
				}

				return
			}

			s.Require().NoError(err)

			got, err := os.ReadFile(path) //nolint:gosec // a path this test chose
			s.Require().NoError(err)
			s.Require().Equal("new", string(got))

			want := tt.wantMode
			if want == 0 {
				want = 0o600
			}

			info, err := os.Stat(path)
			s.Require().NoError(err)
			s.Require().Equal(want, info.Mode().Perm())
		})
	}
}

// TestWriteNew covers putting a file in place only where nothing is.
func (s *AtomicfilePublicTestSuite) TestWriteNew() {
	tests := []struct {
		name string
		setup
		is      error
		errText string
		// the same write, tried again once the directory is fixed, succeeds.
		retry bool
	}{
		{name: "a new file"},
		{
			// A backup or a recipe somebody already has is never replaced.
			name:  "a file already there",
			setup: setup{existing: true},
			is:    fs.ErrExist,
		},
		{
			name:    "a directory it cannot write into",
			setup:   setup{readOnly: true},
			errText: "writing",
		},
		{
			// The link landed and the sync did not. Nothing is left at the
			// path, so the error is not a clash and a retry writes the file.
			name:    "a directory it cannot sync",
			setup:   setup{writeOnly: true},
			errText: "syncing its directory",
			retry:   true,
		},
		{
			name:    "a write that stops partway",
			setup:   setup{limited: true},
			errText: "file too large",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			path := s.arrange(tt.setup)

			err := s.run(atomicfile.WriteNew, path, tt.setup)

			s.Require().Empty(s.leftovers(path), "no temporary file is left behind")

			if tt.is != nil || tt.errText != "" {
				s.Require().Error(err)

				if tt.is != nil {
					s.Require().ErrorIs(err, tt.is)
				}

				if tt.errText != "" {
					s.Require().ErrorContains(err, tt.errText)
					s.Require().NotErrorIs(err, fs.ErrExist)
				}

				if tt.existing {
					got, readErr := os.ReadFile(path) //nolint:gosec // a path this test chose
					s.Require().NoError(readErr)
					s.Require().Equal("old", string(got))

					return
				}

				s.Require().NoFileExists(path)

				if tt.retry {
					s.Require().NoError(atomicfile.WriteNew(path, []byte("new"), 0o600),
						"a retry is not refused as a clash")
				}

				return
			}

			s.Require().NoError(err)

			got, err := os.ReadFile(path) //nolint:gosec // a path this test chose
			s.Require().NoError(err)
			s.Require().Equal("new", string(got))
		})
	}
}

func TestAtomicfilePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(AtomicfilePublicTestSuite))
}
