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

package tools

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/retr0h/tonestack/pkg/sdk"
)

var (
	// ErrNoSource is preset_build given nothing to build from.
	ErrNoSource = errors.New("name a recipe_id or a rig_path to build from")
	// ErrTwoSources is preset_build given both.
	ErrTwoSources = errors.New("name a recipe_id or a rig_path, not both")
	// ErrNotInCatalog is corpus_model's model measured but missing from the
	// catalog it was resolved against, a sign the corpus and catalog have
	// drifted apart.
	ErrNotInCatalog = errors.New("measured but not in the catalog")
	// ErrWouldOverwrite is preset_build or preset_export pointed at a file
	// that already exists, on a server started without --allow-writes.
	ErrWouldOverwrite = errors.New(
		"a file is already there, and replacing it needs the server started with --allow-writes")
)

// mayWrite refuses a path a file already sits at, unless the server was
// started with writes allowed.
//
// An agent picks the path. One naming somebody's own preset would otherwise
// replace it without anybody having agreed to that.
//
// This look is a courtesy: it refuses before a build runs or the pedal is
// claimed. It is not what keeps the file. A file can appear between the look
// and the write, so the write is told the same thing through existing, and
// refuses whatever is there when it lands.
func (h *handlers) mayWrite(
	path string,
) error {
	if h.allowWrites {
		return nil
	}

	if _, err := os.Lstat(path); err == nil {
		return fmt.Errorf("%w: %s", ErrWouldOverwrite, path)
	}

	return nil
}

// existing is what a write does about a file already at its path: replace it
// on a server started with writes allowed, and keep it otherwise.
func (h *handlers) existing() sdk.Existing {
	if h.allowWrites {
		return sdk.ReplaceExisting
	}

	return sdk.KeepExisting
}

// refused says a write the SDK refused for a file already at path the way
// mayWrite says it, so an agent reads one refusal however late the file
// appeared. Any other error comes back as it was.
func (h *handlers) refused(
	path string,
	err error,
) error {
	if !h.allowWrites && errors.Is(err, fs.ErrExist) {
		return fmt.Errorf("%w: %s", ErrWouldOverwrite, path)
	}

	return err
}

// notInCatalog wraps ErrNotInCatalog with the model id that could not be
// found, so the agent sees which model and not just an opaque schema failure.
func notInCatalog(
	id string,
) error {
	return fmt.Errorf("%w: %s", ErrNotInCatalog, id)
}

// remedy names the tool to call next, for the errors that have one.
//
// The SDK says what went wrong and leaves the next step to its caller. A
// person at a terminal is told a command; an agent here is told a tool it can
// call. Any other error comes back as it was.
func remedy(
	err error,
) error {
	switch {
	case errors.Is(err, sdk.ErrNoSuchBlock):
		return fmt.Errorf("%w, call catalog_search to find one", err)
	case errors.Is(err, sdk.ErrNoSuchRecipe):
		return fmt.Errorf("%w, call rigs_list to see the rigs that ship", err)
	default:
		return err
	}
}
