# An MCP server

**Status:** proposed\
**Scope:** `pkg/mcp`, `pkg/mcp/internal`, and a `tonestack mcp` command\
**Builds on:**
[The SDK is the library](2026-09-10-the-sdk-is-the-library-design.md)

## The problem

An agent drives tonestack today by running shell commands and reading what they
print. That works, and it is the slow way: every call starts a process, parses
text meant for a person, and loses the structure the SDK already returns.

The Model Context Protocol lets an agent call tonestack's operations as tools
with typed inputs and structured results. It is what makes "your agent tunes it"
a direct loop: read the pedal, build a rig, put it in a slot, listen, change one
line.

## Where it lives

The same shape as the CLI, so it can leave for a `tonestack-mcp` repository the
way `pkg/cli` can leave for `tonestack-cli`:

| path               | holds                                                                                 |
| ------------------ | ------------------------------------------------------------------------------------- |
| `pkg/mcp`          | the public part: `mcp.New(client *sdk.Client, opts Options)` and `(*Server).Run(ctx)` |
| `pkg/mcp/internal` | one handler per tool, calling only `pkg/sdk`                                          |
| `cmd/mcp.go`       | `tonestack mcp`, which runs the server over stdio                                     |

`TestTheMCPStandsAlone` holds `pkg/mcp` to reaching nothing in this module
beyond `pkg/mcp` and `pkg/sdk`, the way `TestTheCLIStandsAlone` holds the CLI.
`cmd` may reach `pkg/mcp`, as a future `tonestack-cli` would import
`tonestack-mcp`. Nothing here depends on `pkg/cli`: the server returns data, not
text laid out for a terminal.

The protocol comes from the official Go library,
`github.com/modelcontextprotocol/go-sdk`, stable at v1.

## Start, stop and cancel

- **Start.** An agent such as Claude Code runs `tonestack mcp` as a subprocess
  and speaks to it over stdin and stdout.
- **Stop.** `Run` returns when its context ends. `tonestack mcp` passes the same
  signal context `Execute` already builds, so the server stops when stdin closes
  or on Ctrl-C and SIGTERM.
- **Cancel.** Each tool call gets its own context from the MCP session. When the
  agent cancels a call, or the session ends mid-call, that context ends, and
  every SDK call that reaches the pedal already takes it. A cancelled session
  still releases the USB interface the way
  [the rules that keep a device alive](../../protocol.md#rules-that-keep-a-device-alive)
  require.
- **One device call at a time.** A lock around the tools that touch the pedal,
  so two calls never claim the editor interface at once.

## Tools

| tool                                           | does                                                                            | reaches the pedal | writes  |
| ---------------------------------------------- | ------------------------------------------------------------------------------- | ----------------- | ------- |
| `catalog_search`                               | find blocks by name, gear, category or instrument                               | no                | no      |
| `catalog_block`                                | one block's parameters, ranges and DSP cost                                     | no                | no      |
| `corpus_model`                                 | how players set one model                                                       | no                | no      |
| `rigs_list`, `rig_show`                        | the rigs that ship, and one of them                                             | no                | no      |
| `preset_build`                                 | build a `.hlx` from a rig or recipe, with what was added and what each word did | no                | a file  |
| `devices_list`                                 | what is attached                                                                | yes               | no      |
| `presets_list`, `preset_show`, `preset_export` | read the setlist, a slot, or a slot as a rig                                    | yes               | no      |
| `preset_import`                                | put a `.hlx` into a slot                                                        | yes               | **yes** |
| `presets_copy`, `presets_swap`                 | copy or exchange slots                                                          | yes               | **yes** |
| `preset_select`                                | load a slot, as a footswitch does                                               | yes               | no      |

Names are what an agent sees, so they say the operation, not the command that
does it from a terminal.

### Writing to a pedal is opt-in

`tonestack mcp` does not offer the three tools that write to a device.
`tonestack mcp --allow-writes` does. Reading, building and selecting work either
way.

A device has no undo, and a server an agent can reach runs with less attention
than a person typing a command. Every write already saves what it replaces
first; this is the second guard, and it is the person starting the server who
chooses. The writing tools are also marked destructive in their MCP annotations,
so a client that asks before a destructive call does.

## Results

Each tool returns the SDK's own result type as structured content, from which
the library generates the output schema, and one line of text saying what
happened. An agent that reads the structure gets every field; one that reads
text still learns the outcome.

Errors say what went wrong the way the CLI's do: HX Edit holding the interface
tells the agent to have HX Edit quit, and a slot label that does not parse says
so.

## Testing

Handlers run against the library's in-memory transport, through a real client
session, with the SDK's test options standing in for the USB bus. No pedal is
needed and the coverage gate stays where it is.

The device round trip behind the `device` build tag stays the hardware check.

## Not in this

- **HTTP.** stdio is what a local agent uses. A hosted server is a separate
  decision with authentication in it.
- **Recording tuning rounds** (`mutations`) as a tool. Useful, and it needs its
  own design for what an agent may write into a rig.
- **Snapshots** and **footswitch assignments**, which are deferred elsewhere.

## Afterwards

The README's Features table gains a row for it once it ships.
