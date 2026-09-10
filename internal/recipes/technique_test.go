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
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/rig/gen"
)

// TechniqueTestSuite covers writing what a rig stores as what a person says.
type TechniqueTestSuite struct {
	suite.Suite
}

// TestTechnique covers every shape a technique can take.
func (s *TechniqueTestSuite) TestTechnique() {
	at := func(p gen.TechniquePosition) *gen.TechniquePosition { return &p }
	mute := func(m gen.TechniqueMuting) *gen.TechniqueMuting { return &m }

	tests := []struct {
		name string
		in   gen.Technique
		want string
	}{
		{
			name: "attack alone, which is all a rig has to say",
			in:   gen.Technique{Attack: gen.AttackFingers},
			want: "fingers",
		},
		{
			name: "with a place along the string",
			in: gen.Technique{
				Attack: gen.AttackPick, Position: at(gen.PositionBridge),
			},
			want: "pick, near the bridge",
		},
		{
			name: "over the middle",
			in: gen.Technique{
				Attack: gen.AttackThumb, Position: at(gen.PositionMiddle),
			},
			want: "thumb, over the middle",
		},
		{
			name: "over the neck",
			in: gen.Technique{
				Attack: gen.AttackSlap, Position: at(gen.PositionNeck),
			},
			want: "slap, over the neck",
		},
		{
			name: "all three",
			in: gen.Technique{
				Attack:   gen.AttackHybrid,
				Position: at(gen.PositionBridge),
				Muting:   mute(gen.MutingPalm),
			},
			want: "hybrid, near the bridge, palm muted",
		},
		{
			// Nothing damping the string is what an unmuted note already
			// sounds like, so saying so adds a word and no information.
			name: "muting stated as none, which reads no differently",
			in: gen.Technique{
				Attack: gen.AttackPick, Muting: mute(gen.MutingNone),
			},
			want: "pick",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().Equal(tt.want, technique(tt.in))
		})
	}
}

func TestTechniqueTestSuite(t *testing.T) {
	suite.Run(t, new(TechniqueTestSuite))
}
