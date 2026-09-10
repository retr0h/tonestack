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

package rig

import "github.com/retr0h/tonestack/pkg/sdk/internal/gen"

// What a rig is made of.
//
// Declared by oapi-codegen from the contract and named here, so that nobody
// outside this package has to hold a generated type. A caller writing
// rig.Spec is writing against a name this project chose; a caller writing
// gen.RigSpec was writing against whatever the generator happened to call it,
// and finding out at compile time when that changed.
//
// Aliases rather than wrappers. rig.Spec and the generated type are the same
// type, so nothing converts at the seam and a rig built by the compiler is a
// rig a caller can read.
type (
	// Spec is a rig, complete. Sparse when hand-written; the same document
	// carries settings and evidence once anything has been measured.
	Spec = gen.RigSpec
	// SpecVersion is the contract version a rig states. Version is the one
	// this package reads and writes; this is the type a rig carries it in.
	SpecVersion = gen.RigSpecVersion
	// Subject is who or what the rig is attributed to.
	Subject = gen.Subject
	// ChainEntry is one piece of gear in the signal path.
	ChainEntry = gen.ChainEntry
	// Role is what a piece of gear does: amp, cab, drive.
	Role = gen.Role
	// Technique is how the instrument is played.
	Technique = gen.Technique
	// Position is where on the string it is played.
	Position = gen.TechniquePosition
	// Muting is what stops the note.
	Muting = gen.TechniqueMuting
	// Attack is what starts it.
	Attack = gen.TechniqueAttack
	// CharacterTerm is how it should sound, in the words a person would use.
	CharacterTerm = gen.CharacterTerm
	// Evidence is where a claim came from.
	Evidence = gen.Evidence
	// EvidenceKind is how far somebody has to go to disagree with one.
	EvidenceKind = gen.EvidenceKind
	// Confidence is how far a claim should be trusted.
	Confidence = gen.Confidence
)

// The values those types may hold.
const (
	// RoleAmp and RoleCab are the two a listing looks for by name.
	RoleAmp = gen.RoleAmp
	RoleCab = gen.RoleCab

	// Where on the string a note is played.
	PositionBridge = gen.PositionBridge
	PositionMiddle = gen.PositionMiddle
	PositionNeck   = gen.PositionNeck

	// What starts the note.
	AttackPick    = gen.AttackPick
	AttackFingers = gen.AttackFingers
	AttackThumb   = gen.AttackThumb
	AttackSlap    = gen.AttackSlap
	AttackHybrid  = gen.AttackHybrid

	// What stops it. Named only when there is some: "not muted" is what
	// every unmuted note already sounds like.
	MutingPalm = gen.MutingPalm
	MutingNone = gen.MutingNone

	// How far a claim should be trusted. Unstated reads as the lowest,
	// because a rig that says nothing about itself has earned nothing.
	ConfidenceLow    = gen.ConfidenceLow
	ConfidenceMedium = gen.ConfidenceMedium
	ConfidenceHigh   = gen.ConfidenceHigh

	// Where a claim came from, and the one that is not checkable.
	EvidenceCited = gen.EvidenceCited
	EvidenceLLM   = gen.EvidenceLLM
)
