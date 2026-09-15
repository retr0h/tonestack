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
	loader "sigs.k8s.io/yaml"
)

// lineBreak is what a parser counts as the end of a line.
//
// Counting newlines alone finds fewer lines than the parser did in a file
// written on a classic Mac, or one holding a separator a word processor left
// behind, and every line after the first of those is then a line further down
// than it looks.
var lineBreak = regexp.MustCompile(`\r\n|[\n\r\x{0085}\x{2028}\x{2029}]`)

// errNoSubjectField reports a rig to copy whose subject has no such field.
var errNoSubjectField = errors.New("the rig to copy has no such field to replace")

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

	var kindErr, nameErr error

	// Both belong to the subject, and both are replaced first, while the
	// text is still the rig that loaded: the lines removed below are only
	// removed whole where they are one line long.
	if opts.Kind != "" {
		body, kindErr = replaceSubject(body, "kind", opts.Kind)
	}

	if opts.Name != "" {
		body, nameErr = replaceSubject(body, "name", opts.Name)
	}

	return header(from, opts) + asCopy(body, from, opts), errors.Join(kindErr, nameErr)
}

// asCopy writes the copy's own identity over the parent's.
//
// The parent is copied as text rather than parsed and re-marshalled, because
// marshalling loses the comments and the comments are most of what a rig
// carries. Every citation comes across with it, which is the point and also
// the hazard: see the header the copy is given.
//
// What goes is what belongs to the parent alone: the identifier, replaced by
// the copy's own and the link back to it, the aliases and the default, which
// name the parent to a reader asking for it, and the header comments, which
// describe the parent. Each is a whole line at the top level, so each is
// matched as one. The subject's own fields are not, and are found by parsing.
//
// Not everything above `schema` is a header: a rig read off a device has its
// keys in marshalled order, and `schema` comes late.
func asCopy(
	body, from string,
	opts NewOptions,
) string {
	var out []string

	header := true

	for _, line := range breakLines(body) {
		text, ends := lineText(line)

		switch {
		case strings.HasPrefix(text, "aliases: "), strings.HasPrefix(text, "default: "):
		case strings.HasPrefix(text, "id: "):
			header = false

			out = append(out, "id: "+opts.ID+ends, "extends: "+from+ends)
		case header && (text == "" || strings.HasPrefix(text, "#")):
		default:
			header = false

			out = append(out, line)
		}
	}

	return strings.Join(out, "")
}

// lineText splits a line into what it says and what ends it.
func lineText(
	line string,
) (string, string) {
	if at := lineBreak.FindStringIndex(line); at != nil {
		return line[:at[0]], line[at[0]:]
	}

	return line, ""
}

// replaceSubject sets one of a rig's subject fields in the rig's own text and
// leaves every other byte as it was.
//
// A rig names `kind` and `name` in other places at the same indent, a
// device's name and each snapshot's among them, so the field is found by
// parsing rather than by matching a line. The value is then replaced where it
// stands, encoded as YAML so a name like `a: b` stays a name, and the result
// is decoded to prove nothing else moved.
//
// Where the value cannot be replaced in place, because it spans lines or
// carries a tag, the whole document is encoded again. That keeps the comments
// and the key order but not the layout: blank lines go, and indentation and
// flow collections take the encoder's spacing.
func replaceSubject(
	body, key, value string,
) (string, error) {
	var doc yaml.Node

	err := yaml.Unmarshal([]byte(body), &doc)
	field, flow := subjectField(&doc, key)

	// A rig that loaded has a subject carrying both fields, since the
	// contract requires them, so this is a rig that did not load.
	if err != nil || field.Kind == 0 {
		return "", fmt.Errorf("rewriting the copy's subject %s: %w", key,
			errors.Join(errNoSubjectField, err))
	}

	if spliced, ok := splice(body, field, flow, key, value); ok {
		return spliced, nil
	}

	field.Kind = yaml.ScalarNode
	field.Tag = "!!str"
	field.Value = value
	field.Style = styleFor(value, 0, false)
	field.Content = nil
	// The anchor stays with the field. Something else in the rig may be
	// written as whatever this field says, and dropping it would leave that
	// pointing at nothing.

	var out bytes.Buffer

	enc := yaml.NewEncoder(&out)
	enc.SetIndent(2)
	err = enc.Encode(&doc)

	return out.String(), errors.Join(err, enc.Close())
}

