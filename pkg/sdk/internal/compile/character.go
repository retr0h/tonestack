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

package compile

import (
	"encoding/json"
	"sort"
	"strings"
	"unicode"

	riggen "github.com/retr0h/tonestack/pkg/sdk/internal/gen"
)

// UnknownTerm is a word a rig used to describe its sound that the shipped
// vocabulary does not carry.
//
// Reported rather than refused. Nothing compiles a character term into a
// chain, so an unfamiliar one costs nothing and stops nothing, and refusing a
// preset over a word that moves no knob would make the format hostile to the
// person it exists for. Unknown gear is different: there is no model to
// write, so that is an error and stays one.
type UnknownTerm struct {
	// Term is what the rig said.
	Term string
	// Near are terms the vocabulary does carry that look close.
	Near []string
}

// vocabulary is the shipped term list, keyed by axis.
type vocabulary struct {
	Axes map[string]struct {
		About string            `json:"about"`
		Terms map[string]string `json:"terms"`
	} `json:"axes"`
}

// CharacterTerms returns the terms the vocabulary knows, in order.
//
// Exported because the words are the point: a person writing a rig needs to
// see the list, and a test over the rigs this project ships needs to check
// against it.
func CharacterTerms() []string {
	var v vocabulary

	// Embedded and written by this repository, so it parses.
	_ = json.Unmarshal(terms, &v)

	out := []string(nil)

	for _, axis := range v.Axes {
		for term := range axis.Terms {
			out = append(out, term)
		}
	}

	sort.Strings(out)

	return out
}

// CheckCharacter reports the character terms a rig uses that nothing defines.
//
// An empty result means every word in the rig is one the vocabulary carries.
func CheckCharacter(spec riggen.RigSpec) []UnknownTerm {
	if spec.Character == nil {
		return nil
	}

	known := CharacterTerms()
	out := []UnknownTerm(nil)

	for _, c := range *spec.Character {
		if has(known, c.Term) {
			continue
		}

		out = append(out, UnknownTerm{Term: c.Term, Near: closest(known, c.Term)})
	}

	return out
}

// closest returns the terms sharing the most words with what was written.
//
// Word overlap rather than the prefix match a gear name gets. What people
// write here is a phrase in some order, and "pick attack audible" should
// reach "audible-pick-attack" even though neither is a prefix or a substring
// of the other.
//
// The two are not one function because they are not one problem. near, in
// check.go, finds a name somebody typed part of. This finds a term inside a
// sentence somebody wrote. Word overlap cannot reach "Ampeg SVT" from
// "Ampeg", and a prefix cannot reach a phrase whose words are reordered.
func closest(known []string, term string) []string {
	want := words(term)
	if len(want) == 0 {
		return nil
	}

	best, hits := 0, make([]string, 0, len(known))

	for _, k := range known {
		shared := 0

		for w := range words(k) {
			if want[w] {
				shared++
			}
		}

		switch {
		case shared > best:
			// A better match replaces every worse one, reusing the room
			// already taken rather than starting a new slice each time.
			best, hits = shared, append(hits[:0], k)
		case shared == best && shared > 0:
			hits = append(hits, k)
		}
	}

	if best == 0 {
		return nil
	}

	return hits
}

// words splits a term into the words it is made of, however it was spelled.
func words(s string) map[string]bool {
	out := map[string]bool{}

	for _, w := range strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}) {
		out[w] = true
	}

	return out
}

// has says whether the vocabulary carries a term.
func has(known []string, term string) bool {
	for _, k := range known {
		if k == term {
			return true
		}
	}

	return false
}
