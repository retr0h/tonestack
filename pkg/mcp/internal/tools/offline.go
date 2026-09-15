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

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/retr0h/tonestack/pkg/sdk"
	"github.com/retr0h/tonestack/pkg/sdk/catalog"
)

func (h *handlers) catalogSearch(
	ctx context.Context,
	_ *gomcp.CallToolRequest,
	in Search,
) (*gomcp.CallToolResult, sdk.Blocks, error) {
	found, err := h.client.Blocks(ctx, sdk.Filter{
		Category:    in.Category,
		Subcategory: in.Subcategory,
		Search:      in.Search,
	})
	if err != nil {
		return nil, sdk.Blocks{}, err
	}

	return said("%d of %d blocks matched", len(found.Matched), found.Total), found, nil
}

func (h *handlers) catalogBlock(
	ctx context.Context,
	_ *gomcp.CallToolRequest,
	in ID,
) (*gomcp.CallToolResult, catalog.Block, error) {
	block, err := h.client.Block(ctx, in.ID)
	if err != nil {
		return nil, catalog.Block{}, remedy(err)
	}

	return said("%s is %s", block.ID, block.Name), block, nil
}

func (h *handlers) corpusModel(
	ctx context.Context,
	_ *gomcp.CallToolRequest,
	in ID,
) (*gomcp.CallToolResult, Model, error) {
	measured, err := h.client.ModelMeasurements(ctx, in.ID)
	if err != nil {
		return nil, Model{}, err
	}

	stats := measured.Stats.Models[measured.Model]
	block, found := measured.Catalog.Block(measured.Model)
	if !found {
		return nil, Model{}, notInCatalog(string(measured.Model))
	}

	out := Model{Block: block, Uses: stats.Uses, Params: stats.Params}

	return said("%s is used %d times across the measured presets", in.ID, stats.Uses), out, nil
}

func (h *handlers) rigsList(
	ctx context.Context,
	_ *gomcp.CallToolRequest,
	_ None,
) (*gomcp.CallToolResult, sdk.Recipes, error) {
	found, err := h.client.Recipes(ctx)
	if err != nil {
		return nil, sdk.Recipes{}, err
	}

	return said("%d rigs to build from", len(found.Rigs)), found, nil
}

func (h *handlers) rigShow(
	ctx context.Context,
	_ *gomcp.CallToolRequest,
	in ID,
) (*gomcp.CallToolResult, sdk.Recipe, error) {
	found, err := h.client.Recipe(ctx, in.ID)
	if err != nil {
		return nil, sdk.Recipe{}, remedy(err)
	}

	return said("rig %s, extended by %d others", in.ID, len(found.Variants)), found, nil
}

func (h *handlers) presetBuild(
	ctx context.Context,
	_ *gomcp.CallToolRequest,
	in Build,
) (*gomcp.CallToolResult, Built, error) {
	switch {
	case in.RecipeID != "" && in.RigPath != "":
		return nil, Built{}, ErrTwoSources
	case in.RecipeID == "" && in.RigPath == "":
		return nil, Built{}, ErrNoSource
	}

	if err := h.mayWrite(in.Out); err != nil {
		return nil, Built{}, err
	}

	switch {
	case in.RecipeID != "":
		made, err := h.client.Build(ctx, in.RecipeID, in.Out, h.existing())
		if err != nil {
			return nil, Built{}, remedy(h.refused(in.Out, err))
		}

		return said("wrote %s from rig %s", in.Out, in.RecipeID), Built{FromRecipe: &made}, nil
	default:
		built, err := h.client.Compile(ctx, sdk.Compile{
			Rig:      in.RigPath,
			Out:      in.Out,
			Existing: h.existing(),
		})
		if err != nil {
			return nil, Built{}, h.refused(in.Out, err)
		}

		return said("wrote %s from %s", in.Out, in.RigPath), Built{FromRig: &built}, nil
	}
}
