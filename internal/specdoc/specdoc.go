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

// Package specdoc renders the RigSpec contract as a page somebody can read.
//
// The contract is the only description of what a rig may say, and OpenAPI is
// not a thing people read. A hand-written reference drifts: docs/recipes.md
// never mentioned thirteen fields, one of them the field this page was written
// because somebody asked about. So this is generated, and a test fails when
// the page and the contract disagree.
package specdoc

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// Bucket is how a field's values are constrained.
//
// Every field is in one of these, and saying which is most of what a reader
// needs: a closed set can be listed, a looked-up one cannot be listed here at
// all, and an open one is prose nothing checks.
type Bucket string

// The four buckets, in the order the page explains them.
const (
	// Closed is an enumeration in the contract.
	Closed Bucket = "closed"
	// LookedUp is checked against the catalog in hand.
	LookedUp Bucket = "looked up"
	// Shaped is checked against a pattern.
	Shaped Bucket = "shaped"
	// Open is prose, and nothing parses it.
	Open Bucket = "open"
)

// Field is one field of one object, as the page shows it.
type Field struct {
	// Object is the schema it belongs to, e.g. "RigSpec".
	Object string
	// Name is the field, e.g. "technique".
	Name string
	// Type is what it holds, e.g. "string" or "list of ChainEntry".
	Type string
	// Bucket is how its values are constrained.
	Bucket Bucket
	// Allowed is the enumeration, the pattern, or where the values come
	// from. Empty for an open field.
	Allowed string
	// Required says the object cannot be written without it.
	Required bool
	// About is the first sentence of the contract's own description.
	About string
}

// Render writes the contract out as markdown.
func Render(schema []byte) ([]byte, error) {
	doc, err := openapi3.NewLoader().LoadFromData(schema)
	if err != nil {
		return nil, fmt.Errorf("reading the RigSpec schema: %w", err)
	}

	if doc.Components == nil || doc.Components.Schemas == nil {
		return nil, fmt.Errorf("%w: it describes no schemas", errNoSchema)
	}

	var out bytes.Buffer

	out.WriteString(preamble)

	for _, name := range objects(doc.Components.Schemas) {
		s := doc.Components.Schemas[name].Value

		fields := fieldsOf(name, s)

		fmt.Fprintf(&out, "\n## %s\n\n", name)

		if about := first(s.Description); about != "" {
			fmt.Fprintf(&out, "%s\n\n", about)
		}

		fmt.Fprintln(&out, "| field | holds | grammar | allowed |")
		fmt.Fprintln(&out, "| --- | --- | --- | --- |")

		for _, f := range fields {
			name := f.Name
			if f.Required {
				name += " *"
			}

			fmt.Fprintf(&out, "| `%s` | %s | %s | %s |\n",
				name, f.Type, cell(string(f.Bucket)), cell(f.Allowed))
		}
	}

	return out.Bytes(), nil
}

// objects lists the schemas that describe an object, in the order they read.
//
// RigSpec first, because that is the document; the rest alphabetically, since
// no other order means anything to somebody looking a field up.
func objects(all openapi3.Schemas) []string {
	var names []string

	for name, ref := range all {
		if ref.Value == nil || len(ref.Value.Properties) == 0 {
			continue
		}

		names = append(names, name)
	}

	sort.Strings(names)

	for i, name := range names {
		if name == "RigSpec" {
			names = append([]string{name}, append(names[:i], names[i+1:]...)...)

			break
		}
	}

	return names
}

// fieldsOf reads one object's fields, in the order the contract lists them.
func fieldsOf(object string, s *openapi3.Schema) []Field {
	required := make(map[string]bool, len(s.Required))
	for _, name := range s.Required {
		required[name] = true
	}

	names := make([]string, 0, len(s.Properties))
	for name := range s.Properties {
		names = append(names, name)
	}

	sort.Strings(names)

	out := make([]Field, 0, len(names))

	for _, name := range names {
		bucket, allowed := grammarOf(s.Properties[name])

		out = append(out, Field{
			Object:   object,
			Name:     name,
			Type:     typeOf(s.Properties[name]),
			Bucket:   bucket,
			Allowed:  allowed,
			Required: required[name],
			About:    first(s.Properties[name].Value.Description),
		})
	}

	return out
}

// grammarOf says which bucket a field is in, and what it allows.
//
// A field that holds another object is in no bucket: the question moves to
// that object's own table, and the cell points at it. Only the fields
// somebody types a value into have a grammar.
func grammarOf(ref *openapi3.SchemaRef) (Bucket, string) {
	s := ref.Value

	if len(s.Enum) > 0 {
		return Closed, list(s.Enum)
	}

	// A reference either names a vocabulary used more than once, or another
	// object with a table of its own.
	if ref.Ref != "" {
		return "", link(refName(ref.Ref))
	}

	if where, ok := s.Extensions["x-lookup"].(string); ok {
		return LookedUp, where
	}

	if s.Pattern != "" {
		return Shaped, "`" + s.Pattern + "`"
	}

	if s.Type.Is("array") && s.Items != nil {
		return grammarOf(s.Items)
	}

	switch {
	case s.Type.Is("boolean"):
		return "", "`true` or `false`"
	case s.Type.Is("integer"), s.Type.Is("number"):
		// A number nobody bounded has no grammar to state.
		if held := bounds(s); held != "" {
			return Shaped, held
		}

		return "", ""
	case s.Type.Is("object"):
		return "", ""
	}

	return Open, ""
}

// bounds renders the range a number is held to.
func bounds(s *openapi3.Schema) string {
	switch {
	case s.Min != nil && s.Max != nil:
		return fmt.Sprintf("`%g` to `%g`", *s.Min, *s.Max)
	case s.Min != nil:
		return fmt.Sprintf("`%g` or more", *s.Min)
	case s.Max != nil:
		return fmt.Sprintf("`%g` or less", *s.Max)
	default:
		return ""
	}
}

// link points at another object's table on this page.
func link(name string) string {
	return fmt.Sprintf("[%s](#%s)", name, strings.ToLower(name))
}

// typeOf names what a field holds, in words rather than in JSON Schema's.
func typeOf(ref *openapi3.SchemaRef) string {
	if ref.Ref != "" {
		return refName(ref.Ref)
	}

	s := ref.Value

	switch {
	case s.Type.Is("array") && s.Items != nil:
		return "list of " + typeOf(s.Items)
	case s.Type.Is("object") && s.AdditionalProperties.Schema != nil:
		return "map of " + typeOf(s.AdditionalProperties.Schema)
	case s.Type.Is("object"):
		return "object"
	case len(s.Type.Slice()) > 0:
		return s.Type.Slice()[0]
	default:
		return "any"
	}
}

// refName is the schema a reference points at.
func refName(ref string) string {
	at := strings.LastIndex(ref, "/")

	return ref[at+1:]
}

// list renders an enumeration.
func list(all []any) string {
	out := make([]string, 0, len(all))
	for _, v := range all {
		out = append(out, fmt.Sprintf("`%v`", v))
	}

	return strings.Join(out, ", ")
}

// cell keeps a table cell from breaking the row it is in.
func cell(s string) string {
	if s == "" {
		return "—"
	}

	return strings.ReplaceAll(strings.ReplaceAll(s, "|", "\\|"), "\n", " ")
}

// first returns the opening sentence of a description.
func first(s string) string {
	s = strings.TrimSpace(s)

	if at := strings.Index(s, "\n\n"); at >= 0 {
		s = s[:at]
	}

	return strings.Join(strings.Fields(s), " ")
}
