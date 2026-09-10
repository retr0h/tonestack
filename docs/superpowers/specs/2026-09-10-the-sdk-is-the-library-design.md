# The SDK is the library

**Status:** proposed\
**Supersedes:**
[2026-09-09-where-a-package-belongs-design.md](2026-09-09-where-a-package-belongs-design.md)

## The question, finally stated

Asked four times and answered badly three of them, because two people were using
"SDK" for different things. One meant `pkg/sdk`, the USB transport. The other
meant the library everything rallies around: the types, the operations, the
thing a TUI imports and a service imports and one day another repository
imports.

The second is the right meaning, and once it is the one in use every earlier
objection falls over. "RigSpec does not belong under the SDK" was true of a USB
client and false of a library. A TUI needs `RigSpec` more than it needs USB.

## What is actually wrong today

Not the package boundaries. This:

| package            | exported | that take an `io.Writer` |
| ------------------ | -------- | ------------------------ |
| `internal/slots`   | 32       | 16                       |
| `internal/recipes` | 10       | 3                        |
| `internal/presets` | 3        | 1                        |
| `internal/device`  | 3        | 2                        |

**Every operation prints, and not one returns data.** `ListWith` does not return
the slots, it writes a table. `ShowWith` does not return a rig, it writes YAML.
Every one of them returns `error` and nothing else.

So a TUI cannot be a wrapper without logic. There is nothing to call that hands
back a value, so it would re-implement each flow, and the rule is broken before
the first line of it is written. The same is true of an HTTP handler and of an
MCP tool.

Both projects this protocol work is built on publish that layer: `voidx-client`
in tonepush, `fretwire-commands` in fretwire, and fretwire's `-cli`, `-tauri`,
`-serve` and `-mcp` all sit on it. Ours is `internal/` and welded to rendering.

## The shape

Three layers, and the middle one is the product.

```text
wrappers   cmd/            flags in, values out, no decisions
sdk        pkg/sdk/        types and operations, everything returns data
core       pkg/sdk/internal/   how the operations are done
```

`internal/` at the repository root keeps what only this program needs:
rendering, and the generators that build artefacts.

### Why nested internal

The two requirements look incompatible and are not.

- `pkg/` thin, exposing only what a consumer uses. Implementation elsewhere.
- The SDK lifts out to its own repository as a unit.

If the implementation sat in root `internal/`, `pkg/sdk` would import it, and
lifting the SDK out would leave the implementation behind. Go's nested internal
solves it exactly: `pkg/sdk/internal/x` is importable from `pkg/sdk/...` and
from nowhere else, enforced by the compiler. The library gets a private half
that travels with it.

Extraction becomes one directory move.

## The names

```text
pkg/sdk/              the library. One import root.
  sdk.go              Client, Options, and the operations
  rig/  rig/gen/      RigSpec: the format a TUI and a service both need
  catalog/            what a device can do
  corpus/             what real presets say, measured
  chain/              a resolved chain
  preset/             .hlx
  setlist/            .hls and .hlb
  slot/               addressing, 01A to 42C
  device/             USB: discovery, session, transport   (was pkg/sdk)
  device/wire/        framing and the device's own document (was pkg/sdk/wire)
  compile/            a rig to a preset and back
  editor/             a device's document to a chain and back
  internal/           how the operations are done. Invisible outside pkg/sdk.

internal/             this program, not the library
  cli/                rendering: tables, colour, the visual language
  catalogen/  corpusgen/  specdoc/    generators, run by go generate

cmd/                  cobra wiring: flags to a Client call to a renderer
```

`device` rather than `usb` because a session is not a transport. `wire` stays
under it: it is the device's own vocabulary and means nothing without one.

### Why the domains sit under `sdk` rather than beside it

The argument against is real and was made: if everything under `pkg/` is the
library then `pkg/` is already the SDK, and a level saying so again buys a
longer import path and nothing else. A service that compiles rigs should import
`compile`, not `sdk/compile`.

The argument that wins is that the SDK is expected to leave. Not a directory
move as a figure of speech: an actual second repository, with `cmd/` and the
rendering staying behind. When that happens the thing that moves has to be one
directory, and every import path inside it has to be right on the day it arrives
rather than rewritten on the way. Nesting costs six characters now and saves
rewriting every import in the tree later.

