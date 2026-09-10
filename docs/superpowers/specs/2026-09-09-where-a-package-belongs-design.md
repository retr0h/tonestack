# Where a package belongs

**Status:** proposed\
**Relates to:**
[2026-09-06-rigspec-as-the-one-model-design.md](2026-09-06-rigspec-as-the-one-model-design.md),
[2026-09-06-helix-sdk-design.md](2026-09-06-helix-sdk-design.md)

## The problem

Nothing states what `pkg/` is for. It has been filled by instinct, which has
worked, and the instinct has never been written down or tested. Two questions
make that expensive rather than untidy:

- The device SDK may one day be its own repository.
- Somebody may want RigSpec without wanting the rest of this.

Measured rather than assumed, three things are already true and worth keeping:

| fact                                                   | measured by               |
| ------------------------------------------------------ | ------------------------- |
| `pkg/` never imports `internal/`                       | `grep`, whole tree        |
| `pkg/sdk`'s entire footprint is `sdk/wire` and `slot`  | `go list -deps ./pkg/sdk` |
| `pkg/` holds three clusters with no edges between them | `go list -deps` on each   |

The clusters:

```text
device   sdk → wire → slot
files    setlist → preset → {catalog, chain}
format   rig → {rig/gen, resources/schemas}
```

Two things are wrong with it.

**The operations layer is on the wrong side.** `internal/lift` and
`internal/resolve` are pure: no `io.Writer`, no printing, dependencies only on
`pkg/`. So are `decode.go`, `encode.go` and `routing.go` inside
`internal/slots`, which is 4,975 lines and holds both the command
implementations and about 710 lines of catalog-driven translation. Somebody who
imports RigSpec today can author one and validate it, and cannot compile it into
a preset. That is the whole point of the project, and it is unreachable from
outside.

**Nothing says what a package in `pkg/` may expose.** Exported and unexported
have been decided one identifier at a time. A public package that exports what
nobody outside it calls is a promise nobody asked for, and every one of them has
to be kept.

## What the reference implementations do

Both projects [docs/protocol.md](../../protocol.md) is built on have solved this
already, and they agree.

