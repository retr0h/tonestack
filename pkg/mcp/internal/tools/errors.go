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
)

// notInCatalog wraps ErrNotInCatalog with the model id that could not be
// found, so the agent sees which model and not just an opaque schema failure.
func notInCatalog(id string) error {
	return fmt.Errorf("%w: %s", ErrNotInCatalog, id)
}
