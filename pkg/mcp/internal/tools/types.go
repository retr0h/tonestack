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

package tools

import (
	"context"

	"github.com/retr0h/tonestack/pkg/sdk"
	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/corpus"
	"github.com/retr0h/tonestack/pkg/sdk/rig"
	"github.com/retr0h/tonestack/pkg/sdk/slot"
)

// Client is what the tools call. FromSDK makes one of an *sdk.Client.
//
// Declared here, where it is used, so a test can put a generated double in
// front of the handlers without a pedal on the bus.
type Client interface {
	Blocks(ctx context.Context, f sdk.Filter) (sdk.Blocks, error)
	Block(ctx context.Context, id string) (catalog.Block, error)
	Measurements(ctx context.Context, in sdk.Corpus) (sdk.Measured, error)
	Recipes(ctx context.Context) (sdk.Recipes, error)
	Recipe(ctx context.Context, id string) (sdk.Recipe, error)
	Build(ctx context.Context, in sdk.Make) (sdk.Made, error)
	Compile(ctx context.Context, in sdk.Compile) (sdk.Built, error)
	Devices(ctx context.Context) (sdk.Attached, error)
	// Open claims the pedal. The tools hold what it returns across calls.
	Open(ctx context.Context) (Session, error)
}

// Session is what the device tools call while the pedal is held.
// *sdk.Session satisfies it.
type Session interface {
	Presets(ctx context.Context, setlist int) (sdk.Listing, error)
	Preset(ctx context.Context, at slot.Address) (sdk.Reading, error)
	Export(ctx context.Context, at slot.Address, out string, as string) (sdk.Written, error)
	Import(ctx context.Context, file string, at slot.Address) (sdk.Change, error)
	Copy(ctx context.Context, from, to slot.Address) (sdk.Change, error)
	Swap(ctx context.Context, a, b slot.Address) (sdk.Change, error)
	Select(ctx context.Context, at slot.Address) (sdk.Change, error)
	Close() error
}

// Search narrows catalog_search.
type Search struct {
	Category    string `json:"category,omitempty"    jsonschema:"only blocks of one kind, such as amp, cab, drive, dynamics, eq, delay, reverb or modulation"`
	Subcategory string `json:"subcategory,omitempty" jsonschema:"Line 6's own grouping, such as Guitar or Bass"`
	Search      string `json:"search,omitempty"      jsonschema:"text in the block's name or in the real-world gear it models"`
}

// ID names one thing: a model for catalog_block and corpus_model, a rig for
// rig_show.
type ID struct {
	ID string `json:"id" jsonschema:"the identifier, such as HD2_AmpSVBeastBrt for a model or mike-dirnt for a rig"`
}

// None is the input of a tool that takes nothing.
type None struct{}

// Build says what preset_build builds from and where the file goes.
type Build struct {
	RecipeID string `json:"recipe_id,omitempty" jsonschema:"a rig that ships, by identifier; see rigs_list"`
	RigPath  string `json:"rig_path,omitempty"  jsonschema:"a rig file on disk"`
	Out      string `json:"out"                 jsonschema:"where to write the .hlx"`
}

// Built is what preset_build answers: exactly one side is set.
type Built struct {
	FromRecipe *sdk.Made  `json:"from_recipe,omitempty"`
	FromRig    *sdk.Built `json:"from_rig,omitempty"`
}

// Model is one model as corpus_model answers it: the block, and how players
// set it. The whole corpus and catalog stay out, because an agent reading them
// would read nothing else.
type Model struct {
	Block  catalog.Block                `json:"block"`
	Uses   int                          `json:"uses"`
	Params map[string]corpus.ParamStats `json:"params"`
}

// Slot addresses one slot on the pedal.
type Slot struct {
	Slot string `json:"slot" jsonschema:"a slot as the pedal labels it, 01A to 42C"`
}

// Export says which slot preset_export writes out, where, and as what.
type Export struct {
	Slot string `json:"slot"         jsonschema:"a slot as the pedal labels it, 01A to 42C"`
	Out  string `json:"out"          jsonschema:"where to write the file"`
	As   string `json:"as,omitempty" jsonschema:"empty for a rig, which reads on other hardware; hlx for the device's own file"`
}

// Shown is a slot as preset_show answers it. The .hlx document stays out: the
// rig is what an agent reasons about, and preset_export writes the rest.
type Shown struct {
	Name   string      `json:"name"`
	Rig    rig.Spec    `json:"rig"`
	Answer *sdk.Answer `json:"answer,omitempty"`
}

// Put says which .hlx preset_import places, and where.
type Put struct {
	Preset string `json:"preset" jsonschema:"the .hlx file to place"`
	Slot   string `json:"slot"   jsonschema:"a slot as the pedal labels it, 01A to 42C"`
}

// Move names the two slots presets_copy and presets_swap work on.
type Move struct {
	From string `json:"from" jsonschema:"the source slot, 01A to 42C"`
	To   string `json:"to"   jsonschema:"the destination slot, 01A to 42C"`
}
