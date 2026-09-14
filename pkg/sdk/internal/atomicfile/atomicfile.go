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
// file beside the target first and are moved into place in one step.
package atomicfile

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Write puts data at path, replacing whatever is there.
//
// The whole file appears at once or not at all: os.Rename replaces its target
// in one step, so a reader sees the old file or the new one and never part of
// either.
func Write(
	path string,
	data []byte,
	perm os.FileMode,
) error {
	tmp, err := temp(path, data, perm)
	if err != nil {
		return err
	}

	defer func() { _ = os.Remove(tmp) }()

	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}

	return nil
}

// WriteNew puts data at path only if nothing is there yet.
//
// The whole file appears at once or not at all, and an existing file is left
// alone: os.Link refuses to replace its target.
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

	if err := os.Link(tmp, path); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}

	return nil
}

// temp writes data to a new file beside path and returns its name.
//
// Beside it rather than in the system's temporary directory, because a rename
// or a link only works within one filesystem. A temporary file that could not
// be written in full is removed before returning.
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

	// One check for all three: each leaves a file that is not what was asked
	// for, and each is answered the same way.
	if err := errors.Join(err, f.Chmod(perm), f.Close()); err != nil {
		_ = os.Remove(f.Name())

		return "", fmt.Errorf("writing %s: %w", path, err)
	}

	return f.Name(), nil
}
