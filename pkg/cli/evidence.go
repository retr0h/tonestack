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
	"strconv"

	"go.yaml.in/yaml/v3"

	"github.com/retr0h/tonestack/pkg/sdk/audio"
)

// measuredCaveat is what a measurement taken off a record does not show.
//
// Carried on every entry rather than left to the person pasting it. The
// figures are a finished record, and a record is not an amplifier: forgetting
// that is how a measurement gets read as a knob position.
const measuredCaveat = "measures the record rather than the player: the amp, " +
	"the mic, the desk and the master are all in these numbers"

// evidenceHeader is what somebody has to do with the output.
const evidenceHeader = ` Paste under a chain entry's evidence:, and add a url: to each naming the
 recording it came from. A measurement nobody can check is a number without
 a source, and the url is what lets somebody who does not own the record
 disagree with it.`

// Evidence writes measurements as rig evidence, ready to paste into a chain.
//
// One entry per recording rather than one for the set. Evidence is attached
// per claim and a url is what makes a claim checkable, so a single entry
// averaging four records would be the one thing nobody could check.
func Evidence(
	w io.Writer,
	of []audio.Named,
) error {
	seq := &yaml.Node{
		Kind:        yaml.SequenceNode,
		HeadComment: evidenceHeader,
	}

	for _, n := range of {
		seq.Content = append(seq.Content, entryFor(n))
	}

	enc := yaml.NewEncoder(w)
	enc.SetIndent(2)

	err := enc.Encode(seq)

	// Closing finishes the stream and can fail on its own, so its error is
	// kept rather than dropped. It is only worth reporting when the encode
	// did not already fail: a writer that has stopped accepting bytes fails
	// both, and the first failure is the one that says what happened.
	if closeErr := enc.Close(); err == nil {
		err = closeErr
	}

	if err != nil {
		return fmt.Errorf("writing evidence: %w", err)
	}

	return nil
}

// entryFor is one recording as a piece of evidence.
func entryFor(
	n audio.Named,
) *yaml.Node {
	measured := &yaml.Node{Kind: yaml.MappingNode}
	figures := n.Profile.Measured()

	for _, key := range audio.MeasuredKeys() {
		measured.Content = append(measured.Content, text(key), number(figures[key]))
	}

	return &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			text("kind"), text("audio"),
			text("note"), text(n.Name + ", bass isolated from the mix before measuring"),
			text("caveat"), folded(measuredCaveat),
			text("measured"), measured,
		},
	}
}

// text is one string, written plainly.
func text(
	v string,
) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: v}
}

// folded is one string written over as many lines as it needs.
func folded(
	v string,
) *yaml.Node {
	return &yaml.Node{
		Kind:  yaml.ScalarNode,
		Tag:   "!!str",
		Value: v,
		Style: yaml.FoldedStyle,
	}
}

// number is one measurement, written without a trailing zero it did not earn.
//
// No tag, so YAML decides what it is looking at. Tagging these !!float writes
// `centroid: !!float 151` for every figure that lands on a whole number,
// which is noise in a file somebody is about to paste into a rig. A whole
// number is a valid JSON number, and the contract asks for a number.
func number(
	v float64,
) *yaml.Node {
	return &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: strconv.FormatFloat(v, 'f', -1, 64),
	}
}
