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
// crash too, on every platform but Windows, which cannot sync a directory.
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

// openFile opens a file.
//
// A variable so a test can have another program act on a name between the
// moment this package claims it and what it does next, which no real
// directory lets a test time.
var openFile = os.OpenFile

// dirSyncs is whether this platform can sync a directory at all.
//
// Windows cannot: a directory there is not something a program flushes, and
// asking fails whatever state the directory is in. So on Windows no directory
// sync is attempted, and a new name's durability is up to the filesystem.
// Everywhere else the sync is attempted and a failure is reported, except the
// two answers that mean the filesystem cannot do it either. See unsynced.
//
// A variable so a test on another platform can take the Windows path.
var dirSyncs = runtime.GOOS != "windows"

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

	tmp, _, err := temp(path, data, perm)
	if err != nil {
		return err
	}

	// Gone once renamed. Left only when the rename failed, and that error is returned.
	defer func() { _ = os.Remove(tmp) }()

	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}

	return syncDir(path)
}

// WriteNew puts data at path only if nothing is there yet.
//
// The whole file appears at once or not at all, and an existing file is left
// alone: os.Link refuses to replace its target. A file already there is
// refused with an error matching fs.ErrExist, and no other failure matches it.
//
// A failure leaves nothing of this write's at path, a failed directory sync
// included. The link has landed by then, but a name nobody can count on
// surviving a crash is not a finished write, and one left behind would make
// the caller's retry fail as though somebody else's file were there. So the
// file is removed, and the error is the sync's own. It is removed only while
// it is still the file this write put there: one another program has renamed
// over it since is theirs, and is left. When the removal fails, both failures
// are returned and the file stays.
//
// A filesystem with no hard links, FAT and exFAT among them, gets the file
// created in place instead. See create.
func WriteNew(
	path string,
	data []byte,
	perm os.FileMode,
) error {
	tmp, ours, err := temp(path, data, perm)
	if err != nil {
		return err
	}

	// A spare name once linked, and the error that matters is returned either
	// way.
	defer func() { _ = os.Remove(tmp) }()

	err = link(tmp, path)
	if linkless(err) {
		ours, err = create(path, data, perm)
	}

	if err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}

	if err := syncDir(path); err != nil {
		return errors.Join(err, unlink(path, ours))
	}

	return nil
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

// create writes path directly, for a filesystem with no hard links, and
// returns the file it wrote.
//
// O_EXCL refuses a file already there, as a link would, and the data goes
// through the handle that exclusive create returned rather than to the name.
// So another program that puts its own file at path meanwhile, as a
// concurrent Write does by renaming over it, keeps it: this write lands in a
// file nobody can reach any more, and nothing of theirs is written over or
// removed.
//
// That costs writing the data a second time, after temp. Renaming the
// temporary file over an empty claim would write it once, but the rename would
// replace whatever had taken the name by then, and a failed rename's cleanup
// would remove it. A few kilobytes are cheaper than somebody's file.
//
// It is not atomic: a crash partway can leave a partial file at path. A file
// that could not be written in full is removed, if it is still this write's.
func create(
	path string,
	data []byte,
	perm os.FileMode,
) (os.FileInfo, error) {
	f, err := openFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
	if err != nil {
		return nil, err
	}

	_, err = f.Write(data)
	info, statErr := f.Stat()

	if err := errors.Join(err, statErr, f.Sync(), f.Close()); err != nil {
		// Best effort: the write error is the one returned.
		_ = unlink(path, info)

		return nil, err
	}

	return info, nil
}

// unlink removes path, if it is still ours.
//
// Another program may have renamed its own file over the name since this
// write put a file there, and that one is not this write's to remove. The
// look and the removal are two steps, so a file renamed over path between
// them is still removed; the gap is two system calls rather than a whole write.
func unlink(
	path string,
	ours os.FileInfo,
) error {
	now, err := os.Lstat(path)
	if err != nil || !os.SameFile(now, ours) {
		return nil
	}

	if err := os.Remove(path); err != nil {
		return fmt.Errorf("writing %s: removing it after: %w", path, err)
	}

	return nil
}

// temp writes data to a new file beside path and returns its name, and the
// file it is, so a caller can later tell it from one put at the same name.
//
// Beside it rather than in the system's temporary directory, because a rename
// or a link only works within one filesystem. Synced before it is returned, so
// the name moved into place never points at data still only in memory. A
// temporary file that could not be written in full is removed.
func temp(
	path string,
	data []byte,
	perm os.FileMode,
) (string, os.FileInfo, error) {
	f, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return "", nil, fmt.Errorf("writing %s: %w", path, err)
	}

	_, err = f.Write(data)

	// CreateTemp already makes the file 0600. Changing a mode to the one it
	// has is skipped, because a filesystem without modes refuses even that.
	if perm != 0o600 {
		err = errors.Join(err, f.Chmod(perm))
	}

	info, statErr := f.Stat()

	// One check for all of them: each leaves a file that is not what was
	// asked for, and each is answered the same way.
	if err := errors.Join(err, statErr, f.Sync(), f.Close()); err != nil {
		// Best effort: the write error is the one returned.
		_ = os.Remove(f.Name())

		return "", nil, fmt.Errorf("writing %s: %w", path, err)
	}

	return f.Name(), info, nil
}

// syncDir syncs the directory holding path, so a new name in it survives a
// crash. On a platform that cannot sync a directory it does nothing.
func syncDir(
	path string,
) error {
	if !dirSyncs {
		return nil
	}

	d, err := os.Open(filepath.Dir(path))
	if err != nil {
		return unsynced(path, err)
	}

	return unsynced(path, errors.Join(d.Sync(), d.Close()))
}

// unsynced reports a directory that could not be synced.
//
// A filesystem that cannot sync a directory at all is not a failure: it
// answers unsupported or EINVAL, and the file is in place either way, with its
// durability up to the system. Every other failure is reported, on every
// platform that attempts the sync. Windows does not attempt it; see dirSyncs.
func unsynced(
	path string,
	err error,
) error {
	if err == nil ||
		errors.Is(err, errors.ErrUnsupported) ||
		errors.Is(err, syscall.EINVAL) {
		return nil
	}

	return fmt.Errorf("writing %s: syncing its directory: %w", path, err)
}
