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
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"go.yaml.in/yaml/v3"
)

// idLine, aliasesLine, defaultLine and subjectKind find the lines a copy has
// to change.
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
	// firstKey is the first line that is neither blank nor a comment.
	firstKey = regexp.MustCompile(`(?m)^[^#\s]`)
)

// errNoSubjectName reports a rig to copy with no subject name to replace.
var errNoSubjectName = errors.New("the rig to copy has no subject name to replace")

// scaffold writes a copy of one rig as the start of another.
//
// `extends` records that the two are related and nothing merges them: the
// copy is a whole rig and reads as one. That is deliberate. Inheritance would
// mean the file on disk is not the rig that compiles, and this format is
// meant to be read.
//
// The text is not a copy when the error is not nil.
func scaffold(
	parent, from string,
	opts NewOptions,
) (string, error) {
	body := parent

	var err error

	// First, while the text is still the rig that loaded: the lines removed
	// below are only removed whole where they are one line long.
	if opts.Name != "" {
		body, err = rename(body, opts.Name)
	}

	body = aliasesLine.ReplaceAllString(body, "")
	body = defaultLine.ReplaceAllString(body, "")
	body = idLine.ReplaceAllString(body,
		fmt.Sprintf("id: %s\nextends: %s", opts.ID, from))

	if opts.Kind != "" {
		body = subjectKind.ReplaceAllString(body, "  kind: "+opts.Kind)
	}

	// The comments above the first key are the parent's own header, and they
	// describe the parent. Not everything above `schema`: a rig read off a
	// device has its keys in marshalled order, and `schema` comes late.
	if at := firstKey.FindStringIndex(body); at != nil {
		body = body[at[0]:]
	}

	return header(from, opts) + body, err
}

// rename sets the subject's name in a rig's text and leaves every other byte
// as it was.
//
// A rig has other `name:` keys, a device's and each snapshot's among them,
// so the name is found by parsing rather than by matching a line. The value
// is then replaced where it stands, encoded as YAML so a name like `a: b`
// stays a name, and the result is decoded to prove nothing else moved.
//
// Where the value cannot be replaced in place, because it spans lines or
// carries an anchor or a tag, the whole document is encoded again. That keeps
// the comments and the key order but not the layout: blank lines go, and
// indentation and flow collections take the encoder's spacing.
func rename(
	body, name string,
) (string, error) {
	var doc yaml.Node

	err := yaml.Unmarshal([]byte(body), &doc)
	value, flow := subjectName(&doc)

	// A rig that loaded has a subject with a name, since the contract
	// requires one, so this is a rig that did not load.
	if err != nil || value.Kind == 0 {
		return "", fmt.Errorf("renaming the copy: %w", errors.Join(errNoSubjectName, err))
	}

	if spliced, ok := splice(body, value, flow, name); ok {
		return spliced, nil
	}

	*value = yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: name}

	var out bytes.Buffer

	enc := yaml.NewEncoder(&out)
	enc.SetIndent(2)
	err = enc.Encode(&doc)

	return out.String(), errors.Join(err, enc.Close())
}

// subjectName finds the node holding the subject's name, or an empty node,
// and whether it sits inside a flow collection, where a comma or a brace ends
// a bare value.
func subjectName(
	doc *yaml.Node,
) (*yaml.Node, bool) {
	top := &yaml.Node{}
	if len(doc.Content) > 0 {
		top = doc.Content[0]
	}

	subject := valueOf(top, "subject")
	flow := top.Style&yaml.FlowStyle != 0 || subject.Style&yaml.FlowStyle != 0

	return valueOf(subject, "name"), flow
}

// valueOf returns the value a mapping holds under key, or an empty node.
func valueOf(
	mapping *yaml.Node,
	key string,
) *yaml.Node {
	found := &yaml.Node{}

	for i := 1; i < len(mapping.Content); i += 2 {
		if mapping.Kind == yaml.MappingNode && mapping.Content[i-1].Value == key {
			found = mapping.Content[i]
		}
	}

	return found
}

