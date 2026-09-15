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

package tools_test

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/retr0h/tonestack/pkg/mcp/internal/tools"
	"github.com/retr0h/tonestack/pkg/mcp/internal/tools/mocks"
	"github.com/retr0h/tonestack/pkg/sdk"
	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/corpus"
	"github.com/retr0h/tonestack/pkg/sdk/rig"
)

type OfflinePublicTestSuite struct {
	suite.Suite
	client *mocks.MockClient
}

func (s *OfflinePublicTestSuite) SetupSubTest() {
	s.client = mocks.NewMockClient(gomock.NewController(s.T()))
}

// row is one call and what it should come back with. check reads the
// structured answer of a call that succeeded.
type row struct {
	name  string
	args  any
	setup func(c *mocks.MockClient)
	want  string
	err   bool
	check func(s *OfflinePublicTestSuite, res *gomcp.CallToolResult)
	// allowWrites starts the server the way --allow-writes does.
	allowWrites bool
}

func (s *OfflinePublicTestSuite) run(
	tool string,
	tests []row,
) {
	for _, tt := range tests {
		s.Run(tt.name, func() {
			if tt.setup != nil {
				tt.setup(s.client)
			}

			res := call(s.T(), connect(s.T(), s.client, tt.allowWrites), tool, tt.args)

			s.Equal(tt.err, res.IsError)
			s.Contains(text(s.T(), res), tt.want)

			if tt.check != nil {
				tt.check(s, res)
			}
		})
	}
}

var errUnreadable = errors.New("catalog unreadable")

// TestCatalogSearch covers finding blocks.
func (s *OfflinePublicTestSuite) TestCatalogSearch() {
	s.run("catalog_search", []row{
		{
			name: "blocks that match",
			args: tools.Search{Subcategory: "Bass", Search: "SVT"},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().Blocks(gomock.Any(), sdk.Filter{Subcategory: "Bass", Search: "SVT"}).
					Return(sdk.Blocks{Total: 547, Matched: []catalog.Block{
						{ID: "HD2_AmpSVBeastBrt", Params: map[string]catalog.Param{}},
					}}, nil)
			},
			want: "1 of 547 blocks matched",
			check: func(s *OfflinePublicTestSuite, res *gomcp.CallToolResult) {
				var got sdk.Blocks
				structured(s.T(), res, &got)
				s.Len(got.Matched, 1)
			},
		},
		{
			name: "a catalog that will not open",
			args: tools.Search{},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().Blocks(gomock.Any(), sdk.Filter{}).Return(sdk.Blocks{}, errUnreadable)
			},
			want: "catalog unreadable",
			err:  true,
		},
	})
}

// TestCatalogBlock covers reading one block.
func (s *OfflinePublicTestSuite) TestCatalogBlock() {
	s.run("catalog_block", []row{
		{
			name: "a block the catalog has",
			args: tools.ID{ID: "HD2_AmpSVBeastBrt"},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().Block(gomock.Any(), "HD2_AmpSVBeastBrt").
					Return(catalog.Block{
						ID: "HD2_AmpSVBeastBrt", Name: "Ampeg SVT Brt",
						Params: map[string]catalog.Param{},
					}, nil)
			},
			want: "HD2_AmpSVBeastBrt is Ampeg SVT Brt",
			check: func(s *OfflinePublicTestSuite, res *gomcp.CallToolResult) {
				var got catalog.Block
				structured(s.T(), res, &got)
				s.Equal("Ampeg SVT Brt", got.Name)
			},
		},
		{
			// An agent is pointed at the tool that finds blocks, not at a
			// command it cannot run.
			name: "a block it does not",
			args: tools.ID{ID: "HD2_Nope"},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().
					Block(gomock.Any(), "HD2_Nope").
					Return(catalog.Block{}, fmt.Errorf("%w %q", sdk.ErrNoSuchBlock, "HD2_Nope"))
			},
			want: "call catalog_search",
			err:  true,
		},
		{
			name: "a catalog that will not open",
			args: tools.ID{ID: "HD2_AmpSVBeastBrt"},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().
					Block(gomock.Any(), "HD2_AmpSVBeastBrt").
					Return(catalog.Block{}, errUnreadable)
			},
			want: "catalog unreadable",
			err:  true,
		},
	})
}

