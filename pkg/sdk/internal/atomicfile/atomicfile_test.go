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

package atomicfile

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/stretchr/testify/suite"
)

// AtomicfileTestSuite covers what a temporary directory cannot arrange: a
// filesystem with no hard links, and one that cannot sync a directory.
type AtomicfileTestSuite struct {
	suite.Suite
}

// TestCreate covers WriteNew on a filesystem with no hard links, where the
// file is created in place.
func (s *AtomicfileTestSuite) TestCreate() {
	tests := []struct {
		name string
		// what the link is refused with.
		refusal error
		// a file already at the path.
		existing bool
		// a file size limit, so the write stops partway.
		limited bool
		is      error
		errText string
	}{
		{
			// What macOS answers for FAT and exFAT.
			name:    "a filesystem that says links are not supported",
			refusal: syscall.ENOTSUP,
		},
		{
			// What Linux answers for FAT.
			name:    "a filesystem that says links are not permitted",
			refusal: syscall.EPERM,
		},
		{
			// Created in place, it still never writes over a file.
			name:     "a file already there",
			refusal:  syscall.ENOTSUP,
			existing: true,
			is:       fs.ErrExist,
		},
		{
			name:    "a write that stops partway",
			refusal: syscall.ENOTSUP,
			limited: true,
			errText: "file too large",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			was := link
			defer func() { link = was }()

			link = func(oldname, newname string) error {
				return &os.LinkError{Op: "link", Old: oldname, New: newname, Err: tt.refusal}
			}

			dir := s.T().TempDir()
			path := filepath.Join(dir, "backup.bin")

			if tt.existing {
				s.Require().NoError(os.WriteFile(path, []byte("old"), 0o600))
			}

			err := s.write(path, tt.limited)

			entries, readErr := os.ReadDir(dir)
			s.Require().NoError(readErr)

			if tt.is != nil || tt.errText != "" {
				if tt.is != nil {
					s.Require().ErrorIs(err, tt.is)
				}

				if tt.errText != "" {
					s.Require().ErrorContains(err, tt.errText)
					s.Require().Empty(entries, "a partial file is removed")
				}

				return
			}

			s.Require().NoError(err)
			s.Require().Len(entries, 1, "no temporary file is left behind")

			got, err := os.ReadFile(path) //nolint:gosec // a path this test chose
			s.Require().NoError(err)
			s.Require().Equal("new", string(got))
		})
	}
}

// write calls WriteNew, under a file size limit when a case asks for one.
func (s *AtomicfileTestSuite) write(
	path string,
	limited bool,
) error {
	if !limited {
		return WriteNew(path, []byte("new"), 0o600)
	}

	var was syscall.Rlimit

	s.Require().NoError(syscall.Getrlimit(syscall.RLIMIT_FSIZE, &was))

	// Enough for the temporary file, so it is the write in place that stops.
	s.Require().NoError(syscall.Setrlimit(syscall.RLIMIT_FSIZE,
		&syscall.Rlimit{Cur: 3, Max: was.Max}))

	err := createPartially(path)

	s.Require().NoError(syscall.Setrlimit(syscall.RLIMIT_FSIZE, &was))

	return err
}

// createPartially writes more than the limit allows, straight to create.
func createPartially(
	path string,
) error {
	return create(path, []byte("longer than the limit"), 0o600)
}

// TestUnsynced covers which failures to sync a directory are reported.
//
// Called directly: a directory that opens and then fails to sync is not
// something a test can arrange, and one that cannot be opened is covered
// through Write.
func (s *AtomicfileTestSuite) TestUnsynced() {
	tests := []struct {
		name     string
		err      error
		reported bool
	}{
		{name: "nothing went wrong"},
		{name: "a filesystem that cannot sync a directory", err: syscall.EINVAL},
		{name: "a platform that does not support it", err: errors.ErrUnsupported},
		{name: "a directory it may not open", err: syscall.EACCES, reported: true},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			err := unsynced("preset.hlx", tt.err)

			if tt.reported {
				s.Require().ErrorContains(err, "syncing its directory")

				return
			}

			s.Require().NoError(err)
		})
	}
}

func TestAtomicfileTestSuite(t *testing.T) {
	suite.Run(t, new(AtomicfileTestSuite))
}