// splice replaces one value in the text it was parsed from, and reports
// whether the text that results is the same document with only that value
// changed.
func splice(
	body string,
	value *yaml.Node,
	flow bool,
	name string,
) (string, bool) {
	lines := strings.SplitAfter(body, "\n")
	line := lines[value.Line-1]

	start, end, ok := valueSpan(line, value.Column, value.Style, flow)
	if !ok {
		return "", false
	}

	encoded, err := encodeName(name, value.Style, flow)
	lines[value.Line-1] = line[:start] + encoded + line[end:]
	out := strings.Join(lines, "")

	return out, err == nil && onlyNameChanged(body, out, name)
}

// valueSpan finds where a scalar written on one line starts and ends, as byte
// offsets into that line. column is the parser's, counted in characters from
// one.
func valueSpan(
	line string,
	column int,
	style yaml.Style,
	flow bool,
) (int, int, bool) {
	start := len(line)

	for i := range line {
		if column == 1 {
			start = i

			break
		}

		column--
	}

	rest := line[start:]

	switch {
	case style&yaml.DoubleQuotedStyle != 0:
		return quotedSpan(rest, start, '"')
	case style&yaml.SingleQuotedStyle != 0:
		return quotedSpan(rest, start, '\'')
	case style != 0, rest == "", strings.ContainsAny(rest[:1], "&!*"):
		// A block scalar, or an anchor, tag or alias in front of the value.
		return 0, 0, false
	}

	end := len(rest)

	for i, r := range rest {
		if r == '\n' || r == '\r' ||
			(r == '#' && i > 0 && (rest[i-1] == ' ' || rest[i-1] == '\t')) ||
			(flow && strings.ContainsRune(",[]{}", r)) {
			end = i

			break
		}
	}

	return start, start + len(strings.TrimRight(rest[:end], " \t")), true
}

// quotedSpan finds the end of a quoted scalar opening at the start of rest.
func quotedSpan(
	rest string,
	start int,
	quote byte,
) (int, int, bool) {
	for i := 1; i < len(rest); i++ {
		switch {
		case quote == '"' && rest[i] == '\\':
			i++
		case rest[i] == quote && quote == '\'' && i+1 < len(rest) && rest[i+1] == '\'':
			i++
		case rest[i] == quote:
			return start, start + i + 1, true
		}
	}

	// The closing quote is on a later line.
	return 0, 0, false
}

// encodeName writes a name as a YAML scalar that fits on the line it replaces.
//
// It keeps the quoting the parent chose where that can hold the name, and
// double quotes it where nothing else stays on one line or survives inside a
// flow collection.
func encodeName(
	name string,
	style yaml.Style,
	flow bool,
) (string, error) {
	if flow || strings.ContainsAny(name, "\n\r") {
		style = yaml.DoubleQuotedStyle
	}

	out, err := yaml.Marshal(&yaml.Node{
		Kind:  yaml.ScalarNode,
		Tag:   "!!str",
		Value: name,
		Style: style,
	})

	return strings.TrimSuffix(string(out), "\n"), err
}

// onlyNameChanged reports whether after decodes to before with the subject's
// name set to name, and to nothing else.
func onlyNameChanged(
	before, after, name string,
) bool {
	var was, now map[string]any

	errWas := yaml.Unmarshal([]byte(before), &was)
	errNow := yaml.Unmarshal([]byte(after), &now)

	subject, ok := was["subject"].(map[string]any)
	if ok {
		subject["name"] = name
	}

	return errWas == nil && errNow == nil && ok && reflect.DeepEqual(was, now)
}

// header says what a reader of the copy needs to know before believing it.
func header(
	from string,
	opts NewOptions,
) string {
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

// findFile returns one rig as read, with its text, found the way a lookup
// finds it.
//
// Somebody's own directory over the rigs beneath it. Copying a shipped rig
// into a directory of your own is the common case, and it would not work if
// the parent had to live beside the copy. The text rather than only the
// decoded rig, because a copy keeps the comments and decoding drops them.
func findFile(
	src Source,
	id string,
) (stored, error) {
	all, err := read(src)
	if err != nil {
		return stored{}, err
	}

	return all.find(id)
}