// TestCorpusModel covers how players set one model.
func (s *OfflinePublicTestSuite) TestCorpusModel() {
	id := catalog.ModelID("HD2_AmpSVBeastBrt")
	cat := &catalog.Catalog{Blocks: map[catalog.ModelID]catalog.Block{
		id: {ID: id, Name: "Ampeg SVT Brt", Params: map[string]catalog.Param{}},
	}}

	s.run("corpus_model", []row{
		{
			name: "a model the corpus measured",
			args: tools.ID{ID: string(id)},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().
					ModelMeasurements(gomock.Any(), string(id)).
					Return(sdk.Measured{
						Stats: &corpus.Stats{Models: map[catalog.ModelID]corpus.ModelStats{
							id: {
								Uses: 26,
								Params: map[string]corpus.ParamStats{
									"Treble": {N: 26, Median: 0.85},
								},
							},
						}},
						Catalog: cat,
						Model:   id,
					}, nil)
			},
			want: "used 26 times",
			check: func(s *OfflinePublicTestSuite, res *gomcp.CallToolResult) {
				var got tools.Model
				structured(s.T(), res, &got)
				s.Equal("Ampeg SVT Brt", got.Block.Name)
				s.InDelta(0.85, got.Params["Treble"].Median, 1e-9)
			},
		},
		{
			name: "a model nobody measured",
			args: tools.ID{ID: "HD2_Nope"},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().ModelMeasurements(gomock.Any(), "HD2_Nope").
					Return(sdk.Measured{}, errors.New("HD2_Nope was not measured"))
			},
			want: "was not measured",
			err:  true,
		},
		{
			// The corpus measured this model, but the catalog it was
			// resolved against has since dropped it: corpus and catalog
			// have drifted apart.
			name: "a model the corpus measured but the catalog lacks",
			args: tools.ID{ID: string(id)},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().
					ModelMeasurements(gomock.Any(), string(id)).
					Return(sdk.Measured{
						Stats: &corpus.Stats{Models: map[catalog.ModelID]corpus.ModelStats{
							id: {Uses: 26},
						}},
						Catalog: &catalog.Catalog{},
						Model:   id,
					}, nil)
			},
			want: tools.ErrNotInCatalog.Error(),
			err:  true,
		},
	})
}

// TestRigsList covers listing the shipped rigs.
func (s *OfflinePublicTestSuite) TestRigsList() {
	s.run("rigs_list", []row{
		{
			name: "the rigs that ship",
			args: tools.None{},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().Recipes(gomock.Any()).Return(sdk.Recipes{Rigs: []rig.Spec{{}, {}}}, nil)
			},
			want: "2 rigs to build from",
			check: func(s *OfflinePublicTestSuite, res *gomcp.CallToolResult) {
				var got sdk.Recipes
				structured(s.T(), res, &got)
				s.Len(got.Rigs, 2)
			},
		},
		{
			name: "rigs that will not read",
			args: tools.None{},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().
					Recipes(gomock.Any()).
					Return(sdk.Recipes{}, errors.New("rigs unreadable"))
			},
			want: "rigs unreadable",
			err:  true,
		},
	})
}

// TestRigShow covers reading one rig.
func (s *OfflinePublicTestSuite) TestRigShow() {
	s.run("rig_show", []row{
		{
			name: "a rig that ships",
			args: tools.ID{ID: "mike-dirnt"},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().Recipe(gomock.Any(), "mike-dirnt").
					Return(sdk.Recipe{Variants: []sdk.Variant{{}}}, nil)
			},
			want: "rig mike-dirnt, extended by 1 others",
			check: func(s *OfflinePublicTestSuite, res *gomcp.CallToolResult) {
				var got sdk.Recipe
				structured(s.T(), res, &got)
				s.Len(got.Variants, 1)
			},
		},
		{
			name: "a rig that does not",
			args: tools.ID{ID: "nobody"},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().
					Recipe(gomock.Any(), "nobody").
					Return(sdk.Recipe{}, fmt.Errorf("%w %q", sdk.ErrNoSuchRecipe, "nobody"))
			},
			want: "call rigs_list",
			err:  true,
		},
	})
}