It also gives the front door somewhere to live. `pkg/sdk` holds the Client, so
`import ".../pkg/sdk"` is the one line a wrapper needs, and the domains stay
addressable underneath for anybody who wants one of them on its own.

### Nobody outside should have to hold a generated type

`RigSpec` and everything in it comes out of `oapi-codegen`, and today a caller
writes `riggen.ChainEntry` and lives with whatever the generator called things.
That is a contract nobody agreed to: regenerating can rename a field, and the
consumer finds out at compile time.

osapi solves this and the shape is worth copying exactly. `pkg/sdk/client/gen`
is generated and unexported in effect; `pkg/sdk/client/*_types.go` are
hand-written types the consumer sees; and unexported functions named
`jobDetailFromGen` convert between them. Eleven files do it.

So `pkg/sdk` owns the types a caller touches, and the generated ones stay an
implementation detail of `rig`. Where a generated type is already the right
shape, an alias is enough and costs nothing:

```go
type Rig = riggen.RigSpec
```

Where it is not, a hand-written type and a conversion. The test of which is
whether a consumer would be surprised by the name, not whether writing the
struct out again is tedious.

## What moves, and what does not

| from                                                             | to                                          | why                                                       |
| ---------------------------------------------------------------- | ------------------------------------------- | --------------------------------------------------------- |
| `pkg/rig` and every other format package                         | `pkg/sdk/rig`, and so on                    | one import root that lifts out together                   |
| `pkg/sdk`                                                        | `pkg/sdk/device`                            | the transport is part of the library, not the whole of it |
| `pkg/sdk/wire`                                                   | `pkg/sdk/device/wire`                       | it belongs to the device                                  |
| `internal/slots`, `internal/presets`, `internal/recipes`         | `pkg/sdk/internal/…`, minus rendering       | these are the flows, and the flows are the product        |
| `internal/catalogview`, `internal/corpusview`, `internal/device` | split: flow up, rendering to `internal/cli` | they are half operation and half table                    |
| `internal/cli`                                                   | stays                                       | rendering is this program's, not the library's            |
| `internal/catalogen`, `internal/corpusgen`, `internal/specdoc`   | stay                                        | build-time tools that write files into the repository     |
| `cmd/`                                                           | stays, and shrinks                          | it gains rendering calls and loses everything else        |

Nothing is deleted. Every flow that exists keeps existing; what changes is that
it hands back a value and somebody else prints it.

## The operations

One type wrappers rally around:

```go
// pkg/sdk
type Client struct{ … }

func New(opts ...Option) *Client

func (c *Client) Presets(ctx context.Context, setlist int) ([]Preset, error)
func (c *Client) Preset(ctx context.Context, at slot.Slot) (Rig, error)
func (c *Client) Import(ctx context.Context, in Import) (Imported, error)
func (c *Client) Make(in Make) (Made, error)
```

Every one returns a value. The CLI prints it, a TUI draws it, a handler
serialises it, and none of them decides anything.

Where a device is needed the Client holds one; where it is not, the same Client
works with no hardware attached, which is what lets a service compile rigs on a
machine that has never seen a Helix.

## Order of work

Each stage lands on its own and leaves the tree working.

1. **Move the packages.** Import paths change and nothing else. Mechanical,
   large, and reviewable precisely because it carries no other change. Done.
2. **Give the library its internal.** The flows move to `pkg/sdk/internal`,
   still writing to an `io.Writer`, so the move is separable from the reshape.
3. **Turn the flows inside out.** Each operation returns a value; the printing
   moves to `internal/cli`; `cmd/` calls both. One command at a time, starting
   with `presets list`, because it has a device path, a file path and a table.
4. **Put the Client in front.** Once the operations return values, `sdk.Client`
   is a thin thing over them, and `cmd/` stops importing anything else.
5. **Own the types.** `pkg/sdk` declares what a caller holds, aliasing the
   generated types where they are already right and converting where they are
   not, so regenerating the contract cannot rename somebody else's field.

## What this does not do

**It does not extract the SDK.** It makes extracting it a directory move rather
than a project. A second module inside one repository buys friction now and
freedom later, and the day somebody asks is the day to do it.

**It does not change behaviour.** Every command does what it did. The tests that
hold the round trip, the corpus and the coverage gate all still apply, and a
stage that broke one of them would be a stage done wrong.

**It does not put rendering in the library.** `internal/cli` stays where it is.
A library that decided what a table looks like would be a library a TUI has to
fight.
