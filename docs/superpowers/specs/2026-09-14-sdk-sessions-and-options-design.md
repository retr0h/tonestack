# SDK sessions and options

**Status:** implemented, in chunks 49.4 to 49.7. The last of them is the `slots`
split and signatures.\
**Scope:** `pkg/sdk`, `pkg/sdk/internal/device`, `pkg/sdk/internal/slots`,
`pkg/mcp/internal/tools`, the `cmd/` presets and devices commands, and
`main_test.go`\
**Builds on:** [An MCP server](2026-09-13-an-mcp-server-design.md) and
[The SDK is the library](2026-09-10-the-sdk-is-the-library-design.md)\
**Review:** the architectural half of the golang-pro review, chunk 49.3. The
error sweep is 49.2 and is not repeated here.

## The problem

**Nothing reads from the pedal between calls.** A read happens only while a call
is waiting for its reply. Nothing reads during the 750ms `settle()` after a
write, while `awaitLoaded` sleeps between polls, or between two operations.
[Rule 2](../../protocol.md#rules-that-keep-a-device-alive) says a read is always
posted. The review marked the harm inferred, not proven, and it is the one rule
the transport breaks by design.

**The SDK cannot be configured.** `Options` is an empty struct. Tests stand in
for hardware by swapping `slots.OpenDevice` and `attached.NewLister`, package
globals, so no two tests that touch a device can run in parallel. Each device
operation claims, handshakes and closes on its own, so an agent that lists,
shows and selects pays for three handshakes. `Blocks`, `Block`, `Measurements`,
`Recipes`, `Recipe`, `Scaffold`, `Build` and `Compile` take no context.
`Export.As` is a bare string although `slots.Format` exists. The library reads
`TONESTACK_USB_DUMP`, `TONESTACK_USB_DEBUG` and `XDG_STATE_HOME` itself.

**Three option structs say the same thing.** `sdk.Edit` is copied into
`slots.EditOptions`, which is copied into device arguments, and the same goes
for `Put`, `Read` and `Export`. Each copy chooses which fields to carry, and
that is where the ignored fields came from: `OutputPath` on a device,
`BackupDir` on a file, `Name`, `As` and `All` on a select. Chunk 49.1 refuses
them. This design removes them.

**`slots` does three jobs.** It holds file flows, device flows and backups. The
backup policy, meaning what gets kept, in which format and where, lives wherever
those structs happened to meet.

**The signature rule is not followed.** CONTRIBUTING says a function with
parameters puts one on each line. Outside generated code, 480 declarations in
production files and 273 in tests put them all on one line, against 221 that
follow the rule. No linter checks it.

## Client options

`Options` becomes unexported, and every setting has a function of its own:

```go
func New(opts ...Option) *Client

func WithCatalog(path string) Option  // a generated catalog instead of the built-in one
func WithStats(path string) Option    // corpus statistics instead of the built-in ones
func WithRecipes(dir string) Option   // rigs to read instead of the ones that ship
func WithBackupDir(dir string) Option // where a slot's old contents go before a write
func WithCapture(w io.Writer) Option  // each device answer a read gets, verbatim
func WithTrace(w io.Writer) Option    // every USB frame in and out
```

These are shown one per line to keep the listing short. The source follows the
signature rule.

With no options, `sdk.New()` uses the built-in catalog and statistics and the
rigs that ship. It captures nothing and traces nothing, and it reaches hardware
over USB. Backups go to `$XDG_STATE_HOME/tonestack/presets`, or to
`~/.local/state/tonestack/presets` when that variable is unset. `New` opens
nothing and cannot fail. The Client opens the catalog on first use, keeps it for
its lifetime, and adds `Client.Catalog(ctx)` so a renderer uses the same one.

**The catalog, the statistics, the rigs and the backup directory describe the
Client, not a call.** So `CatalogPath`, `StatsPath`, `RecipesDir` and
`BackupDir` leave every input struct. `Blocks(catalogPath, f)` becomes
`Blocks(ctx, f)`. The CLI builds its Client inside `RunE` from the flags it
already has, so `--catalog` and `--backup-dir` keep working unchanged.

**The environment is read by `cmd`.** The CLI turns `TONESTACK_USB_DUMP` into
`WithCapture` on a file it opens, and `TONESTACK_USB_DEBUG` into
`WithTrace(os.Stderr)`. The device's `debug` global becomes a `trace` field on
the session. `XDG_STATE_HOME` is the one variable the library still reads, in
one function, because the default location for state is the platform's
convention in the way `os.UserConfigDir` is. A test in `main_test.go` enforces
that function as the only call to `os.Getenv` under `pkg/sdk`.

**Tests stand in for hardware through the Client they build.** `device` gains
one interface covering the bus:

```go
// pkg/sdk/internal/device
type Opener interface {
	List(ctx context.Context) ([]Descriptor, error)
	Open(ctx context.Context) (Editor, error)
}
```

`device.NewUSB(trace)` is the real one. The option that replaces it is not
public, because a caller outside the library cannot build an `Editor`: it
returns `wire` types. It lives in `pkg/sdk/export_test.go` as a setter, which is
the pattern CONTRIBUTING already names:

```go
// pkg/sdk/export_test.go
func WithDevices(d device.Opener) Option
```

Every test builds its own Client with a generated mock, so `t.Parallel()` is
safe. `slots.OpenDevice`, `attached.NewLister`, `sdk.OpenDevice` and
`sdk.NewLister` are deleted. A caller outside the library tests its own code by
declaring an interface over the Client, the way `pkg/mcp` already does.

**Every method takes a context, including the ones that touch no hardware.** For
a catalog lookup today, that means a context checked on entry and passed to
whatever reads a file. One rule is easier to keep than a list of exceptions, and
it means a catalog read from somewhere slower later does not need a new
signature.

**This breaks the public API.** No tag of this module has been published, so
there is no semver promise to break, and the only callers are in this
repository. `cmd/` and `pkg/mcp` are migrated in the same pull request as each
change. Each chunk's description lists the old and new signatures side by side
as the migration note.

## Session

A Session is one claim of the pedal and one handshake, used for as many
operations as the caller wants.

```go
func (c *Client) Open(ctx context.Context) (*Session, error)

func (s *Session) Model() string
func (s *Session) Presets(ctx context.Context, setlist int) (Listing, error)
func (s *Session) Preset(ctx context.Context, at slot.Address) (Reading, error)
func (s *Session) Export(ctx context.Context, at slot.Address, out string, as Format) (Written, error)
func (s *Session) Import(ctx context.Context, file string, at slot.Address) (Change, error)
func (s *Session) Copy(ctx context.Context, from, to slot.Address) (Change, error)
func (s *Session) Swap(ctx context.Context, a, b slot.Address) (Change, error)
func (s *Session) Select(ctx context.Context, at slot.Address) (Change, error)
func (s *Session) Close() error
```

`slot.Address` is new, `{Setlist, Slot int}`, and belongs in `pkg/sdk/slot`
because a device is addressed that way. `slot` still imports nothing.

The ctx given to `Open` bounds the claim and the handshake, not the session's
life. Each method's ctx bounds that operation. `Close` is idempotent, always
releases the interface, and returns the error that ended the read loop, if one
did. A method called after `Close` returns `ErrClosed`.

**The Client keeps its one-shot device methods.** `Client.Presets`, `Preset`,
`Export`, `Import`, `Copy`, `Swap` and `Select` take the same arguments as the
Session methods. Each one opens a Session, makes one call and closes it. The CLI
runs one operation per process and uses these.

**A Session is safe for concurrent use and runs one operation at a time.** Each
method takes a lock that is a one-slot channel, so a caller waiting for it gives
up when its ctx ends. The lock is held for the whole operation, because a copy
is several exchanges and a swap is four, and nothing may land between them. This
is the pedal lock `pkg/mcp` holds today, moved to the only place that can see
every caller.

`Client.Open` waits, subject to ctx, while another Session from the same Client
is open. Two claims of one interface are the failure the protocol document warns
about, and the Client is the only thing that knows both exist. A caller that
holds a Session and calls a one-shot on the same Client blocks until its ctx
ends. The doc comment on `Open` says so.

**MCP holds one Session across calls, and closes it after 10 seconds without a
device call.** `onDevice` and `handlers.device` go away. A small holder in
`pkg/mcp/internal/tools` opens a Session on the first device tool, hands it to
every device tool after that, and closes it when the server stops or once idle
passes. An agent's list, show and select then cost one handshake. The pedal's
front panel works again between bursts, because it stops refreshing footswitches
while an editor is attached. When an operation fails with a bus error the holder
closes the Session and reports the error. The next tool call opens a fresh one.
That reconnect is the agent's choice, not a retry, so it does not break rule 4.
The holder calls the SDK through an interface of its own over `*sdk.Session`,
generated with mockgen.

## The read loop

This is the riskiest part of the design. It changes the timing of every frame
the library reads, and nothing short of an HX Stomp shows whether the device
agrees. Chunk 49.5 does not merge until `just test-device` has passed on the
user's pedal.

**One goroutine per session reads, and nothing else does.** It starts after the
claim and before the handshake, and it runs until `Close`. It posts
`ReadContext` in a loop on a context of its own, so a read is always posted. It
decodes every frame in a transfer and routes the payload to its channel. The
routing appends to the channel's buffer, adds to its `rxBytes`, and signals the
channel's `arrived`, a one-slot channel that a waiter selects on. Routing holds
a short receive lock and never waits on a send, so a slow USB write cannot hold
a read back.

**`receive` becomes waiting.** `awaitReply` takes complete envelopes off its
channel's buffer. When none has arrived, it selects on `arrived`, on the loop
ending, on ctx and on the budget. The handshake's pauses, the pacing between
write chunks and `drain` are all waits of the same kind. `drain` waits until
three read windows in a row see no transfer. `settle()` stops sleeping and waits
out `flashBudget` on a timer while the loop keeps reading. `awaitLoaded` polls
the same way. The budgets tests shorten today, `replyBudget`, `commitBudget`,
`flashBudget`, `selectPoll` and the rest, move from package variables into a
`budgets` field on the session.

**Sends are serialised.** A send lock covers the sequence counter, the
transaction counter and `out.Write`, so frames leave in the order their numbers
say. Each frame's ack field reads `rxBytes` atomically, as it arrives.

**Acknowledgements.** The ack is cumulative, so what matters is who sends one
and when:

- **While an exchange is in flight** on a channel, its waiter acks after each
  wake that brought bytes, as `awaitReply` does today. Write chunks still carry
  the ack in their header and send no ack frame between them.
- **When no operation holds the channel**, the loop acks bytes that arrived for
  nobody once the channel has been quiet for `replyReadWait`, at most once per
  quiet period. It first drops the complete envelopes nobody asked for. A
  partial envelope stays in the buffer until the rest of its bytes arrive,
  because the framing has no marker to resynchronise on. A flag set for the
  length of each exchange tells the loop to keep out. This is the only traffic
  the design adds.
- **The events channel** has no reader. Its bytes are counted and acked, not
  kept.

**How it stops.**

- **On `Close`**, the session takes the operation lock with no budget of its
  own, so an operation in flight finishes first. That wait is finite, because a
  write is bounded per chunk and by `commitBudget`. It drains, acks each
  channel, sends the closing hellos and drains again, as today. Then it cancels
  the loop's context and waits for the goroutine to exit. `ReadContext` checks
  its context every 100ms slice. Only after that does it release the interface
  and the device.
- **On ctx cancel**, the operation returns `ctx.Err()` and the loop carries on.
  A write that has started still finishes, with its answer read, and the second
  write of a swap still runs. This is the PR #109 behaviour, kept by
  `context.WithoutCancel` where it already is. A late reply to an abandoned call
  lands in the buffer, and the next call skips it because its transaction does
  not match.
- **On a bus error**, the loop records the error, closes `dead` and exits. Every
  waiter returns that error. `Close` sends nothing more, because no read is
  posted to catch the device's answers. It releases the interface and returns
  the error. Nothing reconnects.
- **On a panic in the loop**, the loop recovers, records the value and stack as
  an error, and ends the way a bus error does. The recover is needed because a
  panic on a goroutine nobody joins kills the process without running the
  caller's deferred `Close`, and that path leaves the pedal needing a power
  cycle. A panic on the caller's goroutine is not recovered. The method's
  deferred unlock and the caller's deferred `Close` run as it unwinds.

**The rules still hold.** Rule 1: nothing resets. Rule 2 was broken and is now
kept by construction. Rule 3: writes are still paced by their answer and the
flash pause, and the operation lock stops two operations interleaving. Rule 4:
one handshake per session, and no reconnect on failure. Rule 5: `settle` still
waits the full 750ms. Rule 6 is unchanged. Burst writes stay impossible, because
every write is 256 bytes a frame under the send lock, and the idle ack stays off
any channel with an exchange in flight.

**Testing without hardware.** The conversation tests already run against mockgen
doubles of `sender` and `receiver`. The receiver's expectation becomes
`AnyTimes` with `DoAndReturn` reading transfers from a channel the test feeds.
That keeps the double generated, although the loop calls it at moments the test
cannot establish. `export_test.go` exposes the loop's done channel, so each test
asserts the goroutine has exited after `Close`. Rows cover:

- a notification between two calls read within one slice
- reads continuing through `settle`
- a cancel partway through a write, where the message still goes out whole
- a bus error, and a panic in routing, each reaching every waiter and still
  releasing
- two goroutines on one Session serialised

`just test` already runs with `-race`.

## One options type

**The device and setlist paths keep no input struct.** A field that is not there
cannot be ignored. A file is addressed through a handle, as the device is
through a Session:

```go
func (c *Client) Setlist(path string) *Setlist // a .hls or .hlb; nothing is read yet
func (c *Client) PresetFile(ctx context.Context, path string) (Reading, error)

func (f *Setlist) Presets(ctx context.Context, setlist int) (Listing, error)
func (f *Setlist) Preset(ctx context.Context, at slot.Address) (Reading, error)
func (f *Setlist) Export(ctx context.Context, at slot.Address, out string, as Format) (Written, error)
func (f *Setlist) Import(ctx context.Context, file string, at slot.Address, out string) (Change, error)
func (f *Setlist) Copy(ctx context.Context, from, to slot.Address, out string) (Change, error)
func (f *Setlist) Swap(ctx context.Context, a, b slot.Address, out string) (Change, error)
```

A file edit takes `out` because it always writes somewhere new. A device edit
takes none because it writes in place and keeps a backup. Neither signature has
room for the other's field.

| today       | becomes                                                                                                                             |
| ----------- | ----------------------------------------------------------------------------------------------------------------------------------- |
| `Where`     | vanishes: `Path` is `Client.Setlist`, `Setlist` is an argument, `CatalogPath` is `WithCatalog`                                      |
| `Read`      | vanishes: `File` is `PresetFile`, `Slot` is a `slot.Address`                                                                        |
| `Export`    | vanishes into arguments, with `As` a `Format`                                                                                       |
| `Put`       | vanishes: `File` and `out` are arguments, `BackupDir` is `WithBackupDir`                                                            |
| `Edit`      | vanishes into two `slot.Address` arguments and, for a file, `out`                                                                   |
| `Make`      | vanishes: `Build(ctx, recipeID, out string)`                                                                                        |
| `Corpus`    | vanishes: `Model` and `Instrument` never combine, so `ModelMeasurements(ctx, model)` and `ChainMeasurements(ctx, instrument)`       |
| `NewRecipe` | survives with its gear fields. Copying reads only `From`, `ID`, `Name` and `Kind`, so it becomes `ExtendRecipe` and `Client.Extend` |
| `Compile`   | survives as `{Rig, Template, Out string}`: three paths of one type are too easy to pass in the wrong order                          |
| `Filter`    | survives: all three fields narrow together                                                                                          |

**`Format` is typed.** `result.Format`, aliased as `sdk.Format`, with
`FormatRig` and `FormatPreset`. It lives in `result` because both `sdk` and the
flows name it. It implements `pflag.Value`'s three methods, so `--as hlx` is
checked when flags are parsed and `cmd` stops casting a pointer to `*string`.
`slots.Format` is deleted.

**What `slots` takes.** No `ListOptions`, `ShowOptions`, `DeviceOptions`,
`ExportOptions`, `ImportOptions` or `EditOptions`. A flow is a method on a
struct built once by the Client, holding what the Client was configured with.
Each call passes only an address and, where there is one, a path:

```go
// pkg/sdk/internal/deviceslots
type Flows struct {
	Catalog    *catalog.Catalog
	Compiler   Compiler
	Translator Translator
	Backups    Backups
	Capture    io.Writer
}

func (f *Flows) Copy(ctx context.Context, s device.Editor, from, to slot.Address) (result.Change, error)
```

`Deps` goes away as an embedded struct. Its three collaborators become fields of
`Flows`, where a nil value still reaches the real one.

**How this makes ignored fields impossible.** Every parameter a method takes is
read on every path through it, because the paths that used to share a struct are
now separate methods. The chunk 49.1 errors for ignored fields become
unreachable and are deleted along with the fields. For the two structs that
remain, a table test holds one row per field and asserts the field changes the
result.

## The `slots` split

`pkg/sdk/internal/slots` becomes three packages under `pkg/sdk/internal`, and
`Compile` moves out:

| package        | owns                                                                                                                                              |
| -------------- | ------------------------------------------------------------------------------------------------------------------------------------------------- |
| `fileslots/`   | list, show, export, import, copy and swap over a `.hls` or `.hlb`, and reading a standalone `.hlx`. Never reaches `device`                        |
| `deviceslots/` | the same operations and select over a `device.Editor`, turning a device's answer into a reading, and `KeptError`. Asks `Backups` before any write |
| `backup/`      | what a slot held before a write: whether to keep it, in which format, under which name, and where                                                 |
| `presets/`     | gains `Compile`, beside `Make`, since both build a preset                                                                                         |

The backup policy becomes code with a name. Today it is spread across `backup`,
`keeping`, `keep` and `replacing`:

```go
// pkg/sdk/internal/backup
type Held struct {
	At   slot.Address
	Name string // what the device calls the slot
	Body []byte // what the device answered, nil for nothing
}

type Decoder interface {
	Document(body []byte, at slot.Address, name string) (*preset.Document, error)
}

func New(dir string, d Decoder) *Keeper
func (k *Keeper) Keep(ctx context.Context, held ...Held) ([]string, error)
```

and, in the consumer, as CONTRIBUTING requires:

```go
// pkg/sdk/internal/deviceslots
type Backups interface {
	Keep(ctx context.Context, held ...backup.Held) ([]string, error)
}
```

The rules are the ones that hold today, written down in the package doc and in
one table test:

1. A slot the device answered nothing for is not kept.
2. A slot with no blocks, still called `New Preset`, is not kept.
3. A slot that decodes is kept as `.hlx`.
4. Anything else is kept as the device's bytes, `.bin`.
5. A backup never replaces another.
6. A write whose backup failed does not happen.

Rule 6 is enforced in `deviceslots`, which calls `Keep` before it writes. A file
edit is never backed up, because it writes to a new path. The `Decoder` is a
small type in `deviceslots` holding the catalog and translator, so `backup`
imports neither `device` nor `wire`.

**`main_test.go`.** The existing rules do not change. The device unit is still
`device`, `wire` and `slot`, and every new package sits under `pkg/sdk`. Three
tests are added:

- `TestBackupsDoNotReachTheDevice`: `backup` depends on neither `device` nor
  `wire`, so the policy cannot come to depend on the transport.
- `TestTheSDKReadsOneVariable`: the single `os.Getenv` above.
- `TestEverySignatureTakesALinePerParameter`, below.

## Signatures

**Keep the rule and enforce it with a test in `main_test.go`, then reformat.**
The rule exists so that adding a parameter shows as one added line, and a rule
broken 753 times protects nothing. golangci-lint has no linter that checks it,
since `lll` and `golines` only look at length. An AST walk beside
`TestATestFileSaysWhichKindItIs` is a few dozen lines and runs in `just test`.

The test covers every function and method declaration with parameters, test
files included, in every non-generated `.go` file. Function literals and
interface method specs are exempt. A literal is usually a one-line callback, and
an interface lists shapes rather than code anyone diffs. CONTRIBUTING's section
gains those two exemptions and the test's name. The reformat is a throwaway
program over `go/ast` followed by `just go-fmt`. The program is not committed.

## Chunks and order

Each chunk leaves the tree working. The CLI's flags and output and the MCP
tools' names and schemas stay the same throughout.

**49.4, options.**

- **Changes:** the `With…` functions, ctx on every method, and the four global
  seams replaced by `device.Opener`. `CatalogPath`, `StatsPath`, `RecipesDir`
  and `BackupDir` leave the input structs. The environment moves to `cmd`.
- **Left alone:** the input structs otherwise, `slots`, and the transport.
- **Proof:** the `sdk` suites run with `t.Parallel()` under `-race`, no test
  writes a package variable, and `TestTheSDKReadsOneVariable` passes.
- **Hardware check:** not required. Nothing on the wire changes.

**49.5, session and read loop.**

- **Changes:** the loop, send lock and budgets in `device`. `Client.Open` and
  `Session`, with the one-shots rebuilt on it, and the `…Device` openers in
  `slots` deleted. MCP's holder replaces `onDevice`. `TestRoundTrip` does its
  whole round trip in one Session.
- **Left alone:** the input structs, the package layout, and every frame's
  content. Timing changes, plus the idle acks.
- **Proof:** the loop rows above under `-race`, and MCP handler tests showing
  three device tools on one open.
- **Hardware check:** required before merge. Run `just test-device` on the HX
  Stomp. Then, through `tonestack mcp start`, run list, show and select in one
  burst. Leave the Session open while stepping through presets on the pedal,
  then import into a scratch slot. Wait past the idle close and confirm the
  footswitch labels follow the dial.

**49.6, one options type.**

- **Changes:** `Setlist`, `PresetFile`, `slot.Address` everywhere, `Format`, and
  the struct table above. `slots` takes arguments and `Flows`. The chunk 49.1
  refusals are deleted.
- **Left alone:** package layout and transport.
- **Proof:** the flow suites rewritten as rows, plus the per-field rows for
  `Compile` and `NewRecipe`.
- **Hardware check:** not required. The conversation is untouched, and 49.5's
  run covers it.

**49.7, the `slots` split and signatures.**

- **Changes:** `fileslots`, `deviceslots` and `backup` replace `slots`, and
  `Compile` moves to `presets`. The two remaining `main_test.go` rules land, and
  every signature is reformatted.
- **Left alone:** every public identifier. This is a move and a reformat.
- **Proof:** the backup policy table, the new `main_test.go` rules, and the 99%
  coverage gate.
- **Hardware check:** not required.

The reformat touches most Go files, so 49.7 merges when nothing else is open
against `pkg/`.

## Not in this

- **HTTP transport for MCP.** Still stdio only.
- **Snapshots and footswitch assignments.**
- **Reconnecting after a failure.** Rule 4 forbids it, and the MCP holder only
  opens a new Session when the agent makes another call.
- **Choosing between two attached devices.** The first one found is still the
  one used.
- **Publishing device notifications to callers.** The loop reads the events
  channel and throws the content away.
- **Replacing the flash pause with a completion signal.** None has been seen on
  the wire.
- **A USB backend for Linux or Windows.**
- **The error sweep, chunk 49.2,** and the chunk 49.1 refusals, which this
  deletes once they are unreachable.

## Afterwards

- **`CONTRIBUTING.md`:**
  - the project structure tree, with `slots` replaced by its three packages
  - "What to import": `Session`, `Setlist` and the options
  - "Where an interface lives": `Deps` becomes fields on a flows struct
  - "Test doubles": the `WithDevices` setter in `export_test.go`
  - "Function signatures": the two exemptions and the test
- **`docs/device.md`:** "A session has to be closed" gains the held Session and
  the idle close.
- **`docs/protocol.md`:** rule 2 says how the loop keeps it, and "Writing a
  preset" says the flash pause reads throughout.
- **`README.md`:** the "Go SDK" row mentions opening a Session.
- **[An MCP server](2026-09-13-an-mcp-server-design.md):** a superseded-in-part
  note, because "one device call at a time" now lives in the Session.
- **`docs/knowledge.md`:** checked in each chunk's pull request for a line this
  finishes.