// TestPresetBuild covers building from either source.
func (s *OfflinePublicTestSuite) TestPresetBuild() {
	dir := s.T().TempDir()
	fresh := filepath.Join(dir, "fresh.hlx")
	held := filepath.Join(dir, "held.hlx")
	s.Require().NoError(os.WriteFile(held, []byte("somebody's preset"), 0o600))

	s.run("preset_build", []row{
		{
			name: "a path nothing is at",
			args: tools.Build{RecipeID: "mike-dirnt", Out: fresh},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().Build(gomock.Any(), "mike-dirnt", fresh).
					Return(sdk.Made{}, nil)
			},
			want: "wrote " + fresh,
		},
		{
			// No client call is expected, so reaching one fails the row.
			name: "a path a file is at, with writes off",
			args: tools.Build{RigPath: "mine.yaml", Out: held},
			want: tools.ErrWouldOverwrite.Error() + ": " + held,
			err:  true,
		},
		{
			name: "a path a file is at, with writes on",
			args: tools.Build{RecipeID: "mike-dirnt", Out: held},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().Build(gomock.Any(), "mike-dirnt", held).
					Return(sdk.Made{}, nil)
			},
			want:        "wrote " + held,
			allowWrites: true,
		},
		{
			name: "from a shipped rig",
			args: tools.Build{RecipeID: "mike-dirnt", Out: "mike.hlx"},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().
					Build(gomock.Any(), "mike-dirnt", "mike.hlx").
					Return(sdk.Made{}, nil)
			},
			want: "wrote mike.hlx from rig mike-dirnt",
			check: func(s *OfflinePublicTestSuite, res *gomcp.CallToolResult) {
				var got tools.Built
				structured(s.T(), res, &got)
				s.NotNil(got.FromRecipe)
				s.Nil(got.FromRig)
			},
		},
		{
			name: "from a rig file",
			args: tools.Build{RigPath: "mine.yaml", Out: "mine.hlx"},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().
					Compile(gomock.Any(), sdk.Compile{Rig: "mine.yaml", Out: "mine.hlx"}).
					Return(sdk.Built{}, nil)
			},
			want: "wrote mine.hlx from mine.yaml",
			check: func(s *OfflinePublicTestSuite, res *gomcp.CallToolResult) {
				var got tools.Built
				structured(s.T(), res, &got)
				s.NotNil(got.FromRig)
			},
		},
		{
			name: "a shipped rig that will not build",
			args: tools.Build{RecipeID: "mike-dirnt", Out: "mike.hlx"},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().
					Build(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(sdk.Made{}, errors.New("over budget"))
			},
			want: "over budget",
			err:  true,
		},
		{
			name: "a shipped rig nobody wrote",
			args: tools.Build{RecipeID: "nobody", Out: "nobody.hlx"},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().
					Build(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(sdk.Made{}, fmt.Errorf("%w %q", sdk.ErrNoSuchRecipe, "nobody"))
			},
			want: "call rigs_list",
			err:  true,
		},
		{
			name: "a rig file that will not build",
			args: tools.Build{RigPath: "mine.yaml", Out: "mine.hlx"},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().
					Compile(gomock.Any(), gomock.Any()).
					Return(sdk.Built{}, errors.New("unknown block"))
			},
			want: "unknown block",
			err:  true,
		},
		{
			name: "nothing to build from",
			args: tools.Build{Out: "x.hlx"},
			want: tools.ErrNoSource.Error(),
			err:  true,
		},
		{
			name: "two things to build from",
			args: tools.Build{RecipeID: "a", RigPath: "b", Out: "x.hlx"},
			want: tools.ErrTwoSources.Error(),
			err:  true,
		},
	})
}

func TestOfflinePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(OfflinePublicTestSuite))
}
