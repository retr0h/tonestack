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
package cli

import (
	"fmt"
	"io"

	"go.yaml.in/yaml/v3"

	"github.com/retr0h/tonestack/pkg/sdk/audio"
)

// termsHeader is what somebody has to do with the output.
const termsHeader = ` Paste under the rig of the player it names. Each term carries both sides of
 the comparison that earned it, because the gap between them is what decides
 how far the word moves a control: half of what everybody else reads is half
 a step, not a knob on its limit.

 A word here is an argument, not a verdict. Putting one in a file is still
 somebody deciding to believe it.`

// PlayerTerms writes what each player's records earned them, as character
// terms ready to paste into a rig.
//
// Players who earned nothing are written as a comment rather than left out:
// that a player's records earn no word is worth reading, and an empty
// document reads as a tool that failed.
func PlayerTerms(
	w io.Writer,
	of []audio.Player,
) error {
	doc := &yaml.Node{Kind: yaml.MappingNode, HeadComment: termsHeader}

	for _, p := range of {
		if len(p.Terms) == 0 {
			continue
		}

		doc.Content = append(doc.Content, text(p.ID), termsFor(p))
	}

	if len(doc.Content) == 0 {
		if _, err := fmt.Fprintf(w,
			"# no player's records earned a word: %s\n",
			"a term is earned by sitting clear of the others, and nobody did\n"); err != nil {
			return fmt.Errorf("writing terms: %w", err)
		}

		return nil
	}

	enc := yaml.NewEncoder(w)
	enc.SetIndent(2)

	err := enc.Encode(doc)

	// Closing finishes the stream and can fail on its own, so its error is
	// kept rather than dropped. It is only worth reporting when the encode
	// did not already fail.
	if closeErr := enc.Close(); err == nil {
		err = closeErr
	}

	if err != nil {
		return fmt.Errorf("writing terms: %w", err)
	}

	return nil
}

// termsFor is one player's earned words, as a character block.
func termsFor(
	p audio.Player,
) *yaml.Node {
	seq := &yaml.Node{Kind: yaml.SequenceNode}

	for _, t := range p.Terms {
		mine := &yaml.Node{Kind: yaml.MappingNode, Style: yaml.FlowStyle}
		mine.Content = append(mine.Content, text(t.Key), number(t.Mine))

		theirs := &yaml.Node{Kind: yaml.MappingNode, Style: yaml.FlowStyle}
		theirs.Content = append(theirs.Content, text(t.Key), number(t.Others))

		evidence := &yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{
			text("kind"), text("audio"),
			text("measured"), mine,
			text("against"), theirs,
			text("note"), folded(noteForTerm(p, t)),
			text("caveat"), folded(measuredCaveat),
		}}

		seq.Content = append(seq.Content, &yaml.Node{
			Kind: yaml.MappingNode,
			Content: []*yaml.Node{
				text("term"), text(t.Term),
				text("evidence"),
				{
					Kind: yaml.SequenceNode, Content: []*yaml.Node{evidence},
				},
			},
		})
	}

	return seq
}

// noteForTerm says what was compared, and how much of it there was.
func noteForTerm(
	p audio.Player,
	t audio.Derived,
) string {
	return fmt.Sprintf("%s across %s, against %d other players measured the same way",
		t.Why, recordsRead(p.Records), t.Of-1)
}
