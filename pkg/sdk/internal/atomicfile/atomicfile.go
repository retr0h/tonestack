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

// Package atomicfile writes files that appear whole or not at all.
//
// Every file this project writes is somebody's: a setlist, a preset, a rig, a
// backup of what a device held. A write that fails halfway must not leave a
// truncated file where a good one used to be, so the bytes go to a temporary
// file beside the target first, are synced to disk, and are moved into place
// in one step. The directory is synced afterwards so the new name survives a
// crash too.
package atomicfile

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
)

// link gives a file a second name.
//
// A variable so a test can stand in a filesystem that has no hard links,
// which no temporary directory provides.
var link = os.Link

// Write puts data at path, replacing whatever is there.
//
// The whole file appears at once or not at all: os.Rename replaces its target
// in one step, so a reader sees the old file or the new one and never part of
// either.
//
// A regular file already at path keeps its permission bits, and perm applies
// only to a new one. A symbolic link at path is not followed: it is replaced
// by a regular file.
func Write(
	path string,
	data []byte,
	perm os.FileMode,
) error {
	if info, err := os.Lstat(path); err == nil && info.Mode().IsRegular() {
		perm = info.Mode().Perm()
	}

	tmp, err := temp(path, data, perm)
	if err != nil {
		return err
	}

	defer func() { _ = os.Remove(tmp) }()

	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}

	return syncDir(path)
}

// WriteNew puts data at path only if nothing is there yet.
//
// The whole file appears at once or not at all, and an existing file is left
// alone: os.Link refuses to replace its target.
//
// A filesystem with no hard links, FAT and exFAT among them, gets the file
// created in place instead. It is still never written over an existing one,
// but it is not atomic there: a crash partway can leave a partial file at
// path.
func WriteNew(
	path string,
	data []byte,
	perm os.FileMode,
) error {
	tmp, err := temp(path, data, perm)
	if err != nil {
		return err
	}

	defer func() { _ = os.Remove(tmp) }()

	err = link(tmp, path)
	if linkless(err) {
		return create(path, data, perm)
	}

	if err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}

	return syncDir(path)
}

// linkless reports a link refused because the filesystem has no hard links.
//
// Unsupported where the platform says so, and a permission error where Linux
// answers FAT that way. Falling back on a permission error that meant
// something else costs nothing: create refuses an existing file too, and
// fails on a directory it cannot write into.
func linkless(
	err error,
) bool {
	return errors.Is(err, errors.ErrUnsupported) || errors.Is(err, fs.ErrPermission)
}

// create writes path directly, for a filesystem with no hard links.
//
// O_EXCL refuses a file already there, as a link would. A file that could not
// be written in full is removed, but a crash before then leaves it partial.
func create(
	path string,
	data []byte,
	perm os.FileMode,
) error {
	f, err := os.OpenFile( //nolint:gosec // the path is the caller's own file
		path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
	if err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}

	_, err = f.Write(data)

	if err := errors.Join(err, f.Sync(), f.Close()); err != nil {
		_ = os.Remove(path)

		return fmt.Errorf("writing %s: %w", path, err)
	}

	return syncDir(path)
}

// temp writes data to a new file beside path and returns its name.
//
// Beside it rather than in the system's temporary directory, because a rename
// or a link only works within one filesystem. Synced before it is returned, so
// the name moved into place never points at data still only in memory. A
// temporary file that could not be written in full is removed.
func temp(
	path string,
	data []byte,
	perm os.FileMode,
) (string, error) {
	f, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return "", fmt.Errorf("writing %s: %w", path, err)
	}

	_, err = f.Write(data)

	// CreateTemp already makes the file 0600. Changing a mode to the one it
	// has is skipped, because a filesystem without modes refuses even that.
	if perm != 0o600 {
		err = errors.Join(err, f.Chmod(perm))
	}

	// One check for all of them: each leaves a file that is not what was
	// asked for, and each is answered the same way.
	if err := errors.Join(err, f.Sync(), f.Close()); err != nil {
		_ = os.Remove(f.Name())

		return "", fmt.Errorf("writing %s: %w", path, err)
	}

	return f.Name(), nil
}

// syncDir syncs the directory holding path, so a new name in it survives a
// crash.
func syncDir(
	path string,
) error {
	d, err := os.Open(filepath.Dir(path))
	if err != nil {
		return unsynced(path, err)
	}

	return unsynced(path, errors.Join(d.Sync(), d.Close()))
}

// unsynced reports a directory that could not be synced.
//
// A platform or filesystem that cannot sync a directory at all is not a
// failure: Windows cannot flush one, and some filesystems answer EINVAL. The
// file is in place either way; only its durability is up to the system.
func unsynced(
	path string,
	err error,
) error {
	if err == nil ||
		errors.Is(err, errors.ErrUnsupported) ||
		errors.Is(err, syscall.EINVAL) ||
		runtime.GOOS == "windows" {
		return nil
	}

	return fmt.Errorf("writing %s: syncing its directory: %w", path, err)
}
