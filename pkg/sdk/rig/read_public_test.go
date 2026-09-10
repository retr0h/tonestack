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

package rig_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/sdk/rig"
	"github.com/retr0h/tonestack/pkg/sdk/rig/gen"
)

type ReadPublicTestSuite struct {
	suite.Suite
}

// spec returns a rig holding the given chain, with nothing else filled in.
func spec(chain ...gen.ChainEntry) gen.RigSpec {
	return gen.RigSpec{
		Schema:     gen.RigSpecSchemaRigSpec,
		ID:         "test",
		Subject:    gen.Subject{Kind: gen.KindArtist, Name: "Test"},
		Instrument: gen.InstrumentBass,
		Chain:      chain,
	}
}

// evidence returns the kinds as a rig carries them.
func evidence(kinds ...gen.EvidenceKind) *[]gen.Evidence {
	out := make([]gen.Evidence, 0, len(kinds))
	for _, k := range kinds {
		out = append(out, gen.Evidence{Kind: k})
	}

	return &out
}

// TestGear looks a role up in a chain.
func (s *ReadPublicTestSuite) TestGear() {
	tests := []struct {
		name  string
		chain []gen.ChainEntry
		role  gen.Role
		want  string
		ok    bool
	}{
		{
			name: "a role the chain has",
			chain: []gen.ChainEntry{
				{Role: gen.RoleDrive, Gear: "Klon Centaur"},
				{Role: gen.RoleAmp, Gear: "Ampeg SVT"},
			},
			role: gen.RoleAmp,
			want: "Ampeg SVT",
			ok:   true,
		},
		{
			name:  "a role it does not",
			chain: []gen.ChainEntry{{Role: gen.RoleAmp}},
			role:  gen.RoleCab,
		},
		{
			name: "an empty chain",
			role: gen.RoleAmp,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, ok := rig.Gear(spec(tt.chain...), tt.role)

			s.Require().Equal(tt.ok, ok)
			s.Require().Equal(tt.want, got.Gear)

			// GearName is the same lookup with the miss spelled as an empty
			// name, which is what a template wants.
			s.Require().Equal(tt.want, rig.GearName(spec(tt.chain...), tt.role))
		})
	}
}

// TestTrusted says whether every claim in a rig is supported.
func (s *ReadPublicTestSuite) TestTrusted() {
	tests := []struct {
		name  string
		chain []gen.ChainEntry
		rig   []gen.EvidenceKind
		want  bool
	}{
		{
			name:  "a claim nobody supported",
			chain: []gen.ChainEntry{{Role: gen.RoleAmp}},
		},
		{
			// `llm` means a model said so and nobody checked, which is the
			// same standing as nobody having said anything.
			name: "an assertion alone",
			chain: []gen.ChainEntry{
				{Role: gen.RoleAmp, Evidence: evidence(gen.EvidenceLLM)},
			},
		},
		{
			name: "one unsupported claim among supported ones",
			chain: []gen.ChainEntry{
				{Role: gen.RoleAmp, Evidence: evidence(gen.EvidenceCited)},
				{Role: gen.RoleCab, Evidence: evidence(gen.EvidenceLLM)},
			},
		},
		{
			name: "every claim supported",
			chain: []gen.ChainEntry{
				{Role: gen.RoleAmp, Evidence: evidence(gen.EvidenceCited)},
				{Role: gen.RoleCab, Evidence: evidence(gen.EvidenceVideo)},
			},
			want: true,
		},
		{
			// A rig rundown covers every piece of gear in it. Requiring the
			// citation on each entry would only encourage repeating it.
			name:  "evidence on the rig, covering the whole chain",
			chain: []gen.ChainEntry{{Role: gen.RoleAmp}},
			rig:   []gen.EvidenceKind{gen.EvidenceCited},
			want:  true,
		},
		{name: "a rig with no chain at all"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			r := spec(tt.chain...)
			if tt.rig != nil {
				r.Evidence = evidence(tt.rig...)
			}

			s.Require().Equal(tt.want, rig.Trusted(r))
		})
	}
}

// TestSourced names the strongest evidence anywhere in a rig.
func (s *ReadPublicTestSuite) TestSourced() {
	tests := []struct {
		name  string
		kinds []gen.EvidenceKind
		rig   []gen.EvidenceKind
		want  gen.EvidenceKind
	}{
		{
			// Nothing in this project can hear, so somebody who listened
			// outranks any citation.
			name: "a person, above everything",
			kinds: []gen.EvidenceKind{
				gen.EvidenceMeasured, gen.EvidenceUser, gen.EvidenceCited,
			},
			want: gen.EvidenceUser,
		},
		{
			name: "measured over cited",
			kinds: []gen.EvidenceKind{
				gen.EvidenceCited, gen.EvidenceMeasured,
			},
			want: gen.EvidenceMeasured,
		},
		{
			name:  "cited over video",
			kinds: []gen.EvidenceKind{gen.EvidenceVideo, gen.EvidenceCited},
			want:  gen.EvidenceCited,
		},
		{
			name:  "video over audio",
			kinds: []gen.EvidenceKind{gen.EvidenceAudio, gen.EvidenceVideo},
			want:  gen.EvidenceVideo,
		},
		{
			name:  "audio over corpus",
			kinds: []gen.EvidenceKind{gen.EvidenceCorpus, gen.EvidenceAudio},
			want:  gen.EvidenceAudio,
		},
		{
			name:  "corpus over an assertion",
			kinds: []gen.EvidenceKind{gen.EvidenceLLM, gen.EvidenceCorpus},
			want:  gen.EvidenceCorpus,
		},
		{
			name:  "a kind nobody has ranked",
			kinds: []gen.EvidenceKind{gen.EvidenceKind("seance")},
			want:  gen.EvidenceKind("seance"),
		},
		{
			name: "an unranked kind, which does not outrank a known one",
			kinds: []gen.EvidenceKind{
				gen.EvidenceCorpus, gen.EvidenceKind("seance"),
			},
			want: gen.EvidenceCorpus,
		},
		{
			name:  "the chain, beating what the rig itself carries",
			kinds: []gen.EvidenceKind{gen.EvidenceCited},
			rig:   []gen.EvidenceKind{gen.EvidenceCorpus},
			want:  gen.EvidenceCited,
		},
		{
			// Nothing said where it came from, which is the same standing as
			// a model having asserted it.
			name: "a rig nobody supported",
			want: gen.EvidenceLLM,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			entry := gen.ChainEntry{Role: gen.RoleAmp}
			if tt.kinds != nil {
				entry.Evidence = evidence(tt.kinds...)
			}

			r := spec(entry)
			if tt.rig != nil {
				r.Evidence = evidence(tt.rig...)
			}

			s.Require().Equal(tt.want, rig.Sourced(r))
		})
	}
}

func TestReadPublicTestSuite(t *testing.T) {
	suite.Run(t, new(ReadPublicTestSuite))
}
