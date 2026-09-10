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
solves it exactly, and the rule is not the one people remember. It is not that
`internal` must sit at the top. It is that a package whose path contains an
element named `internal` may be imported only by packages rooted at that
directory's parent:

| package                     | importable from      |
| --------------------------- | -------------------- |
| `internal/cli`              | the whole module     |
| `pkg/sdk/internal/x`        | `pkg/sdk/...`        |
| `pkg/sdk/device/internal/x` | `pkg/sdk/device/...` |

Top level is not a special case. It is the one where the parent happens to be
the module root, which is why root `internal/` is the widest fence there is, not
the only one. The library gets a private half the compiler keeps private, and it
travels with the library.

Extraction becomes one directory move.

So root `internal/` stops being where implementation goes and becomes what its
scope actually says: what belongs to no package here. Rendering, and the
generators. Everything a package owns lives in that package's own `internal`.

## The names

```text
pkg/sdk/              the library. One import root.
  client.go           Client, Options, and the operations
  alias.go            the answer types, named here and declared in result
  result/             what every operation answers with
  internal/           shared private half. Invisible outside pkg/sdk.
    slots/  presets/  recipes/   the flows, once they no longer render
    compile/                     a rig to a preset and back
    setlist/                     .hls and .hlb
  rig/                RigSpec: the format a TUI and a service both need
    internal/gen/     generated types, once rig owns the ones a caller holds
  device/             USB: discovery, session, transport   (was pkg/sdk)
    internal/wire/    framing and the device's own document (was pkg/sdk/wire)
    internal/editor/  a device's document to a chain and back
  catalog/            what a device can do
  corpus/             what real presets say, measured
  chain/              a resolved chain
  preset/             .hlx
  slot/               addressing, 01A to 42C

internal/             belongs to no package here
  cli/                rendering: tables, colour, the visual language
  catalogen/  corpusgen/  specdoc/    generators, run by go generate

cmd/                  cobra wiring: flags to a Client call to a renderer
```

`device` rather than `usb` because a session is not a transport. `wire` stays
under it, now as its private half: it is the device's own vocabulary and means
nothing without one. `editor` joins it there for the same reason. It translates
the document a device hands back, so it is that device's business and nobody
else's.

### How far down the private half goes

The rule is that a package's implementation lives in *its* `internal`, and the
measurement says where that lands. Counting who imports what, with the flows
already moved in:

| package                        | imported by                  |
| ------------------------------ | ---------------------------- |
| `catalog`, `chain`, `preset`   | several domains, plus `cli`  |
| `corpus`, `rig`, `slot`        | one domain, plus a generator |
| `rig/gen`                      | `rig`, `compile`, `editor`   |
| `device/wire`                  | `device`, `editor`           |
| `compile`, `editor`, `setlist` | nothing inside the SDK       |

Two of those cannot go where they belong yet, and doing it found the table half
wrong about which two.

`wire` under `device/internal` is unreachable from `editor`, which this record
said moving `editor` under `device` would fix. It would not: `slots` reads the
wire too, and `slots` is not under `device` either. There is no arrangement
where `wire` is private to one domain, because two domains need it. It sits in
the shared `pkg/sdk/internal` and that is not a compromise — it is the tightest
fence that exists for a package with two consumers.

`rig/gen` is blocked, but not by `compile` as this record guessed. It is
`internal/cli` that holds `gen.RigSpec`, `gen.Technique` and eight more
generated types directly, which is precisely what stage 5 is for. `gen` stays
public until the renderer stops naming it.

A shared private half is still private. The only thing a tighter fence would buy
is saying *which* half of the library a package belongs to, and where two halves
need it there is no such answer to give.

What stays public is what a consumer holds: the `Client`, and the types it hands
back. Roughly three thousand lines of the thirteen thousand there are today.
Everything else is how, not what.

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

### Where the answers live

The first flow to move found the other thing this record had wrong. The
operations build the result types and the Client hands them back, so a type
declared in `pkg/sdk` makes `pkg/sdk/internal/…` import it, and a type declared
in the flow makes `pkg/sdk` import that. Either way the two import each other,
and Go refuses.

So the types live in `pkg/sdk/result`, which depends on neither and which both
depend on. Nobody names it: every type in it is aliased into `sdk`, so a caller
writes `sdk.Listing` and never sees the seam. Aliases rather than wrappers,
because `sdk.Listing` and `result.Listing` being the same type is what keeps the
seam free.

## Order of work

Each stage lands on its own and leaves the tree working.

1. **Move the packages.** Import paths change and nothing else. Mechanical,
   large, and reviewable precisely because it carries no other change. Done.

2. **Turn the flows inside out.** Each operation returns a value; the printing
   moves to `internal/cli`; `cmd/` calls both. One command at a time, starting
   with `presets list`, because it has a device path, a file path and a table.

3. **Give the library its internal, one flow at a time.** This stage and the
   next are the same stage, and it took trying it to find out.

   A flow cannot stay in root `internal/` and be called from `pkg/sdk`:
   `main_test.go` refuses it, and it is the weld this is removing. A flow cannot
   move to `pkg/sdk/internal/` and still be called from `cmd/`: the *compiler*
   refuses that, which is the whole reason for nested internal. So the moment a
   flow goes private it needs the Client in front of it, and the move and the
   method land together or not at all.

   Done per flow, that is small: move one flow, add its Client method, point its
   command at the Client. Six of them, each leaving the tree working.

   Then the domain packages nothing outside the SDK imports follow: `compile`,
   `setlist`, `editor`, `device/wire`, and `rig/gen`. Six packages, two hundred
   and sixteen exported identifiers, public today for no reason anybody can
   name. Demoting is the direction that has to happen now: promoting a package
   later costs nothing, and demoting one later breaks every caller.

   Stages 2 and 3 were first written the other way round and cannot be. A flow
   that still renders imports `internal/cli`, and moving it under `pkg/` would
   put `pkg/` importing `internal/`. The rendering comes out first; the move is
   what is left once nothing in the flow knows what a colour is.

4. **Finish the front door.** Folded into stage 3 above, because the compiler
   folds it: what is left once every flow has moved is a `Client` that already
   covers all of them, and a `cmd/` importing nothing but `sdk` and the
   renderer.

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
