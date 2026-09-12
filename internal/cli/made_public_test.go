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

package cli_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/internal/cli"
	"github.com/retr0h/tonestack/pkg/sdk"
	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/chain"
)

// stops writing after ok writes, so a report written in parts can be failed
// at each seam in turn.
type stops struct {
	ok int
	n  int
}

func (w *stops) Write(p []byte) (int, error) {
	w.n++
	if w.n > w.ok {
		return 0, errors.New("boom")
	}

	return len(p), nil
}

type MadePublicTestSuite struct {
	suite.Suite
}

func (s *MadePublicTestSuite) cat() *catalog.Catalog {
	return &catalog.Catalog{Blocks: map[catalog.ModelID]catalog.Block{
		"amp": {
			Name: "Test Amp", Category: catalog.CategoryAmp,
			BasedOn: "Ampeg SVT", DSP: catalog.DSPCost{Mono: 25},
		},
	}}
}

func (s *MadePublicTestSuite) made(mutate func(*sdk.Made)) sdk.Made {
	m := sdk.Made{
		Chain: chain.Chain{
			Name:   "Test Player",
			Blocks: []chain.Block{{Model: "amp", Enabled: true}},
		},
		Path: "/tmp/out.hlx",
	}

	if mutate != nil {
		mutate(&m)
	}

	return m
}

// TestMade covers saying what was built and what it chose.
func (s *MadePublicTestSuite) TestMade() {
	tests := []struct {
		name   string
		in     sdk.Made
		want   []string
		absent []string
	}{
		{
			name: "the chain it built",
			in:   s.made(nil),
			want: []string{
				"Test Player",
				// The real gear must be named, not only the model identifier.
				"Ampeg SVT",
				// The processor budget, because it is the constraint.
				"dsp0",
				"wrote /tmp/out.hlx",
			},
			absent: []string{"added", "no such character term"},
		},
		{
			// A recipe names an amp; a rig is several blocks. Whatever the
			// corpus contributed has to be visible before anybody plugs in.
			name: "what the corpus added unasked",
			in: s.made(func(m *sdk.Made) {
				m.Added = []sdk.Added{
					{Name: "Minotaur", Reason: "most chains have a drive", Share: 0.95},
				}
			}),
			want: []string{"added", "Minotaur", "95% of chains"},
		},
		{
			// A block substituted for gear no model emulates was never
			// counted, and "0% of chains" would read as a measurement.
			name: "a block nothing measured",
			in: s.made(func(m *sdk.Made) {
				m.Added = []sdk.Added{
					{Name: "Minotaur", Reason: "closest to a Klon Centaur"},
				}
			}),
			want:   []string{"Minotaur", "closest to a Klon Centaur"},
			absent: []string{"% of chains"},
		},
		{
			// The whole point of the vocabulary: a word a rig used, and the
			// knob it turned.
			name: "the knobs its words turned",
			in: s.made(func(m *sdk.Made) {
				m.Moved = []sdk.Moved{
					{Term: "mid-forward", Param: "Mid", From: 0.52, To: 0.60},
				}
			}),
			want: []string{"heard", "mid-forward", "Mid 0.52 to 0.60"},
		},
		{
			// Six of the ten axes are not amplifier controls. Saying so
			// beats letting somebody believe the word did something.
			name: "a word nothing acts on yet",
			in: s.made(func(m *sdk.Made) {
				m.Moved = []sdk.Moved{{Term: "glassy"}}
			}),
			want: []string{"glassy — nothing acts on this yet"},
		},
		{
			// Two words from one axis are two answers to one question.
			// Applying both lands back where it started.
			name: "two words answering for one axis",
			in: s.made(func(m *sdk.Made) {
				m.Moved = []sdk.Moved{
					{Term: "minimal-drive", Against: "drive"},
					{Term: "grit-on-attack", Against: "drive"},
				}
			}),
			want: []string{
				"minimal-drive — another term already answered for drive",
				"grit-on-attack — another term already answered for drive",
			},
		},
		{
			// A word nothing defines is said and not refused.
			name: "a rig describing itself in its own words",
			in: s.made(func(m *sdk.Made) {
				m.Unfamiliar = []sdk.Unfamiliar{
					{Term: "sounds like a wet paper bag"},
				}
			}),
			want:   []string{`no such character term "sounds like a wet paper bag"`},
			absent: []string{"did you mean"},
		},
		{
			name: "a word close to one that is defined",
			in: s.made(func(m *sdk.Made) {
				m.Unfamiliar = []sdk.Unfamiliar{
					{Term: "midforward", Near: []string{"mid-forward", "mid-scooped"}},
				}
			}),
			want: []string{"did you mean mid-forward, mid-scooped?"},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var buf bytes.Buffer

			s.Require().NoError(cli.Made(&buf, tt.in, s.cat()))

			for _, want := range tt.want {
				s.Require().Contains(buf.String(), want)
			}

			for _, gone := range tt.absent {
				s.Require().NotContains(buf.String(), gone)
			}
		})
	}
}

// TestMadeReportsAFailingWriter covers a report nobody can read.
//
// It is written in parts — a title, the chain, what was added, what was
// heard, what was not understood, and where the file went — and a writer that
// fails at any seam must be reported rather than leaving a half-written
// summary and a success. Every seam, rather than a list of indices that goes
// stale the moment a part is added.
func (s *MadePublicTestSuite) TestMadeReportsAFailingWriter() {
	full := s.made(func(m *sdk.Made) {
		m.Added = []sdk.Added{{Name: "Minotaur", Reason: "drive", Share: 0.95}}
		m.Moved = []sdk.Moved{{Term: "mid-forward", Param: "Mid", From: 0.5, To: 0.6}}
		m.Unfamiliar = []sdk.Unfamiliar{{Term: "wet paper bag"}}
	})

	// How many writes a whole report takes, found by letting one through.
	var counted bytes.Buffer

	s.Require().NoError(cli.Made(&counted, full, s.cat()))

	writes := &counting{}
	s.Require().NoError(cli.Made(writes, full, s.cat()))

	for ok := range writes.n {
		s.Run(fmt.Sprintf("after %d writes", ok), func() {
			err := cli.Made(&stops{ok: ok}, full, s.cat())

			s.Require().Error(err)
			s.Require().Contains(err.Error(), "reporting")
		})
	}
}

// counting accepts every write and says how many there were.
type counting struct{ n int }

func (w *counting) Write(p []byte) (int, error) {
	w.n++

	return len(p), nil
}

var _ io.Writer = (*stops)(nil)

func TestMadePublicTestSuite(t *testing.T) {
	suite.Run(t, new(MadePublicTestSuite))
}
