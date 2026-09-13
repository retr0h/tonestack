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

// Command datagen refreshes the corpus statistics this binary embeds.
//
// Run by `just generate` through the directive in generate.go, after the
// catalog's, so the presets are measured against the catalog as it now is. A
// machine without the corpus skips it, and a machine with it writes the
// statistics only when they changed.
package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/retr0h/tonestack/pkg/sdk/internal/corpusgen"
)

// root is the repository, worked out from this file rather than from wherever
// somebody ran the command.
func root() (string, error) {
	_, self, _, ok := runtime.Caller(0)
	if !ok {
		return "", errors.New("cannot tell where this generator lives")
	}

	// pkg/sdk/internal/corpusgen/datagen/main.go: five directories above the
	// one this file is in.
	dir := self
	for range 6 {
		dir = filepath.Dir(dir)
	}

	return dir, nil
}

func main() {
	dir, err := root()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	r, err := corpusgen.Refresh(corpusgen.Options{
		CorpusDir:  filepath.Join(dir, "resources", "schemas", "corpus"),
		OutputPath: filepath.Join(dir, "pkg", "sdk", "corpus", "data", "hx-stomp.stats.json.gz"),
	})

	switch {
	case err != nil:
		fmt.Fprintln(os.Stderr, "corpus:", err)
		os.Exit(1)
	case r.Skipped != "":
		fmt.Println("corpus: skipped,", r.Skipped)
	case !r.Changed:
		fmt.Printf("corpus: unchanged, %d presets measured\n", r.Stats.Presets)
	default:
		fmt.Printf("corpus: wrote statistics from %d presets, %d models\n",
			r.Stats.Presets, len(r.Stats.Models))
	}
}
