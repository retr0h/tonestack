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
// filesystem with no hard links, a platform that cannot sync a directory, and
// a directory that changes between a link and its sync.
type AtomicfileTestSuite struct {
	suite.Suite
}

// refuseLinks stands in a filesystem with no hard links for the rest of the
// test. before runs first, given the temporary file WriteNew would have
// linked.
func (s *AtomicfileTestSuite) refuseLinks(
	refusal error,
	before func(tmp string),
) {
	was := link
	s.T().Cleanup(func() { link = was })

	link = func(oldname, newname string) error {
		before(oldname)

		return &os.LinkError{Op: "link", Old: oldname, New: newname, Err: refusal}
	}
}

// TestCreate covers WriteNew on a filesystem with no hard links, where the
// file is created in place through its own exclusive handle.
func (s *AtomicfileTestSuite) TestCreate() {
	tests := []struct {
		name string
		// a file size limit, so the write in place stops partway. Called
		// straight to create, because the temporary file would stop first.
		limited bool
		// what the link is refused with.
		refusal error
		// a file already at the path.
		existing bool
		// another program renames its own file over the name the moment it
		// is claimed, as a concurrent Write to the same path does.
		replaced bool
		// the directory cannot be opened once the link is refused, so it
		// cannot be synced.
		unsyncable bool
		is         error
		errText    string
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
			// Claimed in place, it still never writes over a file.
			name:     "a file already there",
			refusal:  syscall.ENOTSUP,
			existing: true,
			is:       fs.ErrExist,
		},
		{
			// A disk that fills partway through. No partial file is left.
			name:    "a write that stops partway",
			limited: true,
			errText: "file too large",
		},
		{
			// Somebody's file took the name after it was claimed. Theirs
			// stays: nothing is moved over it, and nothing removes it.
			name:     "another file put over the claim",
			refusal:  syscall.ENOTSUP,
			replaced: true,
		},
		{
			// As on a filesystem with links: a retry is not refused as a
			// clash with a file this write left.
			name:       "a directory it cannot sync",
			refusal:    syscall.ENOTSUP,
			unsyncable: true,
			errText:    "syncing its directory",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			dir := s.T().TempDir()
			path := filepath.Join(dir, "backup.bin")

			if tt.existing {
				s.Require().NoError(os.WriteFile(path, []byte("old"), 0o600))
			}

			s.refuseLinks(tt.refusal, func(string) {
				if tt.unsyncable {
					s.Require().NoError(os.Chmod(dir, 0o300))
				}
			})

			if tt.replaced {
				s.replaceOnClaim(path)
			}

			err := s.write(path, tt.limited)

			s.Require().NoError(os.Chmod(dir, 0o700))

			entries, readErr := os.ReadDir(dir)
			s.Require().NoError(readErr)

			if tt.replaced {
				s.Require().Len(entries, 1, "only their file is left")

				got, err := os.ReadFile(path) //nolint:gosec // a path this test chose
				s.Require().NoError(err)
				s.Require().Equal("theirs", string(got), "their file is not written over")

				return
			}

			if tt.is != nil {
				s.Require().ErrorIs(err, tt.is)
				s.Require().Len(entries, 1, "only the file already there is left")

				return
			}

			if tt.errText != "" {
				s.Require().ErrorContains(err, tt.errText)
				s.Require().NotErrorIs(err, fs.ErrExist)
				s.Require().Empty(entries, "nothing is left at the path or beside it")

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

// write calls WriteNew, or create under a file size limit when a case asks for
// one.
func (s *AtomicfileTestSuite) write(
	path string,
	limited bool,
) error {
	if !limited {
		return WriteNew(path, []byte("new"), 0o600)
	}

	var was syscall.Rlimit

	s.Require().NoError(syscall.Getrlimit(syscall.RLIMIT_FSIZE, &was))
	s.Require().NoError(syscall.Setrlimit(syscall.RLIMIT_FSIZE,
		&syscall.Rlimit{Cur: 3, Max: was.Max}))

	_, err := create(path, []byte("longer than the limit"), 0o600)

	s.Require().NoError(syscall.Setrlimit(syscall.RLIMIT_FSIZE, &was))

	return err
}

// replaceOnClaim has another program rename a file of its own over path the
// moment this package creates path exclusively, for the rest of the test.
func (s *AtomicfileTestSuite) replaceOnClaim(
	path string,
) {
	was := openFile
	s.T().Cleanup(func() { openFile = was })

	openFile = func(name string, flag int, perm os.FileMode) (*os.File, error) {
		f, err := was(name, flag, perm)
		if err != nil || name != path || flag&os.O_EXCL == 0 {
			return f, err
		}

		s.replace(path)

		return f, nil
	}
}

// replace renames a file saying "theirs" over path, as atomicfile.Write in
// another program does.
func (s *AtomicfileTestSuite) replace(
	path string,
) {
	theirs := path + ".theirs"
	s.Require().NoError(os.WriteFile(theirs, []byte("theirs"), 0o600))
	s.Require().NoError(os.Rename(theirs, path))
}

// TestWriteNew covers a sync that fails after the link has landed, where what
// WriteNew leaves decides whether a retry reads as a clash.
func (s *AtomicfileTestSuite) TestWriteNew() {
	tests := []struct {
		name string
		// the mode the directory is given once the link has landed.
		mode    os.FileMode
		errText []string
		// the file is left, because it could not be removed.
		left bool
		// another program renames its own file over the name after the link
		// lands, so the file at the path is no longer this write's.
		replaced bool
	}{
		{
			// Theirs is not this write's to remove.
			name:     "a directory it cannot sync, with the file since replaced",
			mode:     0o300,
			errText:  []string{"syncing its directory"},
			replaced: true,
		},
		{
			// Can be written into but not opened: the name goes, and a
			// retry writes the file.
			name:    "a directory it cannot sync",
			mode:    0o300,
			errText: []string{"syncing its directory"},
		},
		{
			// Can be neither opened nor written into: both failures are
			// said, because the file a retry will find is this one.
			name:    "a directory it can neither sync nor remove from",
			mode:    0o100,
			errText: []string{"syncing its directory", "removing it"},
			left:    true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			dir := s.T().TempDir()
			path := filepath.Join(dir, "backup.bin")

			was := link
			s.T().Cleanup(func() { link = was })

			link = func(oldname, newname string) error {
				if err := os.Link(oldname, newname); err != nil {
					return err
				}

				if tt.replaced {
					s.replace(newname)
				}

				return os.Chmod(dir, tt.mode)
			}

			err := WriteNew(path, []byte("new"), 0o600)

			s.Require().NoError(os.Chmod(dir, 0o700))

			for _, text := range tt.errText {
				s.Require().ErrorContains(err, text)
			}

			s.Require().NotErrorIs(err, fs.ErrExist)

			if tt.replaced {
				got, readErr := os.ReadFile(path) //nolint:gosec // a path this test chose
				s.Require().NoError(readErr)
				s.Require().Equal("theirs", string(got), "their file is left alone")

				return
			}

			if tt.left {
				s.Require().FileExists(path)

				return
			}

			s.Require().NoFileExists(path)

			link = was

			s.Require().NoError(WriteNew(path, []byte("new"), 0o600), "a retry is not a clash")
		})
	}
}

// TestSyncDir covers which platforms sync a directory at all.
//
// The directory cannot be opened, so an attempt to sync it fails. Only a
// platform that syncs directories makes the attempt.
func (s *AtomicfileTestSuite) TestSyncDir() {
	tests := []struct {
		name     string
		dirSyncs bool
		errText  string
	}{
		{
			name:     "a platform that syncs directories",
			dirSyncs: true,
			errText:  "syncing its directory",
		},
		{
			// Windows. Not attempted, so there is no failure to hide.
			name: "a platform that does not",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			was := dirSyncs
			s.T().Cleanup(func() { dirSyncs = was })

			dirSyncs = tt.dirSyncs

			dir := s.T().TempDir()
			s.Require().NoError(os.Chmod(dir, 0o300))
			s.T().Cleanup(func() { s.Require().NoError(os.Chmod(dir, 0o700)) })

			err := syncDir(filepath.Join(dir, "preset.hlx"))

			if tt.errText != "" {
				s.Require().ErrorContains(err, tt.errText)

				return
			}

			s.Require().NoError(err)
		})
	}
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

func TestAtomicfileTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(AtomicfileTestSuite))
}
