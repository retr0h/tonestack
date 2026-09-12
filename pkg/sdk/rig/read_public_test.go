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
)

type ReadPublicTestSuite struct {
	suite.Suite
}

// spec returns a rig holding the given chain, with nothing else filled in.
func spec(chain ...rig.ChainEntry) rig.Spec {
	return rig.Spec{
		Schema:     rig.SchemaName,
		ID:         "test",
		Subject:    rig.Subject{Kind: rig.KindArtist, Name: "Test"},
		Instrument: rig.InstrumentBass,
		Chain:      chain,
	}
}

// evidence returns the kinds as a rig carries them.
func evidence(kinds ...rig.EvidenceKind) *[]rig.Evidence {
	out := make([]rig.Evidence, 0, len(kinds))
	for _, k := range kinds {
		out = append(out, rig.Evidence{Kind: k})
	}

	return &out
}

// TestGear looks a role up in a chain.
func (s *ReadPublicTestSuite) TestGear() {
	tests := []struct {
		name  string
		chain []rig.ChainEntry
		role  rig.Role
		want  string
		ok    bool
	}{
		{
			name: "a role the chain has",
			chain: []rig.ChainEntry{
				{Role: rig.RoleDrive, Gear: "Klon Centaur"},
				{Role: rig.RoleAmp, Gear: "Ampeg SVT"},
			},
			role: rig.RoleAmp,
			want: "Ampeg SVT",
			ok:   true,
		},
		{
			name:  "a role it does not",
			chain: []rig.ChainEntry{{Role: rig.RoleAmp}},
			role:  rig.RoleCab,
		},
		{
			name: "an empty chain",
			role: rig.RoleAmp,
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
		chain []rig.ChainEntry
		rig   []rig.EvidenceKind
		want  bool
	}{
		{
			name:  "a claim nobody supported",
			chain: []rig.ChainEntry{{Role: rig.RoleAmp}},
		},
		{
			// `llm` means a model said so and nobody checked, which is the
			// same standing as nobody having said anything.
			name: "an assertion alone",
			chain: []rig.ChainEntry{
				{Role: rig.RoleAmp, Evidence: evidence(rig.EvidenceLLM)},
			},
		},
		{
			name: "one unsupported claim among supported ones",
			chain: []rig.ChainEntry{
				{Role: rig.RoleAmp, Evidence: evidence(rig.EvidenceCited)},
				{Role: rig.RoleCab, Evidence: evidence(rig.EvidenceLLM)},
			},
		},
		{
			name: "every claim supported",
			chain: []rig.ChainEntry{
				{Role: rig.RoleAmp, Evidence: evidence(rig.EvidenceCited)},
				{Role: rig.RoleCab, Evidence: evidence(rig.EvidenceVideo)},
			},
			want: true,
		},
		{
			// A rig rundown covers every piece of gear in it. Requiring the
			// citation on each entry would only encourage repeating it.
			name:  "evidence on the rig, covering the whole chain",
			chain: []rig.ChainEntry{{Role: rig.RoleAmp}},
			rig:   []rig.EvidenceKind{rig.EvidenceCited},
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
		kinds []rig.EvidenceKind
		rig   []rig.EvidenceKind
		want  rig.EvidenceKind
	}{
		{
			// Nothing in this project can hear, so somebody who listened
			// outranks any citation.
			name: "a person, above everything",
			kinds: []rig.EvidenceKind{
				rig.EvidenceMeasured, rig.EvidenceUser, rig.EvidenceCited,
			},
			want: rig.EvidenceUser,
		},
		{
			name: "measured over cited",
			kinds: []rig.EvidenceKind{
				rig.EvidenceCited, rig.EvidenceMeasured,
			},
			want: rig.EvidenceMeasured,
		},
		{
			name:  "cited over video",
			kinds: []rig.EvidenceKind{rig.EvidenceVideo, rig.EvidenceCited},
			want:  rig.EvidenceCited,
		},
		{
			name:  "video over audio",
			kinds: []rig.EvidenceKind{rig.EvidenceAudio, rig.EvidenceVideo},
			want:  rig.EvidenceVideo,
		},
		{
			name:  "audio over corpus",
			kinds: []rig.EvidenceKind{rig.EvidenceCorpus, rig.EvidenceAudio},
			want:  rig.EvidenceAudio,
		},
		{
			name:  "corpus over an assertion",
			kinds: []rig.EvidenceKind{rig.EvidenceLLM, rig.EvidenceCorpus},
			want:  rig.EvidenceCorpus,
		},
		{
			name:  "a kind nobody has ranked",
			kinds: []rig.EvidenceKind{rig.EvidenceKind("seance")},
			want:  rig.EvidenceKind("seance"),
		},
		{
			name: "an unranked kind, which does not outrank a known one",
			kinds: []rig.EvidenceKind{
				rig.EvidenceCorpus, rig.EvidenceKind("seance"),
			},
			want: rig.EvidenceCorpus,
		},
		{
			name:  "the chain, beating what the rig itself carries",
			kinds: []rig.EvidenceKind{rig.EvidenceCited},
			rig:   []rig.EvidenceKind{rig.EvidenceCorpus},
			want:  rig.EvidenceCited,
		},
		{
			// Nothing said where it came from, which is the same standing as
			// a model having asserted it.
			name: "a rig nobody supported",
			want: rig.EvidenceLLM,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			entry := rig.ChainEntry{Role: rig.RoleAmp}
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
