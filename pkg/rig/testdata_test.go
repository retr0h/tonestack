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

import "github.com/retr0h/tonestack/pkg/catalog"

// newCatalog returns a real *catalog.Catalog holding blocks.
//
// No double is needed here: *catalog.Catalog already satisfies rig.BlockLookup,
// and lookup is the whole of its behaviour. A mock would assert calls into a
// map, which tests the test rather than the code.
func newCatalog(blocks ...catalog.Block) *catalog.Catalog {
	byID := make(map[catalog.ModelID]catalog.Block, len(blocks))
	for _, b := range blocks {
		byID[b.ID] = b
	}

	return &catalog.Catalog{Device: "HX Stomp", DeviceID: 2162694, Blocks: byID}
}

// testAmp returns a block with one float and one enum parameter.
func testAmp() catalog.Block {
	return catalog.Block{
		ID:       "HD2_AmpTest",
		Name:     "Test Amp",
		Category: catalog.CategoryAmp,
		Stereo:   false,
		Prov:     catalog.ProvObserved,
		DSP:      catalog.DSPCost{Mono: 26.67, Stereo: 40.1, Prov: catalog.ProvOfficial},
		Params: map[string]catalog.Param{
			"Gain": {
				Key: "Gain", Label: "Drive", Type: catalog.ParamFloat,
				Min: 0.0, Max: 1.0, Default: catalog.Float(0.5),
			},
			"Mode": {
				Key: "Mode", Label: "Mode", Type: catalog.ParamEnum,
				Enum: []string{"Normal", "Bright"}, Default: catalog.Enum("Normal"),
			},
		},
	}
}
