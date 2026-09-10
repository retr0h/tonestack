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
	"strings"

	"github.com/retr0h/tonestack/pkg/rig/gen"
)

// where reads a position back as the phrase a player would use.
var where = map[gen.TechniquePosition]string{
	gen.PositionBridge: "near the bridge",
	gen.PositionMiddle: "over the middle",
	gen.PositionNeck:   "over the neck",
}

// technique writes the three things a rig stores as the one sentence a person
// would say.
//
// A rig stores them apart so that two rigs can be compared, and nobody says
// "attack: pick, position: bridge" out loud. Muting is named only when there
// is some, because "not muted" is what every unmuted note already sounds like.
func technique(t gen.Technique) string {
	parts := []string{string(t.Attack)}

	if t.Position != nil {
		parts = append(parts, where[*t.Position])
	}

	if t.Muting != nil && *t.Muting == gen.MutingPalm {
		parts = append(parts, "palm muted")
	}

	return strings.Join(parts, ", ")
}
