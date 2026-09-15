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

package presets

import (
	"bytes"
	"context"
	"fmt"
	"os"

	"github.com/retr0h/tonestack/pkg/sdk/internal/atomicfile"
	"github.com/retr0h/tonestack/pkg/sdk/internal/fileslots"
	"github.com/retr0h/tonestack/pkg/sdk/preset"
	"github.com/retr0h/tonestack/pkg/sdk/result"
	"github.com/retr0h/tonestack/pkg/sdk/rig"
)

// CompileOptions says which rig to build, what to build it into, and where the
// preset goes.
type CompileOptions struct {
	// Deps are the collaborators this command works through.
	Deps

	// RigPath is the rig to read.
	RigPath string
	// TemplatePath is a preset to write the chain into. Empty uses an
	// untouched preset the device itself wrote.
	TemplatePath string
	// OutputPath is where the preset is written.
	OutputPath string
}

// Compile turns a rig into a preset a device will load.
//
// The chain is written into a preset rather than assembled from nothing. A
// device expects inputs, outputs, a split and a join around a chain, and
// 98.6% of real presets carry them; one built without them is unlike anything
// the hardware has written. A template is what makes a rig lifted off a device
// rebuild exactly.
func Compile(
	ctx context.Context,
	opts CompileOptions,
) (result.Built, error) {
	spec, err := readRig(ctx, opts.RigPath)
	if err != nil {
		return result.Built{}, err
	}

	cat, err := opts.catalog(ctx)
	if err != nil {
		return result.Built{}, err
	}

	doc, err := template(ctx, opts.TemplatePath)
	if err != nil {
		return result.Built{}, err
	}

	doc.Data.Device = cat.DeviceID
	doc.Data.Meta.Name = spec.Subject.Name

	if err := opts.compiler().Lower(doc, spec, cat); err != nil {
		return result.Built{}, err
	}

	var buf bytes.Buffer

	// A compiler can leave something in the document that does not encode,
	// and a preset file holding nothing is worse than no file.
	if err := preset.Write(&buf, doc); err != nil {
		return result.Built{}, fmt.Errorf("writing %s: %w", opts.OutputPath, err)
	}

	if err := atomicfile.Write(opts.OutputPath, buf.Bytes(), 0o600); err != nil {
		return result.Built{}, err
	}

	return result.Built{
		Name:   spec.Subject.Name,
		Blocks: len(spec.Chain),
		Path:   opts.OutputPath,
	}, nil
}

// readRig loads a rig from disk.
func readRig(
	ctx context.Context,
	path string,
) (rig.Spec, error) {
	if err := ctx.Err(); err != nil {
		return rig.Spec{}, err
	}

	f, err := os.Open(path) //nolint:gosec // the path is the user's own file
	if err != nil {
		return rig.Spec{}, fmt.Errorf("opening %s: %w", path, err)
	}

	// Opened read-only, so Close has nothing to report the read did not.
	defer func() { _ = f.Close() }()

	return rig.Load(f)
}

// template returns the preset a chain is written into.
//
// A named one is read the way a standalone preset is read anywhere else.
func template(
	ctx context.Context,
	path string,
) (*preset.Document, error) {
	if path == "" {
		return preset.Blank()
	}

	return fileslots.ReadPreset(ctx, path)
}
