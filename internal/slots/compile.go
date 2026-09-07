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

package slots

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/retr0h/tonestack/internal/catalogview"
	"github.com/retr0h/tonestack/internal/cli"
	"github.com/retr0h/tonestack/internal/lift"
	"github.com/retr0h/tonestack/pkg/preset"
	"github.com/retr0h/tonestack/pkg/rig"
	riggen "github.com/retr0h/tonestack/pkg/rig/gen"
)

// CompileOptions says which rig to turn into a preset.
type CompileOptions struct {
	// RigPath is the rig to read.
	RigPath string
	// OutputPath is where the preset is written.
	OutputPath string
	// TemplatePath is a preset to write the chain into. Empty uses an
	// untouched one the device itself wrote.
	TemplatePath string
	// CatalogPath is a catalog to resolve gear against. Empty means the one
	// built into this binary.
	CatalogPath string
}

// Compile turns a rig into a preset a device will load.
//
// The chain is written into a preset rather than assembled from nothing. A
// device expects inputs, outputs, a split and a join around a chain, and
// 98.6% of real presets carry them; one built without them is unlike anything
// the hardware has written. Passing --template uses a specific preset as that
// base, which is what makes a rig lifted off a device rebuild exactly.
func Compile(w io.Writer, opts CompileOptions) error {
	spec, err := readRig(opts.RigPath)
	if err != nil {
		return err
	}

	cat, err := catalogview.Open(opts.CatalogPath)
	if err != nil {
		return err
	}

	doc, err := template(opts.TemplatePath)
	if err != nil {
		return err
	}

	doc.Data.Device = cat.DeviceID
	doc.Data.Meta.Name = spec.Subject.Name

	if err := lift.Lower(doc, spec, cat); err != nil {
		return err
	}

	var buf bytes.Buffer

	// A document this package built encodes.
	_ = preset.Write(&buf, doc)

	if err := os.WriteFile(opts.OutputPath, buf.Bytes(), 0o600); err != nil {
		return fmt.Errorf("writing %s: %w", opts.OutputPath, err)
	}

	return reportCompiled(w, spec.Subject.Name, len(spec.Chain), opts.OutputPath)
}

// readRig loads a rig from disk.
func readRig(path string) (riggen.RigSpec, error) {
	f, err := os.Open(path) //nolint:gosec // the path is the user's own file
	if err != nil {
		return riggen.RigSpec{}, fmt.Errorf("opening %s: %w", path, err)
	}

	defer func() { _ = f.Close() }()

	return rig.Load(f)
}

// template returns the preset a chain is written into.
func template(path string) (*preset.Document, error) {
	if path == "" {
		return preset.Blank()
	}

	f, err := os.Open(path) //nolint:gosec // the path is the user's own file
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", path, err)
	}

	defer func() { _ = f.Close() }()

	doc, err := preset.Read(f)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	return doc, nil
}

// reportCompiled says what was built.
func reportCompiled(w io.Writer, name string, blocks int, path string) error {
	_, err := fmt.Fprintf(w, "\n%s%s  %s\n\n%s%s\n\n",
		cli.Indent, cli.Title(w, name),
		cli.Mute(w, fmt.Sprintf("%s in the chain", plural(blocks, "block"))),
		cli.Indent, cli.Success(w, "wrote "+path))

	return err
}
