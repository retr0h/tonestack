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

package main

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/corpus"
)

// KnowledgeTestSuite holds docs/knowledge.md to the data it quotes.
//
// The page is where this project says what it knows, and it quotes figures
// from the catalog and the corpus statistics built into the binary. Those are
// regenerated when Line 6 ships a release or the corpus grows, and the page
// went stale more than once before anything checked it.
type KnowledgeTestSuite struct {
	suite.Suite
	page  string
	cat   *catalog.Catalog
	stats *corpus.Stats
}

func (s *KnowledgeTestSuite) SetupSuite() {
	raw, err := os.ReadFile("docs/knowledge.md")
	s.Require().NoError(err)

	// Prose wraps wherever the formatter puts the line end, so a figure is
	// looked for with every run of whitespace collapsed to one space.
	s.page = strings.Join(strings.Fields(string(raw)), " ")

	s.cat, err = catalog.BuiltIn()
	s.Require().NoError(err)

	s.stats, err = corpus.BuiltIn()
	s.Require().NoError(err)
}

// TestTheFiguresMatchTheData recomputes every figure the page quotes.
//
// A failure names the sentence the data now supports. Either the data changed
// and the sentence should follow it, or the page quotes something new and this
// should learn to compute it.
func (s *KnowledgeTestSuite) TestTheFiguresMatchTheData() {
	bass := s.stats.Grammar["bass"]
	comp := bass.Categories[catalog.CategoryComp]
	drive := bass.Categories[catalog.CategoryDrive]

	names, controls := s.parameters()

	unclear := 0
	for _, key := range []string{"Sag", "Hum", "Ripple", "Bias", "BiasX"} {
		unclear += names[key]
	}

	brt := s.model("HD2_AmpSVBeastBrt")
	nrm := s.model("HD2_AmpSVBeastNrm")
	nrmBlock, _ := s.cat.Block("HD2_AmpSVBeastNrm")
	brtBlock, _ := s.cat.Block("HD2_AmpSVBeastBrt")

	tests := []struct {
		name string
		want string
	}{
		{
			name: "models mapped to real gear",
			want: fmt.Sprintf("%d models", s.named()),
		},
		{
			name: "presets measured",
			want: fmt.Sprintf("Across the %s presets measured", thousands(s.stats.Presets)),
		},
		{
			name: "what a bass chain holds",
			want: fmt.Sprintf("across %d bass chains, %d%% hold a compressor and %d%% hold drive, "+
				"which sits before the amp %d%% of the time",
				bass.Chains, percent(comp.Frequency(bass.Chains)),
				percent(drive.Frequency(bass.Chains)), percent(drive.BeforeAmp())),
		},
		{
			// Not a convention: close to a coin flip, which is why a build does
			// not reorder a rig's chain by it.
			name: "a compressor's side of the amp",
			want: fmt.Sprintf("A compressor on bass sits before the amp %d%% of the time",
				percent(comp.BeforeAmp())),
		},
		{
			name: "a default players move away from",
			want: fmt.Sprintf("Across %d presets using the Ampeg SVT's bright channel, the median "+
				"`Treble` is %.2f where Line 6's stated default is %.2f",
				brt.Uses, brt.Params["Treble"].Median, s.float(brtBlock, "Treble")),
		},
		{
			name: "where players agree and where they do not",
			want: fmt.Sprintf(
				"On its normal channel `Bass` sits in %.2f–%.2f and `Drive` spans %.2f–%.2f",
				nrm.Params["Bass"].P25,
				nrm.Params["Bass"].P75,
				nrm.Params["Drive"].P25,
				nrm.Params["Drive"].P75,
			),
		},
		{
			name: "how many controls there are",
			want: fmt.Sprintf("The catalog has %d parameter names across %s controls",
				len(names), thousands(controls)),
		},
		{
			name: "the controls a name does not explain",
			want: fmt.Sprintf("about %d of them together", roundTo(unclear, 50)),
		},
		{
			// The comparison Derive makes needs artists to compare, so the
			// page says how many there are. A sixth added without the
			// sentence following it makes the page wrong about its own
			// method.
			name: "players in the music corpus",
			want: fmt.Sprintf("The music corpus names %d players", s.players()),
		},
		{
			name: "the amp the pipeline follows",
			want: fmt.Sprintf("Drive %.1f–%.1f, default %.2f, DSP %.2f",
				nrmBlock.Params["Drive"].Min, nrmBlock.Params["Drive"].Max,
				s.float(nrmBlock, "Drive"), nrmBlock.DSP.Mono),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Require().Contains(s.page, tt.want,
				"docs/knowledge.md no longer matches the embedded data")
		})
	}
}

// players counts the artists the music corpus holds records for.
//
// One directory each, with the manifest naming the tracks. The records
// themselves are not in the repository.
func (s *KnowledgeTestSuite) players() int {
	found, err := filepath.Glob(filepath.Join("resources", "music", "*", "corpus.yaml"))
	s.Require().NoError(err)

	return len(found)
}

// model reads one model's measurements, which the page's examples depend on.
func (s *KnowledgeTestSuite) model(
	id catalog.ModelID,
) corpus.ModelStats {
	ms, ok := s.stats.Models[id]
	s.Require().True(ok, "the corpus no longer measures %s", id)

	return ms
}

// float reads a parameter's stated default.
func (s *KnowledgeTestSuite) float(
	b catalog.Block,
	key string,
) float64 {
	v, ok := b.Params[key].Default.Float()
	s.Require().True(ok, "%s %s has no numeric default", b.ID, key)

	return v
}

// named counts the blocks the catalog maps to real-world gear.
func (s *KnowledgeTestSuite) named() int {
	n := 0

	for _, b := range s.cat.Blocks {
		if b.BasedOn != "" {
			n++
		}
	}

	return n
}

// parameters counts how often each parameter name occurs, and all of them.
func (s *KnowledgeTestSuite) parameters() (map[string]int, int) {
	names := map[string]int{}
	total := 0

	for _, b := range s.cat.Blocks {
		for key := range b.Params {
			names[key]++
			total++
		}
	}

	return names, total
}

// percent rounds a share to a whole percentage, the way the page quotes it.
func percent(
	share float64,
) int {
	return int(math.Round(share * 100))
}

// roundTo rounds n to the nearest multiple of step, for figures quoted as
// "about".
func roundTo(
	n, step int,
) int {
	return int(math.Round(float64(n)/float64(step))) * step
}

// thousands writes n with a comma between each group of three digits.
func thousands(
	n int,
) string {
	digits := fmt.Sprint(n)

	var b strings.Builder

	for i, d := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			b.WriteByte(',')
		}

		b.WriteRune(d)
	}

	return b.String()
}

func TestKnowledgeTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(KnowledgeTestSuite))
}
