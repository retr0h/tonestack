# An MCP server: implementation plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use
> superpowers-extended-cc:subagent-driven-development (recommended) or
> superpowers-extended-cc:executing-plans to implement this plan task-by-task.
> Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `tonestack mcp` serves the catalog, the corpus, the shipped rigs,
building, and the pedal to an agent as MCP tools over stdio.

**Architecture:** `pkg/mcp/internal/tools` registers one handler per tool on a
`github.com/modelcontextprotocol/go-sdk` server. Each handler calls a `Client`
interface that `*sdk.Client` satisfies. `pkg/mcp` wraps that in `New` and `Run`.
`cmd/mcp.go` runs it with the signal context `Execute` already builds. Device
tools take one claim on a size-one channel, so two calls never hold the editor
interface at once. The three writing tools exist only with `--allow-writes`.

**Tech Stack:** Go 1.27, `github.com/modelcontextprotocol/go-sdk` v1.7.0, cobra,
testify/suite, go.uber.org/mock (mockgen via `go tool`).

**Spec:**
[docs/superpowers/specs/2026-09-13-an-mcp-server-design.md](../specs/2026-09-13-an-mcp-server-design.md)
(PR #104, merged)

## Global Constraints

- Coverage gate is 99%. Fix code or add tests. Never edit `.github/codecov.yml`,
  `justfile` coverage target, or `.coverignore`.
- An interface lives where it is consumed. Doubles are generated with mockgen
  and committed as `*.gen.go`. No hand-written doubles.
- A test file in a `_test` package is `*_public_test.go` with a
  `{Name}PublicTestSuite`; an internal test is `*_test.go` with
  `{Name}TestSuite`. A test file is named for the production file it tests.
  testify/suite, table-driven, one suite method per function under test.
- Every `.go` file starts with the MIT header copied from `cmd/devices.go`
  lines 1-19.
- Functions with parameters put one parameter per line, closing parenthesis and
  return types on their own line.
- `types.go` holds only type declarations.
- Never set `SilenceUsage`.
- Markdown changes go through the `unslop` skill before committing.
- Run `mise exec -- just ready` and `mise exec -- just test` once, before the
  pull request, not after every edit.
- Commit trailer (AGENTS.md):
  `🤖 Generated with [Claude Code](https://claude.ai/code)`, blank line,
  `Co-Authored-By: Claude <noreply@anthropic.com>`.
- Tools that write to a pedal are registered only when `AllowWrites` is true,
  and carry `DestructiveHint: true`.
- `pkg/mcp` reaches nothing in this module outside `pkg/mcp` and `pkg/sdk`.

**User decisions (already made):**

- MCP is built like the CLI so it can be extracted to `tonestack-mcp`, with
  start, stop, and per-call cancel from the command's context.
- Device writes are off by default and enabled with `tonestack mcp --allow-writes`
  (decided on the user's behalf at their request, recorded in PR #104).
- No Claude skill: the server's instructions and tool descriptions carry the
  guidance, and `docs/workflows.md` stays the one place the steps live.

## Two deviations from the spec, and why

1. **The spec says tests stand in for the USB bus through the SDK's test
   options.** Those hooks (`sdk.NewLister`, `sdk.OpenDevice`) live in
   `pkg/sdk/export_test.go` and are invisible outside `pkg/sdk`. Handlers are
   tested against a mockgen double of the `Client` interface instead, through
   a real in-memory MCP session. No pedal is needed either way.
2. **The spec says each tool returns the SDK's own result type.** Two of them
   would flood an agent: `sdk.Measured` carries the whole corpus and the whole
   catalog, and `sdk.Reading` carries the full `.hlx` document. `corpus_model`
   returns the one model's block and stats, and `preset_show` returns the name,
   the rig, and any non-preset answer. Every other tool returns the SDK type
   unchanged. `preset_build` returns whichever of `sdk.Made` or `sdk.Built`
   the source produced.

Verified before writing this plan: `mcp.AddTool` infers an output schema without
panicking for every SDK result type used here (`Blocks`, `catalog.Block`,
`Measured`, `Recipes`, `Recipe`, `Made`, `Built`, `Attached`, `Listing`,
`Reading`, `Written`, `Change`).

## File map

| file                                                | holds                                                        |
| --------------------------------------------------- | ------------------------------------------------------------ |
| `pkg/mcp/internal/tools/types.go`                   | `Client` interface, tool inputs and trimmed outputs          |
| `pkg/mcp/internal/tools/mocks/generate.go`          | mockgen directive                                            |
| `pkg/mcp/internal/tools/mocks/types.gen.go`         | generated `MockClient`                                       |
| `pkg/mcp/internal/tools/register.go`                | `Register`, the tool table, annotations, `said`              |
| `pkg/mcp/internal/tools/errors.go`                  | `ErrNoSource`, `ErrTwoSources`                               |
| `pkg/mcp/internal/tools/offline.go`                 | catalog, corpus, rigs, build handlers                        |
| `pkg/mcp/internal/tools/device.go`                  | `claim`, `slotOf`, device read handlers                      |
| `pkg/mcp/internal/tools/writes.go`                  | import, copy, swap handlers                                  |
| `pkg/mcp/internal/tools/register_public_test.go`    | session helpers, `RegisterPublicTestSuite`                   |
| `pkg/mcp/internal/tools/offline_public_test.go`     | `OfflinePublicTestSuite`                                     |
| `pkg/mcp/internal/tools/device_public_test.go`      | `DevicePublicTestSuite`                                      |
| `pkg/mcp/internal/tools/device_test.go`             | `DeviceTestSuite` for `claim`                                |
| `pkg/mcp/internal/tools/writes_public_test.go`      | `WritesPublicTestSuite`                                      |
| `pkg/mcp/mcp.go`                                    | `Options`, `Server`, `New`, `Run`, `Serve`, instructions     |
| `pkg/mcp/mcp_public_test.go`                        | `MCPPublicTestSuite` against a real `sdk.Client`             |
| `main_test.go`                                      | `TestTheMCPStandsAlone`; CLI test allows `pkg/mcp`           |
| `cmd/root.go`                                       | `var version = "dev"` (goreleaser's `-X cmd.version` target) |
| `cmd/mcp.go`                                        | `tonestack mcp [--allow-writes]`                             |
| `docs/commands.md`                                  | regenerated by `just ready`                                  |
| `docs/workflows.md`                                 | "Use it from an agent" section                               |
| `README.md`, `CONTRIBUTING.md`, the spec's `Status` | the feature row, package layout, status                      |

---

### Task 1: The tools package, with the tools that never touch a pedal

**Goal:** `tools.Register` adds `catalog_search`, `catalog_block`,
`corpus_model`, `rigs_list`, `rig_show` and `preset_build` to an MCP server,
each calling a `Client` interface and answering with structured content plus
one line of text.

**Files:**

- Modify: `go.mod`, `go.sum`
- Create: `pkg/mcp/internal/tools/types.go`
- Create: `pkg/mcp/internal/tools/mocks/generate.go`
- Create: `pkg/mcp/internal/tools/mocks/types.gen.go` (generated)
- Create: `pkg/mcp/internal/tools/register.go`
- Create: `pkg/mcp/internal/tools/errors.go`
- Create: `pkg/mcp/internal/tools/offline.go`
- Create: `pkg/mcp/internal/tools/doc.go`
- Test: `pkg/mcp/internal/tools/register_public_test.go`
- Test: `pkg/mcp/internal/tools/offline_public_test.go`

**Acceptance Criteria:**

- [ ] `ListTools` on a session over `Register(server, client, false)` returns
      exactly the six offline tool names in this task (device tools arrive in
      Task 2).
- [ ] Every offline tool has `ReadOnlyHint: true` except `preset_build`, which
      has `ReadOnlyHint: false`.
- [ ] Each handler's success row asserts the text line and at least one
      structured field; each error row asserts `IsError` and the client's
      error text.
- [ ] `preset_build` with neither source returns `ErrNoSource`'s text; with both
      returns `ErrTwoSources`'s text.
- [ ] `go test ./pkg/mcp/...` passes, and coverage of `pkg/mcp/internal/tools`
      is 100%.

**Verify:**
`go test -cover ./pkg/mcp/internal/tools/` → `ok ... coverage: 100.0% of statements`

**Steps:**

- [ ] **Step 1: Branch, and add the dependency**

```bash
git switch main && git pull && git switch -c feat/mcp-server
go get github.com/modelcontextprotocol/go-sdk@v1.7.0
```

Every task in this plan commits on `feat/mcp-server`.

- [ ] **Step 2: Write `types.go`** (licence header first, then:)

```go
package tools

import (
	"context"

	"github.com/retr0h/tonestack/pkg/sdk"
	"github.com/retr0h/tonestack/pkg/sdk/catalog"
	"github.com/retr0h/tonestack/pkg/sdk/corpus"
	"github.com/retr0h/tonestack/pkg/sdk/rig"
)

// Client is what the tools call. *sdk.Client satisfies it.
//
// Declared here, where it is used, so a test can put a generated double in
// front of the handlers without a pedal on the bus.
type Client interface {
	Blocks(catalogPath string, f sdk.Filter) (sdk.Blocks, error)
	Block(catalogPath, id string) (catalog.Block, error)
	Measurements(in sdk.Corpus) (sdk.Measured, error)
	Recipes(dir string) (sdk.Recipes, error)
	Recipe(dir, id string) (sdk.Recipe, error)
	Build(in sdk.Make) (sdk.Made, error)
	Compile(in sdk.Compile) (sdk.Built, error)
	Devices(ctx context.Context) (sdk.Attached, error)
	Presets(ctx context.Context, in sdk.Where) (sdk.Listing, error)
	Preset(ctx context.Context, in sdk.Read) (sdk.Reading, error)
	Export(ctx context.Context, in sdk.Export) (sdk.Written, error)
	Select(ctx context.Context, in sdk.Read) (sdk.Change, error)
	Import(ctx context.Context, in sdk.Put) (sdk.Change, error)
	Copy(ctx context.Context, in sdk.Edit) (sdk.Change, error)
	Swap(ctx context.Context, in sdk.Edit) (sdk.Change, error)
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
```

- [ ] **Step 3: Write `doc.go`, `errors.go` and `mocks/generate.go`**

```go
// doc.go
// Package tools is one handler per MCP tool, each calling only pkg/sdk.
package tools
```

```go
// errors.go
package tools

import "errors"

var (
	// ErrNoSource is preset_build given nothing to build from.
	ErrNoSource = errors.New("name a recipe_id or a rig_path to build from")
	// ErrTwoSources is preset_build given both.
	ErrTwoSources = errors.New("name a recipe_id or a rig_path, not both")
)
```

```go
// mocks/generate.go
// Package mocks holds generated test doubles for the tools package's interfaces.
package mocks

//go:generate go tool go.uber.org/mock/mockgen -source=../types.go -destination=types.gen.go -package=mocks
```

Run: `go generate ./pkg/mcp/internal/tools/mocks/` and confirm `types.gen.go`
declares `MockClient`. Add the licence header to `types.gen.go` only if
`just license-check` demands it of other `*.gen.go` files (check
`pkg/sdk/internal/presets/mocks/types.gen.go` and match it).

- [ ] **Step 4: Write the session helpers and the failing register test**

`register_public_test.go`:

```go
package tools_test

import (
	"context"
	"encoding/json"
	"testing"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/retr0h/tonestack/pkg/mcp/internal/tools"
	"github.com/retr0h/tonestack/pkg/mcp/internal/tools/mocks"
)

// connect puts the tools on a server and a client session in front of it,
// over the library's in-memory transport, so a test calls a tool the way an
// agent does.
func connect(
	t *testing.T,
	c tools.Client,
	allowWrites bool,
) *gomcp.ClientSession {
	t.Helper()

	server := gomcp.NewServer(&gomcp.Implementation{Name: "tonestack", Version: "test"}, nil)
	tools.Register(server, c, allowWrites)

	serverEnd, clientEnd := gomcp.NewInMemoryTransports()
	ctx := context.Background()

	_, err := server.Connect(ctx, serverEnd, nil)
	require.NoError(t, err)

	session, err := gomcp.NewClient(
		&gomcp.Implementation{Name: "test", Version: "test"}, nil,
	).Connect(ctx, clientEnd, nil)
	require.NoError(t, err)

	t.Cleanup(func() { _ = session.Close() })

	return session
}

// call calls one tool and fails the test only on a protocol error. A tool
// error comes back as a result with IsError set, which is what the tests read.
func call(
	t *testing.T,
	session *gomcp.ClientSession,
	name string,
	args any,
) *gomcp.CallToolResult {
	t.Helper()

	res, err := session.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      name,
		Arguments: args,
	})
	require.NoError(t, err)

	return res
}

// text is the line a tool said.
func text(
	t *testing.T,
	res *gomcp.CallToolResult,
) string {
	t.Helper()
	require.NotEmpty(t, res.Content)

	said, ok := res.Content[0].(*gomcp.TextContent)
	require.True(t, ok, "first content is %T", res.Content[0])

	return said.Text
}

// structured decodes what a tool answered into a Go value.
func structured(
	t *testing.T,
	res *gomcp.CallToolResult,
	into any,
) {
	t.Helper()

	raw, err := json.Marshal(res.StructuredContent)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, into))
}

type RegisterPublicTestSuite struct {
	suite.Suite
}

// TestRegister covers which tools an agent is offered.
func (s *RegisterPublicTestSuite) TestRegister() {
	offline := []string{
		"catalog_block", "catalog_search", "corpus_model",
		"preset_build", "rig_show", "rigs_list",
	}

	tests := []struct {
		name        string
		allowWrites bool
		want        []string
		readOnly    map[string]bool
	}{
		{
			name: "without writes",
			want: offline,
			readOnly: map[string]bool{
				"catalog_block": true, "catalog_search": true, "corpus_model": true,
				"preset_build": false, "rig_show": true, "rigs_list": true,
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			client := mocks.NewMockClient(gomock.NewController(s.T()))
			session := connect(s.T(), client, tt.allowWrites)

			listed, err := session.ListTools(context.Background(), nil)
			s.Require().NoError(err)

			var names []string
			for _, tool := range listed.Tools {
				names = append(names, tool.Name)

				if want, ok := tt.readOnly[tool.Name]; ok {
					s.Equal(want, tool.Annotations.ReadOnlyHint, tool.Name)
				}
			}

			s.ElementsMatch(tt.want, names)
		})
	}
}

func TestRegisterPublicTestSuite(t *testing.T) {
	suite.Run(t, new(RegisterPublicTestSuite))
}
```

Run: `go test ./pkg/mcp/internal/tools/` → FAIL, `undefined: tools.Register`.

- [ ] **Step 5: Write `register.go`**

```go
package tools

import (
	"fmt"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// handlers holds what every tool shares.
type handlers struct {
	client Client
	// device is taken by any tool that reaches the pedal, so two calls
	// never claim the editor interface at once.
	device chan struct{}
}

// Register adds tonestack's tools to a server.
//
// The tools that write to a pedal are added only when allowWrites is true. A
// device has no undo, and whoever starts the server decides.
func Register(
	s *gomcp.Server,
	c Client,
	allowWrites bool,
) {
	h := &handlers{client: c, device: make(chan struct{}, 1)}

	gomcp.AddTool(s, &gomcp.Tool{
		Name:        "catalog_search",
		Description: "Find blocks the device models, by name, real-world gear, category or instrument. Use this before naming any model: a model it does not find does not exist.",
		Annotations: readOnly(),
	}, h.catalogSearch)
	gomcp.AddTool(s, &gomcp.Tool{
		Name:        "catalog_block",
		Description: "One block's parameters, their ranges and defaults, and its DSP cost.",
		Annotations: readOnly(),
	}, h.catalogBlock)
	gomcp.AddTool(s, &gomcp.Tool{
		Name:        "corpus_model",
		Description: "How players set one model across measured presets: median and quartiles per parameter. A narrow spread is consensus; a wide one is taste.",
		Annotations: readOnly(),
	}, h.corpusModel)
	gomcp.AddTool(s, &gomcp.Tool{
		Name:        "rigs_list",
		Description: "The rigs that ship with tonestack.",
		Annotations: readOnly(),
	}, h.rigsList)
	gomcp.AddTool(s, &gomcp.Tool{
		Name:        "rig_show",
		Description: "One shipped rig, and the rigs that extend it.",
		Annotations: readOnly(),
	}, h.rigShow)
	gomcp.AddTool(s, &gomcp.Tool{
		Name:        "preset_build",
		Description: "Build a .hlx from a shipped rig or a rig file. Read what it added and what each character word moved before putting it on a pedal.",
		Annotations: &gomcp.ToolAnnotations{OpenWorldHint: new(false)},
	}, h.presetBuild)
}

// readOnly marks a tool that changes nothing anywhere.
func readOnly() *gomcp.ToolAnnotations {
	return &gomcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: new(false)}
}

// said is the one line of text beside a tool's structured answer, for an agent
// that reads text rather than structure.
func said(
	format string,
	args ...any,
) *gomcp.CallToolResult {
	return &gomcp.CallToolResult{
		Content: []gomcp.Content{&gomcp.TextContent{Text: fmt.Sprintf(format, args...)}},
	}
}
```

`allowWrites` is unused until Task 3. If the linter flags it, name it `_` for
now and restore the name in Task 3.

- [ ] **Step 6: Write `offline.go`**

```go
package tools

import (
	"context"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/retr0h/tonestack/pkg/sdk"
	"github.com/retr0h/tonestack/pkg/sdk/catalog"
)

func (h *handlers) catalogSearch(
	_ context.Context,
	_ *gomcp.CallToolRequest,
	in Search,
) (*gomcp.CallToolResult, sdk.Blocks, error) {
	found, err := h.client.Blocks("", sdk.Filter{
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
	_ context.Context,
	_ *gomcp.CallToolRequest,
	in ID,
) (*gomcp.CallToolResult, catalog.Block, error) {
	block, err := h.client.Block("", in.ID)
	if err != nil {
		return nil, catalog.Block{}, err
	}

	return said("%s is %s", block.ID, block.Name), block, nil
}

func (h *handlers) corpusModel(
	_ context.Context,
	_ *gomcp.CallToolRequest,
	in ID,
) (*gomcp.CallToolResult, Model, error) {
	measured, err := h.client.Measurements(sdk.Corpus{Model: in.ID})
	if err != nil {
		return nil, Model{}, err
	}

	stats := measured.Stats.Models[measured.Model]
	block, _ := measured.Catalog.Block(measured.Model)

	out := Model{Block: block, Uses: stats.Uses, Params: stats.Params}

	return said("%s is used %d times across the measured presets", in.ID, stats.Uses), out, nil
}

func (h *handlers) rigsList(
	_ context.Context,
	_ *gomcp.CallToolRequest,
	_ None,
) (*gomcp.CallToolResult, sdk.Recipes, error) {
	found, err := h.client.Recipes("")
	if err != nil {
		return nil, sdk.Recipes{}, err
	}

	return said("%d rigs ship with tonestack", len(found.Rigs)), found, nil
}

func (h *handlers) rigShow(
	_ context.Context,
	_ *gomcp.CallToolRequest,
	in ID,
) (*gomcp.CallToolResult, sdk.Recipe, error) {
	found, err := h.client.Recipe("", in.ID)
	if err != nil {
		return nil, sdk.Recipe{}, err
	}

	return said("rig %s, extended by %d others", in.ID, len(found.Variants)), found, nil
}

func (h *handlers) presetBuild(
	_ context.Context,
	_ *gomcp.CallToolRequest,
	in Build,
) (*gomcp.CallToolResult, Built, error) {
	switch {
	case in.RecipeID != "" && in.RigPath != "":
		return nil, Built{}, ErrTwoSources
	case in.RecipeID != "":
		made, err := h.client.Build(sdk.Make{RecipeID: in.RecipeID, OutputPath: in.Out})
		if err != nil {
			return nil, Built{}, err
		}

		return said("wrote %s from rig %s", in.Out, in.RecipeID), Built{FromRecipe: &made}, nil
	case in.RigPath != "":
		built, err := h.client.Compile(sdk.Compile{RigPath: in.RigPath, OutputPath: in.Out})
		if err != nil {
			return nil, Built{}, err
		}

		return said("wrote %s from %s", in.Out, in.RigPath), Built{FromRig: &built}, nil
	default:
		return nil, Built{}, ErrNoSource
	}
}
```

- [ ] **Step 7: Write `offline_public_test.go`**

```go
package tools_test

import (
	"errors"
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
}

func (s *OfflinePublicTestSuite) run(tool string, tests []row) {
	for _, tt := range tests {
		s.Run(tt.name, func() {
			if tt.setup != nil {
				tt.setup(s.client)
			}

			res := call(s.T(), connect(s.T(), s.client, false), tool, tt.args)

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
				c.EXPECT().Blocks("", sdk.Filter{Subcategory: "Bass", Search: "SVT"}).
					Return(sdk.Blocks{Total: 547, Matched: []catalog.Block{{ID: "HD2_AmpSVBeastBrt"}}}, nil)
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
				c.EXPECT().Blocks("", sdk.Filter{}).Return(sdk.Blocks{}, errUnreadable)
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
				c.EXPECT().Block("", "HD2_AmpSVBeastBrt").
					Return(catalog.Block{ID: "HD2_AmpSVBeastBrt", Name: "Ampeg SVT Brt"}, nil)
			},
			want: "HD2_AmpSVBeastBrt is Ampeg SVT Brt",
			check: func(s *OfflinePublicTestSuite, res *gomcp.CallToolResult) {
				var got catalog.Block
				structured(s.T(), res, &got)
				s.Equal("Ampeg SVT Brt", got.Name)
			},
		},
		{
			name: "a block it does not",
			args: tools.ID{ID: "HD2_Nope"},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().Block("", "HD2_Nope").Return(catalog.Block{}, errors.New("no block HD2_Nope"))
			},
			want: "no block HD2_Nope",
			err:  true,
		},
	})
}

// TestCorpusModel covers how players set one model.
func (s *OfflinePublicTestSuite) TestCorpusModel() {
	id := catalog.ModelID("HD2_AmpSVBeastBrt")
	cat := &catalog.Catalog{Blocks: map[catalog.ModelID]catalog.Block{id: {ID: id, Name: "Ampeg SVT Brt"}}}

	s.run("corpus_model", []row{
		{
			name: "a model the corpus measured",
			args: tools.ID{ID: string(id)},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().Measurements(sdk.Corpus{Model: string(id)}).Return(sdk.Measured{
					Stats: &corpus.Stats{Models: map[catalog.ModelID]corpus.ModelStats{
						id: {Uses: 26, Params: map[string]corpus.ParamStats{"Treble": {N: 26, Median: 0.85}}},
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
				c.EXPECT().Measurements(sdk.Corpus{Model: "HD2_Nope"}).
					Return(sdk.Measured{}, errors.New("HD2_Nope was not measured"))
			},
			want: "was not measured",
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
				c.EXPECT().Recipes("").Return(sdk.Recipes{Rigs: []rig.Spec{{}, {}}}, nil)
			},
			want: "2 rigs ship",
		},
		{
			name: "rigs that will not read",
			args: tools.None{},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().Recipes("").Return(sdk.Recipes{}, errors.New("rigs unreadable"))
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
				c.EXPECT().Recipe("", "mike-dirnt").
					Return(sdk.Recipe{Variants: []sdk.Variant{{}}}, nil)
			},
			want: "rig mike-dirnt, extended by 1 others",
		},
		{
			name: "a rig that does not",
			args: tools.ID{ID: "nobody"},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().Recipe("", "nobody").Return(sdk.Recipe{}, errors.New("no rig nobody"))
			},
			want: "no rig nobody",
			err:  true,
		},
	})
}

// TestPresetBuild covers building from either source.
func (s *OfflinePublicTestSuite) TestPresetBuild() {
	s.run("preset_build", []row{
		{
			name: "from a shipped rig",
			args: tools.Build{RecipeID: "mike-dirnt", Out: "mike.hlx"},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().Build(sdk.Make{RecipeID: "mike-dirnt", OutputPath: "mike.hlx"}).
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
				c.EXPECT().Compile(sdk.Compile{RigPath: "mine.yaml", OutputPath: "mine.hlx"}).
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
				c.EXPECT().Build(gomock.Any()).Return(sdk.Made{}, errors.New("over budget"))
			},
			want: "over budget",
			err:  true,
		},
		{
			name: "a rig file that will not build",
			args: tools.Build{RigPath: "mine.yaml", Out: "mine.hlx"},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().Compile(gomock.Any()).Return(sdk.Built{}, errors.New("unknown block"))
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

func TestOfflinePublicTestSuite(t *testing.T) {
	suite.Run(t, new(OfflinePublicTestSuite))
}
```

Check before running: `catalog.Catalog` has a `Blocks []Block` field that
`Catalog.Block(id)` searches (read `pkg/sdk/catalog/catalog.go:24`). If the
lookup is built from an index instead, construct the catalog the way
`pkg/sdk/catalog`'s own tests do. Likewise check `rig.Spec{}` and
`sdk.Variant{}` are valid zero values to put in a slice.

- [ ] **Step 8: Run the tests**

Run: `go test -cover ./pkg/mcp/internal/tools/`
Expected: `ok`, coverage 100.0%. If output validation rejects a nil slice or
map in a zero result (the error text names the output schema), give the
fixture a non-nil value in that row rather than changing the handler.

- [ ] **Step 9: Commit**

```bash
git add go.mod go.sum pkg/mcp
git commit -m "feat(mcp): tools that read the catalog and build presets"
```

---

### Task 2: The tools that read the pedal

**Goal:** `devices_list`, `presets_list`, `preset_show`, `preset_export` and
`preset_select` are registered, take slot labels such as `07A`, and hold one
device claim at a time that gives up when the call's context ends.

**Files:**

- Create: `pkg/mcp/internal/tools/device.go`
- Modify: `pkg/mcp/internal/tools/register.go`
- Modify: `pkg/mcp/internal/tools/register_public_test.go`
- Test: `pkg/mcp/internal/tools/device_public_test.go`
- Test: `pkg/mcp/internal/tools/device_test.go`

**Acceptance Criteria:**

- [ ] `ListTools` without writes returns the six offline names plus the five
      in this task.
- [ ] `preset_select` has `ReadOnlyHint: false`, `DestructiveHint: false`,
      `IdempotentHint: true`; the other four have `ReadOnlyHint: true`.
- [ ] A slot label that does not parse is an `IsError` result, and the client
      is never called (gomock fails on an unexpected call).
- [ ] Two concurrent `devices_list` calls never run inside the client at the
      same time: the recorded peak is 1.
- [ ] `claim` with the device held and a cancelled context returns an error
      wrapping `context.Canceled`.
- [ ] Coverage of `pkg/mcp/internal/tools` stays at 100%.

**Verify:**
`go test -race -cover ./pkg/mcp/internal/tools/` → `ok ... coverage: 100.0%`

**Steps:**

- [ ] **Step 1: Write the failing tests**

`device_test.go` (internal, package `tools`):

```go
package tools

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
)

type DeviceTestSuite struct {
	suite.Suite
}

// TestClaim covers waiting for the pedal.
func (s *DeviceTestSuite) TestClaim() {
	tests := []struct {
		name   string
		held   bool
		cancel bool
		err    bool
	}{
		{name: "a free device"},
		{
			// An agent that gives up on a call should not stay queued
			// behind another one that holds the pedal.
			name:   "a held device and a call that gave up",
			held:   true,
			cancel: true,
			err:    true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			h := &handlers{device: make(chan struct{}, 1)}
			if tt.held {
				h.device <- struct{}{}
			}

			ctx, cancel := context.WithCancel(context.Background())
			if tt.cancel {
				cancel()
			} else {
				defer cancel()
			}

			release, err := h.claim(ctx)

			if tt.err {
				s.Require().ErrorIs(err, context.Canceled)
				return
			}

			s.Require().NoError(err)
			release()
			s.Empty(h.device)
		})
	}
}

func TestDeviceTestSuite(t *testing.T) {
	suite.Run(t, new(DeviceTestSuite))
}
```

`device_public_test.go`:

```go
package tools_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/retr0h/tonestack/pkg/mcp/internal/tools"
	"github.com/retr0h/tonestack/pkg/mcp/internal/tools/mocks"
	"github.com/retr0h/tonestack/pkg/sdk"
)

type DevicePublicTestSuite struct {
	suite.Suite
	client *mocks.MockClient
}

func (s *DevicePublicTestSuite) SetupSubTest() {
	s.client = mocks.NewMockClient(gomock.NewController(s.T()))
}

type deviceRow struct {
	name  string
	args  any
	setup func(c *mocks.MockClient)
	want  string
	err   bool
}

func (s *DevicePublicTestSuite) run(tool string, tests []deviceRow) {
	for _, tt := range tests {
		s.Run(tt.name, func() {
			if tt.setup != nil {
				tt.setup(s.client)
			}

			res := call(s.T(), connect(s.T(), s.client, false), tool, tt.args)

			s.Equal(tt.err, res.IsError)
			s.Contains(text(s.T(), res), tt.want)
		})
	}
}

var errHXEdit = errors.New("the editor interface is in use, quit HX Edit")

// TestDevicesList covers what is attached, and one call at a time.
func (s *DevicePublicTestSuite) TestDevicesList() {
	s.run("devices_list", []deviceRow{
		{
			name: "a pedal attached",
			args: tools.None{},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().Devices(gomock.Any()).
					Return(sdk.Attached{Devices: []sdk.Attachment{{Model: "HX Stomp"}}}, nil)
			},
			want: "1 attached",
		},
		{
			name: "a bus that will not answer",
			args: tools.None{},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().Devices(gomock.Any()).Return(sdk.Attached{}, errors.New("bus unavailable"))
			},
			want: "bus unavailable",
			err:  true,
		},
	})

	s.Run("two calls at once", func() {
		var inside, peak atomic.Int32

		s.client.EXPECT().Devices(gomock.Any()).Times(2).DoAndReturn(
			func(context.Context) (sdk.Attached, error) {
				now := inside.Add(1)
				for {
					old := peak.Load()
					if now <= old || peak.CompareAndSwap(old, now) {
						break
					}
				}
				time.Sleep(20 * time.Millisecond)
				inside.Add(-1)

				return sdk.Attached{}, nil
			})

		session := connect(s.T(), s.client, false)

		var wg sync.WaitGroup
		for range 2 {
			wg.Go(func() {
				_, _ = session.CallTool(context.Background(), &gomcp.CallToolParams{
					Name: "devices_list", Arguments: tools.None{},
				})
			})
		}
		wg.Wait()

		s.Equal(int32(1), peak.Load())
	})
}

// TestPresetsList covers reading the setlist.
func (s *DevicePublicTestSuite) TestPresetsList() {
	s.run("presets_list", []deviceRow{
		{
			name: "a setlist",
			args: tools.None{},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().Presets(gomock.Any(), sdk.Where{}).
					Return(sdk.Listing{Slots: []sdk.Held{{}, {}}}, nil)
			},
			want: "of 2 slots",
		},
		{
			name: "HX Edit holding the pedal",
			args: tools.None{},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().Presets(gomock.Any(), sdk.Where{}).Return(sdk.Listing{}, errHXEdit)
			},
			want: "quit HX Edit",
			err:  true,
		},
	})
}

// TestPresetShow covers reading one slot.
func (s *DevicePublicTestSuite) TestPresetShow() {
	s.run("preset_show", []deviceRow{
		{
			name: "a slot by its label",
			args: tools.Slot{Slot: "01B"},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().Preset(gomock.Any(), sdk.Read{Slot: 1}).
					Return(sdk.Reading{Name: "Chunky Monkey"}, nil)
			},
			want: "01B holds Chunky Monkey",
		},
		{
			name: "a label the pedal does not have",
			args: tools.Slot{Slot: "99Z"},
			err:  true,
		},
		{
			name: "HX Edit holding the pedal",
			args: tools.Slot{Slot: "01A"},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().Preset(gomock.Any(), sdk.Read{Slot: 0}).Return(sdk.Reading{}, errHXEdit)
			},
			want: "quit HX Edit",
			err:  true,
		},
	})
}

// TestPresetExport covers writing a slot out.
func (s *DevicePublicTestSuite) TestPresetExport() {
	s.run("preset_export", []deviceRow{
		{
			name: "a slot as a rig",
			args: tools.Export{Slot: "01A", Out: "a.yaml"},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().Export(gomock.Any(), sdk.Export{Slot: 0, OutputPath: "a.yaml"}).
					Return(sdk.Written{Path: "a.yaml"}, nil)
			},
			want: "wrote a.yaml from 01A",
		},
		{
			name: "a label the pedal does not have",
			args: tools.Export{Slot: "nope", Out: "a.yaml"},
			err:  true,
		},
		{
			name: "HX Edit holding the pedal",
			args: tools.Export{Slot: "01A", Out: "a.hlx", As: "hlx"},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().Export(gomock.Any(), sdk.Export{Slot: 0, OutputPath: "a.hlx", As: "hlx"}).
					Return(sdk.Written{}, errHXEdit)
			},
			want: "quit HX Edit",
			err:  true,
		},
	})
}

// TestPresetSelect covers loading a slot.
func (s *DevicePublicTestSuite) TestPresetSelect() {
	s.run("preset_select", []deviceRow{
		{
			name: "a slot by its label",
			args: tools.Slot{Slot: "07A"},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().Select(gomock.Any(), sdk.Read{Slot: 18}).Return(sdk.Change{}, nil)
			},
			want: "loaded 07A",
		},
		{
			name: "a label the pedal does not have",
			args: tools.Slot{Slot: "43A"},
			err:  true,
		},
		{
			name: "HX Edit holding the pedal",
			args: tools.Slot{Slot: "07A"},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().Select(gomock.Any(), gomock.Any()).Return(sdk.Change{}, errHXEdit)
			},
			want: "quit HX Edit",
			err:  true,
		},
	})
}

func TestDevicePublicTestSuite(t *testing.T) {
	suite.Run(t, new(DevicePublicTestSuite))
}
```

Check before running: `07A` is slot 18 only if labels count from `01A` = 0
with three per bank. Confirm with `slot.Label(18)` in `pkg/sdk/slot/slot.go`
and fix the expected number if it differs. Confirm `99Z`, `nope` and `43A`
fail `slot.Value.Set`.

Update `TestRegister` in `register_public_test.go`: add the five names to
`offline` (rename the variable to `reads`), and add to `readOnly`:
`"devices_list": true, "presets_list": true, "preset_show": true, "preset_export": true, "preset_select": false`.
Add one assertion for `preset_select`:

```go
if tool.Name == "preset_select" {
	s.Require().NotNil(tool.Annotations.DestructiveHint)
	s.False(*tool.Annotations.DestructiveHint)
	s.True(tool.Annotations.IdempotentHint)
}
```

Run: `go test ./pkg/mcp/internal/tools/` → FAIL, `h.claim undefined`.

- [ ] **Step 2: Write `device.go`**

```go
package tools

import (
	"context"
	"fmt"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/retr0h/tonestack/pkg/sdk"
	"github.com/retr0h/tonestack/pkg/sdk/slot"
)

// claim takes the pedal for one call, or gives up when the call does.
//
// A USB editor interface serves one session. Two calls claiming it at once
// is the failure docs/protocol.md warns leaves a pedal needing a power cycle.
func (h *handlers) claim(
	ctx context.Context,
) (func(), error) {
	select {
	case h.device <- struct{}{}:
		return func() { <-h.device }, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("waiting for the device: %w", ctx.Err())
	}
}

// slotOf reads a slot the way the pedal labels it.
func slotOf(
	label string,
) (int, error) {
	var n int
	if err := slot.NewValue(&n).Set(label); err != nil {
		return 0, err
	}

	return n, nil
}

func (h *handlers) devicesList(
	ctx context.Context,
	_ *gomcp.CallToolRequest,
	_ None,
) (*gomcp.CallToolResult, sdk.Attached, error) {
	release, err := h.claim(ctx)
	if err != nil {
		return nil, sdk.Attached{}, err
	}
	defer release()

	found, err := h.client.Devices(ctx)
	if err != nil {
		return nil, sdk.Attached{}, err
	}

	return said("%d attached", len(found.Devices)), found, nil
}

func (h *handlers) presetsList(
	ctx context.Context,
	_ *gomcp.CallToolRequest,
	_ None,
) (*gomcp.CallToolResult, sdk.Listing, error) {
	release, err := h.claim(ctx)
	if err != nil {
		return nil, sdk.Listing{}, err
	}
	defer release()

	listing, err := h.client.Presets(ctx, sdk.Where{})
	if err != nil {
		return nil, sdk.Listing{}, err
	}

	return said("%d of %d slots hold a preset", listing.Used(), len(listing.Slots)), listing, nil
}

func (h *handlers) presetShow(
	ctx context.Context,
	_ *gomcp.CallToolRequest,
	in Slot,
) (*gomcp.CallToolResult, Shown, error) {
	n, err := slotOf(in.Slot)
	if err != nil {
		return nil, Shown{}, err
	}

	release, err := h.claim(ctx)
	if err != nil {
		return nil, Shown{}, err
	}
	defer release()

	reading, err := h.client.Preset(ctx, sdk.Read{Slot: n})
	if err != nil {
		return nil, Shown{}, err
	}

	out := Shown{Name: reading.Name, Rig: reading.Rig, Answer: reading.Answer}

	return said("%s holds %s", in.Slot, reading.Name), out, nil
}

func (h *handlers) presetExport(
	ctx context.Context,
	_ *gomcp.CallToolRequest,
	in Export,
) (*gomcp.CallToolResult, sdk.Written, error) {
	n, err := slotOf(in.Slot)
	if err != nil {
		return nil, sdk.Written{}, err
	}

	release, err := h.claim(ctx)
	if err != nil {
		return nil, sdk.Written{}, err
	}
	defer release()

	written, err := h.client.Export(ctx, sdk.Export{Slot: n, OutputPath: in.Out, As: in.As})
	if err != nil {
		return nil, sdk.Written{}, err
	}

	return said("wrote %s from %s", written.Path, in.Slot), written, nil
}

func (h *handlers) presetSelect(
	ctx context.Context,
	_ *gomcp.CallToolRequest,
	in Slot,
) (*gomcp.CallToolResult, sdk.Change, error) {
	n, err := slotOf(in.Slot)
	if err != nil {
		return nil, sdk.Change{}, err
	}

	release, err := h.claim(ctx)
	if err != nil {
		return nil, sdk.Change{}, err
	}
	defer release()

	change, err := h.client.Select(ctx, sdk.Read{Slot: n})
	if err != nil {
		return nil, sdk.Change{}, err
	}

	return said("loaded %s", in.Slot), change, nil
}
```

Coverage note: the `claim` error branch inside each handler is reachable only
with a cancelled per-call context while another call holds the pedal. If
coverage shows those five branches uncovered, add a row per handler to the
internal `device_test.go` that holds `h.device`, cancels the context, and calls
the handler method directly, asserting `ErrorIs(err, context.Canceled)`. Name
the method `TestHandlersGiveUp` and make it one table over the five handlers.

- [ ] **Step 3: Register the five tools** (in `Register`, after `preset_build`)

```go
gomcp.AddTool(s, &gomcp.Tool{
	Name:        "devices_list",
	Description: "The Line 6 Helix hardware attached over USB. HX Edit must be quit for any tool that reaches the pedal.",
	Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
}, h.devicesList)
gomcp.AddTool(s, &gomcp.Tool{
	Name:        "presets_list",
	Description: "Every slot on the attached pedal and what it holds.",
	Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
}, h.presetsList)
gomcp.AddTool(s, &gomcp.Tool{
	Name:        "preset_show",
	Description: "One slot on the pedal, read back as a rig.",
	Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
}, h.presetShow)
gomcp.AddTool(s, &gomcp.Tool{
	Name:        "preset_export",
	Description: "Write one slot to a file: a rig by default, or the device's own .hlx with as=hlx.",
	Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
}, h.presetExport)
gomcp.AddTool(s, &gomcp.Tool{
	Name:        "preset_select",
	Description: "Load a slot on the pedal, as pressing its footswitch does. Changes nothing stored.",
	Annotations: &gomcp.ToolAnnotations{DestructiveHint: new(false), IdempotentHint: true},
}, h.presetSelect)
```

`preset_export` writes a local file but nothing on the pedal or in the
setlist, which is why it is marked read-only here; the spec's table says the
same.

- [ ] **Step 4: Run the tests**

Run: `go test -race -cover ./pkg/mcp/internal/tools/`
Expected: `ok`, coverage 100.0%.

- [ ] **Step 5: Commit**

```bash
git add pkg/mcp
git commit -m "feat(mcp): tools that read the pedal, one call at a time"
```

---

### Task 3: The tools that write to the pedal, behind `allowWrites`

**Goal:** `preset_import`, `presets_copy` and `presets_swap` are registered only
when `allowWrites` is true, marked destructive, and share the device claim.

**Files:**

- Create: `pkg/mcp/internal/tools/writes.go`
- Modify: `pkg/mcp/internal/tools/register.go`
- Modify: `pkg/mcp/internal/tools/register_public_test.go`
- Test: `pkg/mcp/internal/tools/writes_public_test.go`

**Acceptance Criteria:**

- [ ] `ListTools` without writes has none of the three names; with writes it
      has all 14 tools.
- [ ] Each of the three has `DestructiveHint` set to `true`.
- [ ] Bad `slot`, `from` or `to` labels are `IsError` with no client call.
- [ ] Coverage of `pkg/mcp/internal/tools` stays at 100%.

**Verify:**
`go test -race -cover ./pkg/mcp/internal/tools/` → `ok ... coverage: 100.0%`

**Steps:**

- [ ] **Step 1: Write the failing tests**

Add a row to `TestRegister`:

```go
{
	name:        "with writes",
	allowWrites: true,
	want:        append(slices.Clone(reads), "preset_import", "presets_copy", "presets_swap"),
	readOnly:    map[string]bool{"preset_import": false, "presets_copy": false, "presets_swap": false},
},
```

and inside the loop over tools:

```go
switch tool.Name {
case "preset_import", "presets_copy", "presets_swap":
	s.Require().NotNil(tool.Annotations.DestructiveHint, tool.Name)
	s.True(*tool.Annotations.DestructiveHint, tool.Name)
}
```

Import `slices`. The `readOnly` map for the existing "without writes" row
stays as Task 2 left it.

`writes_public_test.go`:

```go
package tools_test

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/retr0h/tonestack/pkg/mcp/internal/tools"
	"github.com/retr0h/tonestack/pkg/mcp/internal/tools/mocks"
	"github.com/retr0h/tonestack/pkg/sdk"
)

type WritesPublicTestSuite struct {
	suite.Suite
	client *mocks.MockClient
}

func (s *WritesPublicTestSuite) SetupSubTest() {
	s.client = mocks.NewMockClient(gomock.NewController(s.T()))
}

func (s *WritesPublicTestSuite) run(tool string, tests []deviceRow) {
	for _, tt := range tests {
		s.Run(tt.name, func() {
			if tt.setup != nil {
				tt.setup(s.client)
			}

			res := call(s.T(), connect(s.T(), s.client, true), tool, tt.args)

			s.Equal(tt.err, res.IsError)
			s.Contains(text(s.T(), res), tt.want)
		})
	}
}

// TestPresetImport covers putting a file into a slot.
func (s *WritesPublicTestSuite) TestPresetImport() {
	s.run("preset_import", []deviceRow{
		{
			name: "a preset into a slot",
			args: tools.Put{Preset: "mike.hlx", Slot: "01A"},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().Import(gomock.Any(), sdk.Put{File: "mike.hlx", Slot: 0}).Return(sdk.Change{}, nil)
			},
			want: "put mike.hlx into 01A",
		},
		{
			name: "a label the pedal does not have",
			args: tools.Put{Preset: "mike.hlx", Slot: "nope"},
			err:  true,
		},
		{
			name: "HX Edit holding the pedal",
			args: tools.Put{Preset: "mike.hlx", Slot: "01A"},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().Import(gomock.Any(), gomock.Any()).Return(sdk.Change{}, errHXEdit)
			},
			want: "quit HX Edit",
			err:  true,
		},
	})
}

// TestPresetsCopy covers copying a slot.
func (s *WritesPublicTestSuite) TestPresetsCopy() {
	s.run("presets_copy", []deviceRow{
		{
			name: "one slot onto another",
			args: tools.Move{From: "01A", To: "01B"},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().Copy(gomock.Any(), sdk.Edit{FromSlot: 0, ToSlot: 1}).Return(sdk.Change{}, nil)
			},
			want: "copied 01A to 01B",
		},
		{name: "a source that does not parse", args: tools.Move{From: "nope", To: "01B"}, err: true},
		{name: "a destination that does not parse", args: tools.Move{From: "01A", To: "nope"}, err: true},
		{
			name: "HX Edit holding the pedal",
			args: tools.Move{From: "01A", To: "01B"},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().Copy(gomock.Any(), gomock.Any()).Return(sdk.Change{}, errHXEdit)
			},
			want: "quit HX Edit",
			err:  true,
		},
	})
}

// TestPresetsSwap covers exchanging two slots.
func (s *WritesPublicTestSuite) TestPresetsSwap() {
	s.run("presets_swap", []deviceRow{
		{
			name: "two slots",
			args: tools.Move{From: "01A", To: "01B"},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().Swap(gomock.Any(), sdk.Edit{FromSlot: 0, ToSlot: 1}).Return(sdk.Change{}, nil)
			},
			want: "swapped 01A and 01B",
		},
		{name: "a source that does not parse", args: tools.Move{From: "nope", To: "01B"}, err: true},
		{name: "a destination that does not parse", args: tools.Move{From: "01A", To: "nope"}, err: true},
		{
			name: "HX Edit holding the pedal",
			args: tools.Move{From: "01A", To: "01B"},
			setup: func(c *mocks.MockClient) {
				c.EXPECT().Swap(gomock.Any(), gomock.Any()).Return(sdk.Change{}, errHXEdit)
			},
			want: "quit HX Edit",
			err:  true,
		},
	})
}

func TestWritesPublicTestSuite(t *testing.T) {
	suite.Run(t, new(WritesPublicTestSuite))
}
```

Run: `go test ./pkg/mcp/internal/tools/` → FAIL, tools not found.

- [ ] **Step 2: Write `writes.go`**

```go
package tools

import (
	"context"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/retr0h/tonestack/pkg/sdk"
)

func (h *handlers) presetImport(
	ctx context.Context,
	_ *gomcp.CallToolRequest,
	in Put,
) (*gomcp.CallToolResult, sdk.Change, error) {
	n, err := slotOf(in.Slot)
	if err != nil {
		return nil, sdk.Change{}, err
	}

	release, err := h.claim(ctx)
	if err != nil {
		return nil, sdk.Change{}, err
	}
	defer release()

	change, err := h.client.Import(ctx, sdk.Put{File: in.Preset, Slot: n})
	if err != nil {
		return nil, sdk.Change{}, err
	}

	return said("put %s into %s", in.Preset, in.Slot), change, nil
}

// slotsOf reads both ends of a copy or a swap.
func slotsOf(
	in Move,
) (int, int, error) {
	from, err := slotOf(in.From)
	if err != nil {
		return 0, 0, err
	}

	to, err := slotOf(in.To)
	if err != nil {
		return 0, 0, err
	}

	return from, to, nil
}

func (h *handlers) presetsCopy(
	ctx context.Context,
	_ *gomcp.CallToolRequest,
	in Move,
) (*gomcp.CallToolResult, sdk.Change, error) {
	from, to, err := slotsOf(in)
	if err != nil {
		return nil, sdk.Change{}, err
	}

	release, err := h.claim(ctx)
	if err != nil {
		return nil, sdk.Change{}, err
	}
	defer release()

	change, err := h.client.Copy(ctx, sdk.Edit{FromSlot: from, ToSlot: to})
	if err != nil {
		return nil, sdk.Change{}, err
	}

	return said("copied %s to %s", in.From, in.To), change, nil
}

func (h *handlers) presetsSwap(
	ctx context.Context,
	_ *gomcp.CallToolRequest,
	in Move,
) (*gomcp.CallToolResult, sdk.Change, error) {
	from, to, err := slotsOf(in)
	if err != nil {
		return nil, sdk.Change{}, err
	}

	release, err := h.claim(ctx)
	if err != nil {
		return nil, sdk.Change{}, err
	}
	defer release()

	change, err := h.client.Swap(ctx, sdk.Edit{FromSlot: from, ToSlot: to})
	if err != nil {
		return nil, sdk.Change{}, err
	}

	return said("swapped %s and %s", in.From, in.To), change, nil
}
```

If Task 2 added `TestHandlersGiveUp`, add these three handlers to its table.

- [ ] **Step 3: Register them** (end of `Register`)

```go
if !allowWrites {
	return
}

gomcp.AddTool(s, &gomcp.Tool{
	Name:        "preset_import",
	Description: "Put a .hlx into a slot on the pedal. Whatever the slot held is saved to a file first and then gone from the pedal.",
	Annotations: destructive(),
}, h.presetImport)
gomcp.AddTool(s, &gomcp.Tool{
	Name:        "presets_copy",
	Description: "Copy one slot onto another. The destination's old preset is saved to a file first.",
	Annotations: destructive(),
}, h.presetsCopy)
gomcp.AddTool(s, &gomcp.Tool{
	Name:        "presets_swap",
	Description: "Exchange two slots. Both are saved to files first.",
	Annotations: destructive(),
}, h.presetsSwap)
```

```go
// destructive marks a tool that overwrites what a pedal holds.
func destructive() *gomcp.ToolAnnotations {
	return &gomcp.ToolAnnotations{DestructiveHint: new(true)}
}
```

- [ ] **Step 4: Run the tests**

Run: `go test -race -cover ./pkg/mcp/internal/tools/`
Expected: `ok`, coverage 100.0%.

- [ ] **Step 5: Commit**

```bash
git add pkg/mcp
git commit -m "feat(mcp): tools that write the pedal, only when allowed"
```

---

### Task 4: `pkg/mcp`, and holding it to `pkg/sdk`

**Goal:** `mcp.New(client, Options)` builds a server with tonestack's tools and
instructions, `Run` serves it over stdio until its context ends, and
`main_test.go` fails if `pkg/mcp` reaches beyond `pkg/mcp` and `pkg/sdk`.

**Files:**

- Create: `pkg/mcp/mcp.go`
- Test: `pkg/mcp/mcp_public_test.go`
- Modify: `main_test.go`

**Acceptance Criteria:**

- [ ] A client session over `Serve` with a real `sdk.New()` calls
      `catalog_search` with `search: "SVT"` and gets at least one match from
      the built-in catalog.
- [ ] The session's initialize result carries non-empty instructions naming
      `catalog_search`.
- [ ] `Options{}` offers 11 tools; `Options{AllowWrites: true}` offers 14.
- [ ] `Serve` returns an error wrapping `context.Canceled` after its context is
      cancelled.
- [ ] `Run` with a cancelled context returns within two seconds.
- [ ] `TestTheMCPStandsAlone` passes, and `TestTheCLIStandsAlone` passes with
      `cmd` allowed to reach `pkg/mcp`.

**Verify:**
`go test -cover ./pkg/mcp/ && go test -run 'TestMainTestSuite' .` → both `ok`,
`pkg/mcp` coverage 100.0%

**Steps:**

- [ ] **Step 1: Write the failing test** (`pkg/mcp/mcp_public_test.go`)

```go
package mcp_test

import (
	"context"
	"testing"
	"time"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/suite"

	"github.com/retr0h/tonestack/pkg/mcp"
	"github.com/retr0h/tonestack/pkg/sdk"
)

type MCPPublicTestSuite struct {
	suite.Suite
}

// TestServe covers an agent's session with the server.
func (s *MCPPublicTestSuite) TestServe() {
	tests := []struct {
		name  string
		opts  mcp.Options
		tools int
	}{
		{name: "without writes", opts: mcp.Options{Version: "1.2.3"}, tools: 11},
		{name: "with writes", opts: mcp.Options{AllowWrites: true}, tools: 14},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			serverEnd, clientEnd := gomcp.NewInMemoryTransports()
			ctx, cancel := context.WithCancel(context.Background())

			served := make(chan error, 1)
			go func() { served <- mcp.New(sdk.New(), tt.opts).Serve(ctx, serverEnd) }()

			session, err := gomcp.NewClient(
				&gomcp.Implementation{Name: "test", Version: "test"}, nil,
			).Connect(context.Background(), clientEnd, nil)
			s.Require().NoError(err)

			s.Contains(session.InitializeResult().Instructions, "catalog_search")

			listed, err := session.ListTools(context.Background(), nil)
			s.Require().NoError(err)
			s.Len(listed.Tools, tt.tools)

			res, err := session.CallTool(context.Background(), &gomcp.CallToolParams{
				Name:      "catalog_search",
				Arguments: map[string]string{"search": "SVT"},
			})
			s.Require().NoError(err)
			s.False(res.IsError)
			s.NotContains(res.Content[0].(*gomcp.TextContent).Text, "0 of")

			cancel()
			s.ErrorIs(<-served, context.Canceled)
		})
	}
}

// TestRun covers stdio, which ends when the command's context does.
func (s *MCPPublicTestSuite) TestRun() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan struct{})
	go func() {
		_ = mcp.New(sdk.New(), mcp.Options{}).Run(ctx)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		s.Fail("Run did not return after its context ended")
	}
}

func TestMCPPublicTestSuite(t *testing.T) {
	suite.Run(t, new(MCPPublicTestSuite))
}
```

Check before running: `ClientSession.InitializeResult()` exists in v1.7.0
(`go doc github.com/modelcontextprotocol/go-sdk/mcp.ClientSession.InitializeResult`).
If it does not, read the instructions from the initialize result the library
does expose, found with `go doc -all .../mcp | grep -n Initialize`.

Run: `go test ./pkg/mcp/` → FAIL, `undefined: mcp.New`.

- [ ] **Step 2: Write `pkg/mcp/mcp.go`**

```go
// Package mcp serves tonestack's operations to an agent over the Model Context
// Protocol.
//
// It reaches pkg/sdk and nothing else in this module, so it can leave for a
// repository of its own the way pkg/cli can.
package mcp

import (
	"context"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/retr0h/tonestack/pkg/mcp/internal/tools"
	"github.com/retr0h/tonestack/pkg/sdk"
)

// instructions are what a connecting agent is told before its first call.
const instructions = `tonestack builds Line 6 Helix presets from rigs, and reads and writes the pedal.

Use catalog_search before naming any model. If it does not find the gear, the
device does not model it: say so, and never invent a model identifier.

Build with preset_build, then read what it added and what each character word
moved before putting the preset on a pedal. Quit HX Edit before any tool that
reaches the pedal.

A preset that builds, a preset HX Edit imports, and a preset the hardware loads
are three different claims. Say which one you have.`

// Options say how the server runs.
type Options struct {
	// Version is what the server reports itself as. Empty reports "dev".
	Version string
	// AllowWrites offers the tools that overwrite what a pedal holds.
	AllowWrites bool
}

// Server is tonestack's MCP server.
type Server struct {
	server *gomcp.Server
}

// New builds a server whose tools call client.
func New(
	client *sdk.Client,
	opts Options,
) *Server {
	version := opts.Version
	if version == "" {
		version = "dev"
	}

	s := gomcp.NewServer(
		&gomcp.Implementation{Name: "tonestack", Version: version},
		&gomcp.ServerOptions{Instructions: instructions},
	)
	tools.Register(s, client, opts.AllowWrites)

	return &Server{server: s}
}

// Run serves over stdin and stdout until ctx ends or the agent disconnects.
func (s *Server) Run(
	ctx context.Context,
) error {
	return s.Serve(ctx, &gomcp.StdioTransport{})
}

// Serve serves over any transport until ctx ends or the agent disconnects.
func (s *Server) Serve(
	ctx context.Context,
	t gomcp.Transport,
) error {
	return s.server.Run(ctx, t)
}
```

- [ ] **Step 3: Hold the package to `pkg/sdk`** (`main_test.go`)

Add after `TestTheCLIStandsAlone`:

```go
// TestTheMCPStandsAlone holds the MCP server to the SDK, so it can leave for a
// tonestack-mcp repository without taking the CLI with it.
func (s *MainTestSuite) TestTheMCPStandsAlone() {
	out, err := exec.Command("go", "list", "-deps", "./pkg/mcp/...").Output()
	s.Require().NoError(err)

	for _, dep := range strings.Fields(string(out)) {
		if !strings.HasPrefix(dep, mod) {
			continue
		}

		switch {
		case strings.HasPrefix(dep, mod+"pkg/mcp"):
		case strings.HasPrefix(dep, mod+"pkg/sdk"):
		default:
			s.Require().Fail("reaches too far", "pkg/mcp reaches %s, which a tonestack-mcp would not have", dep)
		}
	}
}
```

In `TestTheCLIStandsAlone`, add a case so `cmd` may reach the server the way a
future `tonestack-cli` would import `tonestack-mcp`:

```go
case strings.HasPrefix(dep, mod+"pkg/mcp"):
```

The CLI test's package list includes `./pkg/cli/...`, which never imports
`pkg/mcp`. The new case only lets `cmd` through.

- [ ] **Step 4: Run the tests**

Run: `go test -cover ./pkg/mcp/ && go test -run TestMainTestSuite .`
Expected: both `ok`; `pkg/mcp` coverage 100.0%.

- [ ] **Step 5: Commit**

```bash
git add pkg/mcp/mcp.go pkg/mcp/mcp_public_test.go main_test.go
git commit -m "feat(mcp): a server that stands on the SDK alone"
```

---

### Task 5: `tonestack mcp`, the docs, and the pull request

**Goal:** `tonestack mcp [--allow-writes]` runs the server with the signal
context, the docs say how to connect an agent, and a pull request is open with
the gate passed.

**Files:**

- Modify: `cmd/root.go` (add `var version = "dev"`)
- Create: `cmd/mcp.go`
- Modify: `docs/commands.md` (regenerated)
- Modify: `docs/workflows.md`
- Modify: `README.md`
- Modify: `CONTRIBUTING.md`
- Modify: `docs/superpowers/specs/2026-09-13-an-mcp-server-design.md`

**Acceptance Criteria:**

- [ ] `go run . mcp --help` shows `--allow-writes`.
- [ ] Piping an `initialize` request into `go run . mcp` prints a JSON-RPC
      response with `"name":"tonestack"` on stdout.
- [ ] `docs/commands.md` has a `tonestack mcp` section, written by `just ready`.
- [ ] README Features has an MCP row; CONTRIBUTING's project structure, import
      table and "Where the SDK ends" name `pkg/mcp`; workflows.md has "Use it
      from an agent"; the spec's status reads `accepted, built`.
- [ ] `mise exec -- just ready` and `mise exec -- just test` pass, total
      coverage at or above 99%.
- [ ] A pull request is open against `main`.

**Verify:** `mise exec -- just test` → exit 0 with coverage ≥ 99%

**Steps:**

- [ ] **Step 1: Confirm the branch**

Run: `git branch --show-current` → `feat/mcp-server`, holding the commits from
Tasks 1-4.

- [ ] **Step 2: Give the version a home** (`cmd/root.go`, below the imports)

```go
// version is set at release by goreleaser's -X cmd.version, and says "dev"
// for anything built another way.
var version = "dev"
```

- [ ] **Step 3: Write `cmd/mcp.go`** (licence header first)

```go
package cmd

import (
	"context"
	"errors"

	"github.com/spf13/cobra"

	"github.com/retr0h/tonestack/pkg/mcp"
	"github.com/retr0h/tonestack/pkg/sdk"
)

var mcpAllowWrites bool

// mcpCmd represents the mcp command.
var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Serve tonestack to an agent over MCP",
	Long: `Serve the catalog, the corpus, the shipped rigs, building presets and the
attached pedal to an agent over the Model Context Protocol, on stdin and stdout.

An agent starts this itself. For Claude Code:

  claude mcp add tonestack -- tonestack mcp

Tools that overwrite what a pedal holds (import, copy, swap) are offered only
with --allow-writes. Each still saves what it replaces to a file first.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		err := mcp.New(sdk.New(), mcp.Options{
			Version:     version,
			AllowWrites: mcpAllowWrites,
		}).Run(cmd.Context())

		// Ctrl-C and SIGTERM are how this is meant to stop, not a failure.
		if errors.Is(err, context.Canceled) {
			return nil
		}

		return err
	},
}

func init() {
	rootCmd.AddCommand(mcpCmd)

	mcpCmd.Flags().BoolVar(&mcpAllowWrites, "allow-writes", false,
		"offer the tools that overwrite slots on the pedal")
}
```

- [ ] **Step 4: Check it by hand**

```bash
go run . mcp --help
printf '%s\n' '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"t","version":"0"}}}' | go run . mcp
```

Expected: help lists `--allow-writes`; the second prints one JSON line with
`"serverInfo":{"name":"tonestack","version":"dev"}` and exits when stdin
closes, with status 0.

- [ ] **Step 5: Docs**

`docs/workflows.md`: add before `## For agents`:

````markdown
## Use it from an agent

`tonestack mcp` gives an agent the same operations as tools, with typed results
instead of text to parse. Claude Code starts it for you once it is added:

```bash
claude mcp add tonestack -- tonestack mcp
```

That offers everything except writing to a pedal. To let the agent import, copy
and swap slots, add it with `tonestack mcp --allow-writes` instead. Each write
still saves what it replaces to a file first. Selecting a slot works either way.

Quit HX Edit before asking for anything that reaches the pedal.
````

`README.md` Features: add after the "Talks to your Helix" row:

```markdown
| [MCP server](docs/workflows.md#use-it-from-an-agent) | `tonestack mcp` gives an agent the catalog, the corpus, building and the pedal as tools with typed results. It cannot write to a pedal unless you start it with `--allow-writes` |
```

`CONTRIBUTING.md`:

- Project structure block, after the `pkg/cli/internal/` line:
  ```text
  pkg/mcp/             the MCP server an agent runs: New and Run
  pkg/mcp/internal/    one handler per tool. Invisible outside pkg/mcp.
  ```
- "What to import" table, after the `cli` row:
  `| serve the operations to an agent over MCP | `mcp` |`
- "Where the SDK ends", last paragraph: change "and the CLI reaches only `cmd`,
  `pkg/cli` and `pkg/sdk`." to "the CLI reaches only `cmd`, `pkg/cli`,
  `pkg/mcp` and `pkg/sdk`, and the MCP server only `pkg/mcp` and `pkg/sdk`."

Spec: change `**Status:** proposed\` to `**Status:** accepted, built\`.

Put each changed markdown file through the `unslop` skill, then let
`just ready` format them.

- [ ] **Step 6: Gate**

```bash
mise exec -- just ready
mise exec -- just test
```

Expected: both exit 0. `just ready` regenerates `docs/commands.md` with a
`tonestack mcp` section. If coverage is under 99%, find the uncovered lines
with `go tool cover -func` on the profile `just test` writes and add test rows;
do not touch coverage configuration.

- [ ] **Step 7: Commit and open the pull request**

```bash
git add -A
git commit -m "feat: tonestack mcp

Serve the catalog, corpus, shipped rigs, building and the pedal to
an agent over MCP on stdio. Tools that overwrite slots appear only
with --allow-writes and are marked destructive.

🤖 Generated with [Claude Code](https://claude.ai/code)

Co-Authored-By: Claude <noreply@anthropic.com>"
git merge-base --is-ancestor origin/main HEAD && git push -u origin feat/mcp-server
gh pr create --title "feat: tonestack mcp" --body-file <body>
```

The body says what shipped, the two deviations from the spec and why, that
nothing was run against a pedal through MCP (only the mocked client and the
built-in catalog), and ends with
`🤖 Generated with [Claude Code](https://claude.com/claude-code)`.
