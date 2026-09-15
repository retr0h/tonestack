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
	"path"
	"strings"

	"sigs.k8s.io/yaml"

	"github.com/retr0h/tonestack/pkg/sdk/rig"
)

// merged is every rig asking by name can find: somebody's own, and each rig
// beneath that none of theirs replaces, in identifier order.
func (s set) merged() []stored {
	out := make([]stored, 0, len(s.user)+len(s.base))
	out = append(out, s.user...)

	for _, e := range s.base {
		if _, replaced := replacement(s.user, e.spec); !replaced {
			out = append(out, e)
		}
	}

	sortEntries(out)

	return out
}

// find picks the rig asking for id finds.
//
// One of theirs that answers to id first. Then a rig beneath, unless one of
// theirs shares a name with it, so asking for a rig by any of its names finds
// the rig a listing shows.
//
// A file of theirs that is not a rig stops a lookup only when it may be the
// rig asked for: when its filename, or the id or aliases it states, is the
// name asked for or a name of the rig found. Anything else would build a rig
// that ships while the one they wrote in its place goes unread, and say
// nothing. A broken file with no name in common stops nothing, and List
// reports it.
func (s set) find(
	id string,
) (stored, error) {
	for _, b := range s.broken {
		if answers(b.names, id) {
			return stored{}, b.err
		}
	}

	found, ok := first(s.user, id)
	if !ok {
		found, ok = first(s.base, id)
		if !ok {
			return stored{}, &NotFoundError{ID: id, Known: len(s.merged())}
		}

		if theirs, replaced := replacement(s.user, found.spec); replaced {
			found = theirs
		}
	}

	for _, b := range s.broken {
		for _, n := range names(found.spec) {
			if answers(b.names, n) {
				return stored{}, b.err
			}
		}
	}

	return found, nil
}

// first is the first entry that answers to id.
func first(
	all []stored,
	id string,
) (stored, bool) {
	for _, e := range all {
		if answers(names(e.spec), id) {
			return e, true
		}
	}

	return stored{}, false
}

// replacement is the rig of theirs standing in for one beneath.
//
// Any name in common, identifier or alias, because asking for that name would
// otherwise find one rig through a lookup and list the other.
func replacement(
	user []stored,
	beneath rig.Spec,
) (stored, bool) {
	for _, n := range names(beneath) {
		if theirs, ok := first(user, n); ok {
			return theirs, true
		}
	}

	return stored{}, false
}

// names are every way a rig can be asked for, in lower case.
func names(
	spec rig.Spec,
) []string {
	out := []string{strings.ToLower(spec.ID)}

	if spec.Aliases != nil {
		for _, a := range *spec.Aliases {
			out = append(out, strings.ToLower(a))
		}
	}

	return out
}

// answers reports whether id is one of names, whatever its case.
func answers(
	names []string,
	id string,
) bool {
	id = strings.ToLower(id)

	for _, n := range names {
		if n == id {
			return true
		}
	}

	return false
}

// claimed are the names a file that is not a rig may have been meant to
// answer to: its filename stem, which is the id recipes new writes it under,
// and whatever id and aliases the text states where it parses that far.
func claimed(
	p string,
	raw []byte,
) []string {
	out := []string{strings.ToLower(strings.TrimSuffix(path.Base(p), ".yaml"))}

	var doc map[string]any
	if yaml.Unmarshal(raw, &doc) != nil {
		return out
	}

	if id, ok := doc["id"].(string); ok {
		out = append(out, strings.ToLower(id))
	}

	aliases, _ := doc["aliases"].([]any)
	for _, a := range aliases {
		if alias, ok := a.(string); ok {
			out = append(out, strings.ToLower(alias))
		}
	}

	return out
}
