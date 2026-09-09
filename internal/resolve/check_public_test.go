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

package resolve_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/internal/resolve"
	"github.com/retr0h/tonestack/pkg/catalog"
	"github.com/retr0h/tonestack/pkg/chain"
	riggen "github.com/retr0h/tonestack/pkg/rig/gen"
)

// CheckPublicTestSuite covers what a rig claims beside its chain.
//
// A colour, a parameter name and a device name are all valid strings, so the
// contract cannot rule on them. Whether they name anything is a question about
// the catalog in hand.
type CheckPublicTestSuite struct {
	suite.Suite

	cat *catalog.Catalog
}

func (s *CheckPublicTestSuite) SetupSuite() {
	s.cat = loadCatalog(&s.Suite)
}

// blocks is a chain holding the bass amp, which is what a controller in these
// cases points at.
func (s *CheckPublicTestSuite) blocks() []chain.Block {
	return []chain.Block{{Model: "HD2_AmpSVBeastNrm"}}
}

// catalogWith returns a catalog whose one block carries the given parameters,
// for the lists too long or too crowded to build out of the fixture.
func (s *CheckPublicTestSuite) catalogWith(params ...string) *catalog.Catalog {
	held := make(map[string]catalog.Param, len(params))
	for _, name := range params {
		held[name] = catalog.Param{Key: name, Type: catalog.ParamFloat}
	}

	return &catalog.Catalog{
		Device:     s.cat.Device,
		LEDColours: s.cat.LEDColours,
		Blocks: map[catalog.ModelID]catalog.Block{
			"HD2_AmpSVBeastNrm": {ID: "HD2_AmpSVBeastNrm", Params: held},
		},
	}
}

// TestCheck covers every claim a rig makes that a catalog has to supply.
func (s *CheckPublicTestSuite) TestCheck() {
	tests := []struct {
		name string
		// what the rig says beside its chain.
		device      string
		led         string
		parameter   string
		block       int
		hasSwitch   bool
		hasControl  bool
		emptyBlocks bool
		// a chain whose one block sits at position 4 rather than 0.
		spaced bool
		// a catalog whose block carries these parameters instead.
		params []string
		// a chain naming a model this catalog does not carry.
		unknownModel bool
		// what the failure must not say.
		absent string

		err     error
		field   string
		suggest string
	}{
		{name: "a rig claiming nothing"},
		{
			// A rig read off a device numbers its blocks the way the device
			// lays them out, so the position a controller names is not where
			// the block sits in the list.
			name:       "a controller on a chain that states its positions",
			parameter:  "Drive",
			block:      4,
			hasControl: true,
			spaced:     true,
		},
		{name: "the device this catalog is for", device: "HX Stomp"},
		{name: "the same name in another case", device: "hx stomp"},
		{
			name:    "a device this catalog is not for",
			device:  "Kemper Profiler",
			err:     resolve.ErrNoSuchValue,
			field:   "target.device",
			suggest: "it has: HX Stomp",
		},
		{name: "a colour the device lights", led: "violet", hasSwitch: true},
		{name: "the same colour shouted", led: "VIOLET", hasSwitch: true},
		{
			// Five colours are worth printing whole, since the list is the
			// answer to the question.
			name:      "a colour it does not",
			led:       "chartruse",
			hasSwitch: true,
			err:       resolve.ErrNoSuchValue,
			field:     "footswitches[0].led",
			suggest:   "it has: auto color, white, green, violet, off",
		},
		{name: "a switch naming no colour", hasSwitch: true},
		{
			name:       "a parameter the block has",
			parameter:  "Drive",
			hasControl: true,
		},
		{
			// Near enough to guess at, so the guess is the answer rather
			// than the whole list.
			name:       "a parameter close to one it has",
			parameter:  "Driv",
			hasControl: true,
			err:        resolve.ErrNoSuchValue,
			field:      "controllers[0].parameter",
			suggest:    "did you mean: Drive",
		},
		{
			name:       "a parameter nothing is close to",
			parameter:  "Loudness",
			hasControl: true,
			err:        resolve.ErrNoSuchValue,
			field:      "controllers[0].parameter",
			suggest:    "it has: Bass, Bright, Drive, MidFreq, Treble",
		},
		{
			// A model from newer firmware than the catalog was generated
			// from. A rig may name one, and nothing here can say whether its
			// parameter is real.
			name:         "a controller on a model this catalog lacks",
			parameter:    "Anything",
			hasControl:   true,
			unknownModel: true,
		},
		{
			// Too many to be an answer, so the list is left out rather than
			// printed at somebody.
			name:       "a parameter on a model with more names than a list is worth",
			parameter:  "Loudness",
			hasControl: true,
			params: []string{
				"A", "B", "C", "D", "E", "F", "G", "H",
				"I", "J", "K", "L", "M", "N", "O",
			},
			err:    resolve.ErrNoSuchValue,
			field:  "controllers[0].parameter",
			absent: "it has",
		},
		{
			name:       "a parameter close to more of them than anybody reads",
			parameter:  "Midd",
			hasControl: true,
			params: []string{
				"Mid1", "Mid2", "Mid3", "Mid4", "Mid5", "Mid6", "Mid7", "Mid8",
			},
			err:     resolve.ErrNoSuchValue,
			field:   "controllers[0].parameter",
			suggest: "did you mean: Mid1, Mid2, Mid3, Mid4, Mid5",
			absent:  "Mid6",
		},
		{
			// A device stores the parameter's place rather than its name, so
			// a controller pointing past the chain writes an assignment onto
			// whatever happens to sit there.
			name:       "a block the chain does not have",
			parameter:  "Drive",
			block:      9,
			hasControl: true,
			err:        resolve.ErrNoSuchBlock,
			field:      "controllers[0].block",
			suggest:    "no block sits at position 9",
		},
		{
			name:        "a controller on a chain with nothing in it",
			parameter:   "Drive",
			hasControl:  true,
			emptyBlocks: true,
			err:         resolve.ErrNoSuchBlock,
			field:       "controllers[0].block",
			suggest:     "the chain holds 0 blocks",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			spec := recipe("Ampeg SVT (normal", "")

			if tt.device != "" {
				spec.Target = &riggen.Target{Device: &tt.device}
			}

			if tt.hasSwitch {
				fs := riggen.Footswitch{}
				if tt.led != "" {
					fs.Led = &tt.led
				}

				spec.Footswitches = &[]riggen.Footswitch{fs}
			}

			if tt.hasControl {
				spec.Controllers = &[]riggen.Controller{{
					Controller: 2, Block: tt.block, Parameter: tt.parameter,
				}}
			}

			cat := s.cat
			if tt.params != nil {
				cat = s.catalogWith(tt.params...)
			}

			blocks := s.blocks()

			if tt.unknownModel {
				blocks[0].Model = "HD2_FromNewerFirmware"
			}

			if tt.spaced {
				blocks[0].Pos = 4
			}

			if tt.emptyBlocks {
				blocks = nil
			}

			err := resolve.Check(spec, blocks, cat)

			if tt.err == nil {
				s.Require().NoError(err)

				return
			}

			s.Require().ErrorIs(err, tt.err)
			s.Require().Contains(err.Error(), tt.field)

			if tt.suggest != "" {
				s.Require().Contains(err.Error(), tt.suggest)
			}

			if tt.absent != "" {
				s.Require().NotContains(err.Error(), tt.absent)
			}
		})
	}
}

func TestCheckPublicTestSuite(t *testing.T) {
	suite.Run(t, new(CheckPublicTestSuite))
}