// subjectField finds the node holding one of the subject's fields, or an
// empty node, and whether it sits inside a flow collection, where a comma or
// a brace ends a bare value.
func subjectField(
	doc *yaml.Node,
	key string,
) (*yaml.Node, bool) {
	top := &yaml.Node{}
	if len(doc.Content) > 0 {
		top = doc.Content[0]
	}

	subject := valueOf(top, "subject")
	flow := top.Style&yaml.FlowStyle != 0 || subject.Style&yaml.FlowStyle != 0

	return valueOf(subject, key), flow
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
	field *yaml.Node,
	flow bool,
	key, value string,
) (string, bool) {
	lines := breakLines(body)
	// Held inside the document rather than trusted: a line counted here and
	// not by the parser, or the other way about, would put the edit on the
	// wrong line, which is what the comparison below is for.
	at := min(max(field.Line-1, 0), len(lines)-1)
	line := lines[at]

	start, end, ok := valueSpan(line, field.Column, field.Style, flow)
	if !ok {
		return "", false
	}

	encoded, err := encodeScalar(value, styleFor(value, field.Style, flow))
	lines[at] = line[:start] + encoded + line[end:]
	out := strings.Join(lines, "")

	return out, err == nil && onlyFieldChanged(body, out, key, value)
}

// breakLines splits a document where its parser ends a line, keeping the
// break on the end of the line it ends.
func breakLines(
	body string,
) []string {
	var lines []string

	at := 0

	for _, where := range lineBreak.FindAllStringIndex(body, -1) {
		lines = append(lines, body[at:where[1]])
		at = where[1]
	}

	return append(lines, body[at:])
}

// isBreak reports whether a parser ends a line at this character.
func isBreak(
	r rune,
) bool {
	return r == '\n' || r == '\r' || r == 0x0085 || r == 0x2028 || r == 0x2029
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
		if isBreak(r) ||
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

// styleFor decides how a value has to be written for the rig to read back as
// the value it is.
//
// Two YAML libraries see this file. The document is parsed and written here
// by one that reads YAML 1.2, where `Yes` is the string "Yes"; a rig is
// loaded by another that reads 1.1, where `Yes` is a boolean and a subject's
// name has to be a string. So the encoded value is put through the loader's
// own decoder and quoted when it does not come back, which keeps the two
// from disagreeing again over some other word.
func styleFor(
	value string,
	style yaml.Style,
	flow bool,
) yaml.Style {
	if strings.IndexFunc(value, isBreak) >= 0 {
		// Nothing written over several lines fits on the line it replaces.
		style = yaml.DoubleQuotedStyle
	}

	encoded, err := encodeScalar(value, style)
	if err == nil && loaderReads(encoded, flow) == value {
		return style
	}

	return yaml.DoubleQuotedStyle
}

// encodeScalar writes one value as YAML, in the style asked for.
func encodeScalar(
	value string,
	style yaml.Style,
) (string, error) {
	out, err := yaml.Marshal(&yaml.Node{
		Kind:  yaml.ScalarNode,
		Tag:   "!!str",
		Value: value,
		Style: style,
	})

	return strings.TrimSuffix(string(out), "\n"), err
}

// loaderReads is what the decoder a rig is loaded through makes of an encoded
// value, written where it is about to be written.
func loaderReads(
	encoded string,
	flow bool,
) any {
	doc := "name: " + encoded + "\n"
	if flow {
		doc = "{name: " + encoded + "}\n"
	}

	var held map[string]any

	// A value the loader cannot read at all is not the value it was asked
	// to hold, which is what the caller does with the answer.
	_ = loader.Unmarshal([]byte(doc), &held)

	return held["name"]
}

// onlyFieldChanged reports whether after loads as before does with one of the
// subject's fields set to value, and with nothing else about it changed.
//
// Read by the decoder a rig is loaded through, so that what is compared is
// what a reader of the copy will get.
func onlyFieldChanged(
	before, after, key, value string,
) bool {
	var was, now map[string]any

	errWas := loader.Unmarshal([]byte(before), &was)
	errNow := loader.Unmarshal([]byte(after), &now)

	subject, ok := was["subject"].(map[string]any)
	if ok {
		subject[key] = value
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
