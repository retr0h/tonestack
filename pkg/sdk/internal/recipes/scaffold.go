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

package recipes

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/retr0h/tonestack/pkg/sdk/rigs"
)

// idLine, subjectKind and subjectName find the lines a copy has to change.
//
// The parent is copied as text rather than parsed and re-marshalled, because
// marshalling loses the comments and the comments are most of what a rig
// carries. Every citation comes across with it, which is the point and also
// the hazard: see the header the copy is given.
var (
	idLine      = regexp.MustCompile(`(?m)^id: .*$`)
	aliasesLine = regexp.MustCompile(`(?m)^aliases: .*\n`)
	defaultLine = regexp.MustCompile(`(?m)^default: .*\n`)
	subjectKind = regexp.MustCompile(`(?m)^  kind: .*$`)
	subjectName = regexp.MustCompile(`(?m)^  name: .*$`)
)

// scaffold writes a copy of one rig as the start of another.
//
// `extends` records that the two are related and nothing merges them: the
// copy is a whole rig and reads as one. That is deliberate. Inheritance would
// mean the file on disk is not the rig that compiles, and this format is
// meant to be read.
func scaffold(parent, from string, opts NewOptions) string {
	body := parent

	body = aliasesLine.ReplaceAllString(body, "")
	body = defaultLine.ReplaceAllString(body, "")
	body = idLine.ReplaceAllString(body,
		fmt.Sprintf("id: %s\nextends: %s", opts.ID, from))

	if opts.Kind != "" {
		body = subjectKind.ReplaceAllString(body, "  kind: "+opts.Kind)
	}

	if opts.Name != "" {
		body = subjectName.ReplaceAllString(body, "  name: "+opts.Name)
	}

	// Everything above the first key is the parent's own header, and it
	// describes the parent.
	if at := strings.Index(body, "schema: RigSpec"); at > 0 {
		body = body[at:]
	}

	return header(from, opts) + body
}

// header says what a reader of the copy needs to know before believing it.
func header(from string, opts NewOptions) string {
	var b strings.Builder

	fmt.Fprintf(&b, "# %s\n#\n", opts.ID)
	fmt.Fprintf(&b, "# Copied from %s, which is what `extends` above records. "+
		"Nothing\n", from)
	b.WriteString("# merges the two: this is a whole rig and reads as one, " +
		"and editing it\n# does not touch the rig it came from.\n#\n")
	b.WriteString("# Every citation below came across with the copy, " +
		"including the ones for\n# gear and technique you are about to " +
		"change. A claim about a different\n# rig is not evidence for this " +
		"one. Re-check what you edit, and drop the\n# evidence you cannot " +
		"stand behind.\n\n")

	return b.String()
}

// findFile returns the text of one rig, from a directory or from the binary.
//
// The text rather than the decoded rig, because a copy keeps the comments and
// decoding drops them.
func findFile(dir, id string) (string, string, error) {
	// Somebody's own directory first, then the ones in the binary. Copying a
	// shipped rig into a directory of your own is the common case, and it
	// would not work if the parent had to live beside the copy.
	sources := []fs.FS{rigs.FS}
	if dir != "" {
		sources = []fs.FS{os.DirFS(dir), rigs.FS}
	}

	seen := 0

	for _, fsys := range sources {
		// The pattern is a constant, so it cannot be malformed.
		paths, _ := fs.Glob(fsys, path.Join(".", "*", "*.yaml"))

		for _, p := range paths {
			raw, err := fs.ReadFile(fsys, p)
			if err != nil {
				return "", "", fmt.Errorf("opening %s: %w", path.Base(p), err)
			}

			spec, err := decode(raw, p)
			if err != nil {
				return "", "", err
			}

			seen++

			if strings.EqualFold(spec.ID, id) || matchesAlias(spec, id) {
				// The rig's own identifier, not whatever was typed. An alias
				// belongs to the parent and `extends` is matched against an
				// id, so recording the alias would leave a link that never
				// resolves.
				return string(raw), spec.ID, nil
			}
		}
	}

	// The same complaint a lookup gives, so a mistyped parent reads like a
	// mistyped recipe. Counted while walking rather than by loading
	// everything a second time.
	return "", "", &NotFoundError{ID: id, Known: seen}
}