| layer                                      | [tonepush](https://github.com/crmne/tonepush) | [fretwire](https://github.com/john-baxter-dev/fretwire) | here            |
| ------------------------------------------ | --------------------------------------------- | ------------------------------------------------------- | --------------- |
| pure codec: framing, RPC, preset documents | `hx-proto`                                    | `fretwire-protocol`                                     | `pkg/sdk/wire`  |
| USB transport                              | `hx-usb`                                      | `fretwire-usb`                                          | `pkg/sdk`       |
| device session                             | in the client                                 | `fretwire-core`                                         | `pkg/sdk`       |
| reference data, indices to names           | `hx-catalog`                                  | `fretwire-data`                                         | `pkg/catalog`   |
| transport-independent operations           | `voidx-client`                                | `fretwire-commands`                                     | **`internal/`** |
| frontends                                  | `-cli`, `-gui`                                | `-cli`, `-tauri`, `-serve`, `-mcp`                      | `cmd/`          |

Five of the six match. The row that does not is the operations layer, which is a
published crate in both and internal here.

fretwire ships `-serve` and `-mcp` over the same core as its command line, which
is the question "what would we extract if this became a service" already
answered by somebody else: the same operations layer, a different frontend.

Their vocabulary is worth borrowing where it is Line 6's own. tonepush: "a
preset travels as the device's own document byte for byte", and it reserves
"blob" for impulse responses rather than presets. fretwire: "importing reference
data transforms those indices into meaningful model and parameter names".
[protocol.md](../../protocol.md) already calls the `.hlx` "a host-side format"
and the two "different representations of the same preset".

## The rule

**`pkg/` holds what something outside this repository would call. `internal/`
holds everything else.**

And within `pkg/`, the same test applies to each identifier: exported means
somebody outside the package calls it, and for a package in `pkg/` that means
somebody outside this repository plausibly would. Everything else is unexported,
or it belongs in `internal/`.

The test to apply, in this order:

1. Would the cloud half of a service call it?
2. Would an agent on somebody's desktop call it?
3. Would a stranger writing their own tool call it?

If the answer is no three times, it is application code, however clean it is.

Maximising `internal/` is not the goal. Applying the test to the tree as it
stands moves three things up and nothing down, because `internal/` already holds
only application code apart from those three.

## The target

```text
pkg/
  sdk/          the device: discovery, session, transport
  sdk/wire/     the framing, and the device's own document
  slot/         addressing, 01A to 42C
                              ^ these three are the extraction unit

  rig/          RigSpec: types, validation, reading and writing
  rig/gen/      generated from the contract
  catalog/      what a device can do
  corpus/       what real presets say
  preset/       .hlx
  setlist/      .hls and .hlb
  chain/        a resolved chain

  editor/       the device's document to a chain and back, against a catalog
  compile/      RigSpec to a chain to a preset and back, against a catalog

internal/
  cli/          rendering
  slots/        the commands
  recipes/ presets/ catalogview/ corpusview/ device/
  catalogen/ corpusgen/ specdoc/
```

Two packages rather than one, so a consumer takes the half they need. `compile`
depends on the format cluster and `editor` on the device cluster, and neither
depends on the other.

**`editor` is Line 6's word.** Their application is HX Edit,
[protocol.md](../../protocol.md) says "the editor lives on a vendor-specific
interface", and fretwire calls the same layer its editor command layer. The
alternative was inventing a noun for "a device document read as a chain", and
the prior art says not to: it is not a thing, it is an operation.

## The order of work

Four steps, each landing on its own. The restructure is last because it is the
one that moves files, and it is easier to review a move that carries no other
change.

### 1. Audit, and write down what it found

For every exported identifier in `pkg/`, find whether anything outside its own
package uses it, and classify:

- used outside the repository plausibly: keep
- used only inside this repository: judge it against the three questions
- used nowhere outside its own package: unexport

Method: parse the declarations, grep the tree for each, discount test files.
Deliverable is a table in the pull request, no code changes.

### 2. Shrink, with the code where it is

Unexport what the audit found. Move any whole package that fails all three
questions into `internal/`. No renames, no new packages, no moved
responsibilities. Every change is mechanical and the gate proves it.

Doing this before the restructure means the packages that move in step 3 move at
the size they should be, rather than being moved and then trimmed.

### 3. Restructure

- `pkg/compile` from `internal/resolve` and `internal/lift`.
- `pkg/editor` from `decode.go`, `encode.go` and `routing.go` in
  `internal/slots`, and whatever of `deviceread.go` is translation rather than
  rendering.
- `internal/slots` keeps the commands and the writers.

### 4. Write the rule down, and test it

CONTRIBUTING gets the rule and the three questions. The SDK's boundary gets
stated: what the extraction unit is, and that `wire` is its public vocabulary
rather than its private guts.

And a test asserts the rule, because the compiler will not. `internal/` sits at
the repository root, so Go permits `pkg/` to import it; only a test can keep
that from happening by accident.

## What this does not do

**No second module.** Extraction stays cheap precisely because the graph is
disjoint, and a module boundary inside one repository buys friction first and
freedom later. If somebody asks for the SDK, that is the moment.

**No `pkg/sdk/*` umbrella.** Putting the formats under the SDK would tie a
device-independent format to the device it exists to abstract, and would
conflate three audiences with different lifecycles.

**No rename of `pkg/rig`.** Whether RigSpec's import path should say `rigspec`
is a real question and a separate one.

**Nothing in `cmd/` moves.** It is already the thin part.

## What it costs if it is wrong

Step 2 is reversible: unexporting is a compile error away from being noticed,
and re-exporting is trivial. Step 3 moves files, which git records as moves and
review can read. Neither changes behaviour, so the gate is the check: same
tests, same coverage, no new gaps.

## What the audit found, and what changed in doing it

Added rather than rewritten, so the record shows what the design got wrong.

**The audit.** Every exported identifier under `pkg/`, resolved with `go/types`
rather than by grepping, so a method reached through an interface counts and a
name that merely appears in a comment does not. Struct fields are excluded; they
belong to their type. Uses from a package's own external test are counted
separately, because that test lives in the package's directory and is not
evidence that anybody outside would call the thing.

| bucket                                          | count |
| ----------------------------------------------- | ----- |
| exported in `pkg/`                              | 364   |
| called from another package's production code   | 168   |
| reached only by its own package's external test | 117   |
| reached by nothing                              | 79    |

Two things stop that last column being a list of deletions, and both were missed
when the step was written.

A type nobody names is still public. `wire.Document` is reached by nothing,
because every caller writes `doc, err := wire.DecodeDocument(...)` and lets the
compiler name the type. Unexporting it would leave an exported function
returning a value a caller cannot declare. The same holds for every error struct
a caller reaches through `errors.As`.

And some of it is exported on purpose against this test. `ValidateStructure`,
`ValidateParams`, `ValidateTopology` and `ValidateBudget` are called only by
`Validate`, and CONTRIBUTING says why they are four functions rather than one:
each has its own error type and its own suite, and a combined test says
something failed without saying which layer.

So the count is a reading list, not a work list. Shrinking the surface is worth
doing and needs the judgment of whoever knows what each thing is for. It is not
in this change.

**Nothing moved down.** `pkg/corpus` looked like the one package that would,
until `compile.Resolve` turned out to take a `*corpus.Stats`. A type in the
signature of a function a consumer calls is part of what that consumer compiles
against, wherever it is declared. The prediction that this test moves code up
and nothing down held.

**The mirror.** `pkg/foo` is what a consumer calls and `internal/foo` is the
rest of that domain, named to match so both halves are findable. No domain needs
the second half today: unexported identifiers already give a package its private
side, and a twin package earns its place only when the implementation has to be
several files with tests of its own that nobody outside may import. When one
appears its tests still live in `internal/foo_test` as `*_public_test.go`,
because the suffix says how Go sees the surface rather than who may import it.

**So the boundary test reads the way it was first written.** Adding `pkg/foo`
over `internal/foo` would have `pkg/` importing `internal/`, and the assertion
would have had to invert. It did not happen, so `main_test.go` asserts that no
package under `pkg/` imports `internal/`, which is also the thing that keeps the
SDK liftable.

**Two packages became one.** `internal/resolve` and `internal/lift` were merged
into `pkg/compile` rather than kept as neighbours under it. They shared one
unexported name between them, `lift` already imported `resolve` for `Check` and
`Gear`, and splitting a rig's journey to a preset across two import paths made a
consumer learn an ordering that is not theirs to know.

**`pkg/editor` came out of `internal/slots`, which kept the commands.**
`decode.go`, `encode.go` and `routing.go` moved whole. `deviceread.go` split:
the translation went to `pkg/editor/document.go` and `writeDeviceRig`, which
opens a catalog and writes YAML to an `io.Writer`, stayed behind. That file went
from 265 lines to 95.

The tests that came with it kept their shape. What reaches an unexported helper
stayed an internal test in the same package, and `Document`, `Snapshots` and
`Footswitches` gained public tests of their own, because the suite that covered
them stayed in `internal/slots` with the command it tests and coverage is
counted per package.

## What step 2 did

Twenty-seven identifiers became unexported and one turned out to be dead. The
packages that were mostly machinery lost the most: `wire` and `sdk` between them
account for sixteen of the twenty-seven.

The count of 210 candidates a first pass produced was wrong four times over, and
each way it was wrong is worth naming, because the same mistake is available to
anybody repeating the measurement.

**A seam in `export_test.go` is not surface.** `rig.Against`, `catalog.Decode`
and six others are declared in test files, which are compiled into the test
binary and shipped to nobody.

**A type nobody names is still public.** Callers write
`doc, err := wire.DecodeDocument(...)` and let the compiler name the type, so
`wire.Document` is referenced by nothing while being the thing the function
hands back. Anything reachable through the signature of something that stays,
transitively, stays with it.

**A method implementing an interface is called by the interface.**
`Session.Call` looks unreferenced because callers hold an `sdk.Editor`. Checking
every interface in the build against every candidate receiver covers `error`,
`json.Marshaler` and `pflag.Value` without naming any of them.

**An error escapes as `error`, never as its own type.** `chain.ErrOverBudget`
and `OverBudgetError` are referenced by nothing and are exactly what
`chain.Validate`'s caller matches on. CONTRIBUTING already says every error is a
sentinel plus a struct so callers can use `errors.Is` and `errors.As`, which
makes both halves part of the contract of whatever can return them.

Two things were exempted after the measurement rather than by it.
`ValidateStructure`, `ValidateParams`, `ValidateTopology` and `ValidateBudget`
are called only by `Validate`, and CONTRIBUTING says why they are four functions
with four error types and four suites. And the methods of a type that has itself
been unexported were left alone: `session.Call` is unreachable because nothing
outside can hold a `session`, so the type closed the question and renaming the
method would only churn the tests.

**The tests did not move.** Every external test that reached one of the
thirty-two now reaches it through an alias in `export_test.go`, which is the
pattern this repository already uses and which compiles into the test binary
alone. `go doc` and any importer see the lowercase name. Moving those suites
into their packages would have cost hundreds of lines and bought nothing.

**One identifier stayed exported because of where it lives.** `sdk.USBLister` is
declared in `usb.go`, every line of which is uncovered and uncoverable without a
device on the bus. Renaming it puts those lines in the diff, and a patch that
touches them cannot meet a 99% patch target however well the change is tested.
Loosening the target or excluding the file would trade a real guarantee for a
cosmetic rename, and nothing outside the package can name the type anyway,
because `NewUSBLister` returns `Bus`. It stays.

That file also carries the one thing a type-aware rename cannot see. `usb.go`
and `usb_nocgo.go` both declare `USBLister` behind opposite build tags, and a
pass over the syntax trees only sees the build it runs under. Renaming one would
have left the package exporting a different surface depending on whether cgo was
enabled, and nothing in the gate would have said so. Any rename inside a
build-tagged file needs both builds checked, which is `CGO_ENABLED=0 go build`
here.

**One thing was dead rather than over-exported.** `setlist.SlotsPerSetlist` was
referenced by nothing, inside the package or out, and the linter said so the
moment it went lowercase. The fact it recorded, that a setlist file always holds
128 slots, is in [preset-format.md](../../preset-format.md) where a reader will
find it.

`slot.Parse` is the one to argue about. It is the inverse of `slot.Label`, which
is exported and used, and a pair like that reads as an API even when only half
of it has a caller. It went lowercase because the rule is about callers rather
than symmetry, and re-exporting it is one line.
