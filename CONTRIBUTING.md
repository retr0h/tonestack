# Contributing

Thanks for contributing to tonestack.

## Before you start

- Read the [Code of Conduct](CODE_OF_CONDUCT.md). It applies to every
  interaction in this repo.
- **Check existing work.** Is there an existing PR? Are there issues discussing
  the feature/change you want to make? Please make sure you consider/address
  these discussions in your work.
- **Backwards compatibility.** Will your change break existing consumers of
  tonestack? It is much more likely that your change will be merged if it is
  backwards compatible. Is there an approach you can take that maintains this
  compatibility? If not, consider opening an issue first so that API changes can
  be discussed before you invest your time into a PR.

## Prerequisites

Install tools using [mise]:

```bash
mise install
```

- **[Go] 1.27.** Pinned in `.mise.toml` and in `go.mod`, and continuous
  integration reads it from `go.mod` so the two cannot drift apart. They did
  once: a test asserted that a map keyed by `any` could not be encoded as JSON,
  which was true on Go 1.26 and stopped being true on 1.27, so it passed locally
  and failed in CI.
- **[uv].** Python package runner. `just md-fmt` formats markdown with
  [mdformat] through `uvx`; nothing is installed into the repository.
- **[just].** Task runner used for building, testing, formatting, and other
  development workflows. Install with `brew install just`.

### Claude Code

If you use [Claude Code] for development, install this plugin from the default
marketplace:

```
/plugin install commit-commands@claude-plugins-official
```

- **commit-commands.** provides `/commit` and `/commit-push-pr` slash commands
  that follow the project's commit conventions automatically.

## Setup

Fetch shared justfiles and install all dependencies:

```bash
just fetch
just deps
```

## Project structure

```text
main.go              a single call into cmd
cmd/                 cobra wiring: flags to behaviour, no logic
internal/            implementation, not importable
internal/cli/        the shared visual language: theme, table, detail, help
internal/resolve/    a rig and a catalog become a chain
internal/lift/       a preset becomes a rig, and a rig becomes a preset
pkg/rig/             RigSpec, the one authored format, and its validation
pkg/chain/           a resolved chain. An internal struct, not a format.
pkg/catalog/         what a device can do: blocks, parameters, DSP costs
pkg/corpus/          what real presets say about a device, measured
pkg/preset/          read and write a .hlx preset file
pkg/setlist/         read and write .hls setlists and .hlb device backups
pkg/sdk/             talk to a device over USB. The only cgo in the tree.
pkg/sdk/wire/        the framing a device speaks. Pure Go, no hardware needed.
schemas/             the RigSpec contract, generated catalog, preset corpus
recipes/             curated rigs: which gear a player uses
docs/                how the format, catalog and generation work
.github/workflows/   CI
```

## How the system works

The domain lives in [docs/](docs/), not here. That covers turning a request into
a signal chain, the preset format, and the device:

- [docs/workflows.md](docs/workflows.md) says what to do, in order, for the
  common tasks
- [docs/knowledge.md](docs/knowledge.md) covers how a request becomes a signal
  chain
- [docs/recipes.md](docs/recipes.md) covers writing a rig, and the worked
  example beside it
- [docs/catalog.md](docs/catalog.md) covers what a device can do and where that
  comes from
- [docs/preset-format.md](docs/preset-format.md) covers how a `.hlx` file is
  laid out
- [docs/device.md](docs/device.md) covers reading and editing what a device
  holds
- [docs/protocol.md](docs/protocol.md) documents the USB protocol a device
  speaks

Keep that split. A fact about the domain belongs in `docs/`; a fact about
working on the project belongs here.

## Code style

Go code is formatted by [gofumpt] and linted using [golangci-lint], enforced by
CI.

```bash
just go-fmt-check   # Check formatting
just go-fmt         # Auto-fix formatting
just go-vet         # Run linter
```

The linters that run are declared in `.golangci.yml`. Read them there rather
than looking for a list here. A copied list goes stale the first time the
configuration changes. Generated files (`*.gen.go`, `*.pb.go`) are excluded from
formatting.

### Documentation

Markdown files are formatted with [mdformat] through `uvx`. This style is
enforced by CI.

```bash
just md-fmt-check   # Check formatting
just md-fmt         # Auto-fix formatting
```

## Code standards

### Function signatures

Functions with parameters use multi-line format, one parameter per line, with
the closing parenthesis and the return types on a line of their own:

```go
func FunctionName(
    param1 type1,
    param2 type2,
) (returnType, error) {
}
```

Functions taking no parameters stay on one line:

```go
func Name() string {
}
```

Adding a parameter then shows as one added line rather than a rewritten
signature.

### File naming

Name a file for what it holds. Avoid `helpers.go`, `utils.go`, and names of that
kind: they describe where code was put rather than what it is, and they
accumulate whatever has no other home.

`types.go` holds only type declarations: structs, interfaces, constants, and
aliases. A function belongs in a file named for what it does.

A test file is named for the production file it tests. Where tests grow too
large to read, split the production file first so each test file keeps a
counterpart, rather than splitting tests away from the file they cover.

### Errors live with whoever produces them

There is no shared errors package. `catalog` owns `ErrBadParam` because
`ParamValue` produces it; `rig` owns `ErrUnknownBlock`, `ErrOverBudget` and
`ErrBadTopology`. `rig` imports `catalog`, so the one error both need lives in
`catalog` and no cycle is possible.

Each error is a sentinel plus a struct carrying the detail, so callers match
with `errors.Is` and reach the detail with `errors.As`.

**Omit the receiver name when unused**, as in
`func (*OverBudgetError) Unwrap()`. `revive`'s `unused-receiver` says rename it
to `_`; `receiver-naming` says never use `_`. Omitting is the only form
satisfying both.

### Generated code

No Python, no shell one-offs in the build. Anything regenerating an artifact is
a Go program carrying a `//go:generate` directive, run by `go generate ./...`,
which `just generate` invokes and `just ready` includes. A generator lives
beside what it generates. Generated files are `*.gen.go` or live under `gen/`.

One exception, and it is outside the build: `tools/extract_gear_map.py` reads
the model-to-gear table out of HX Edit's manual. The manual's model-name column
uses a subset-embedded font no Go PDF library decodes, and it runs once per Line
6 release rather than on every build. `just gear-map` invokes it; nothing in
`just test` or `just ready` does.

### Go patterns

- Error wrapping: `fmt.Errorf("context: %w", err)`, so the chain names each
  layer it passed through and stays inspectable with `errors.Is` and
  `errors.As`.
- Early returns rather than nesting the successful path inside conditionals.
- Unused parameters: rename to `_`.
- Import order: standard library, third party, then local, separated by blank
  lines.

### Test doubles

A double for an interface this organization defines is generated with `mockgen`
and committed. Do not write a struct by hand to satisfy one.

Generated mocks live in a `mocks` package beside the code they mock, produced by
a `generate.go` holding the directive:

```go
package mocks

//go:generate go tool go.uber.org/mock/mockgen -source=../types.go -destination=types.gen.go -package=mocks
```

The generator is resolved through the module's tool dependencies, so every
checkout runs the version `go.mod` records. Destination files end in `.gen.go`
and are committed. Do not use `gen/` for mocks. That name is taken by API code
generation.

When the interface is **unexported**, a sibling package cannot work: the mock
has to import the package to name the types in the interface, and the package's
own tests have to import the mock. Generate it into the package instead, with a
destination scoped to tests so the mocking library stays out of the dependency
graph of anything that imports the package:

```go
// generate.go, in the package that declares the interface
package thispackage

//go:generate go tool go.uber.org/mock/mockgen -source=thing.go -destination=thing.gen_test.go -package=thispackage
```

Either way the directives live in a `generate.go` that holds no code, and the
generated file carries `.gen` so a reader knows not to edit it.

Where call sites would otherwise repeat the same expectations, write a
constructor returning a configured mock rather than introducing a hand-written
type. The generated mock is still what satisfies the interface.

Three doubles are written by hand, because generating them buys nothing:

- One standing in for a standard library interface: `net.Conn`, `fs.File`,
  `io.Writer`, `slog.Handler`. Those do not move when our code does.
- One carrying a real implementation of the behavior under test, such as signing
  with a genuinely generated key pair.
- A recorder for a dependency called from a goroutine the test cannot join,
  where a generated mock would assert a call count at a moment the test cannot
  establish. State that reason where the recorder is defined.

### File headers

Every `.go` file MUST start with the MIT license header. See any existing Go
file in the repo for the exact format. Build-tagged files put `//go:build` on
line 1, blank line, then the header.

## Testing

```bash
just test           # Run all tests (lint + unit + coverage)
just go-unit       # Run unit tests only
just go-unit-cov   # Generate coverage report
go test -run TestName -v ./...  # Run a single test
```

Coverage is gated at 99%. `just test` fails if total coverage drops below it, so
a change that adds untested code fails locally and in CI:

```bash
just go-unit-cov-check   # Report coverage and fail below the target
```

The target is declared in `.github/codecov.yml` and in this repository's
`justfile`. Change both together.

It is 99 rather than 100 because of one file. `pkg/sdk/usb.go` is every call
this project makes into libusb, one expression per method, and there is no way
to reach it without a device on the bus. Everything it forwards to is behind an
interface and covered: finding a device, choosing between two, claiming an
interface, waiting on a busy one, framing, sequence numbers, acknowledgements,
opening a channel and making a call all run against a bus a test supplies.

That file is counted rather than excluded on purpose. An exclusion hides how big
a file is; a target says what cannot be reached and gets worse if that file
grows. `.coverignore` holds only generated code and command wiring, and anything
added to it needs a better reason than being hard to test.

### Validation layers are tested independently

`ValidateStructure`, `ValidateParams`, `ValidateTopology` and `ValidateBudget`
are separate functions with separate error types and separate suites. A combined
test says something failed without saying which layer, and which layer is the
only useful part. `Validate` composes them in the order giving the most
actionable first failure.

### A preset must survive being read and written

Reading a `.hlx` and writing it back reproduces the file exactly, and there is a
test over the whole corpus asserting it. A change that breaks that is a change
that silently rewrites somebody's preset.

An earlier version of this section claimed the opposite, that floats could not
survive a round trip, because 30% of corpus values are not exactly
float32-representable. That is true of float32 and irrelevant here: values are
parsed to float64, where `0.707` survives exactly.

Making it hold needed three things, and each is a rule for any field added
later:

- **Do not use `omitempty` on a field that can legitimately be empty.** It drops
  an explicit `""`, which is a different document from one with the field
  absent. Use a pointer, or keep the field raw.
- **Keep what is not modelled.** Presets carry fields nobody documented: song,
  band, author, an appVersion spelled two ways. `DataMeta` holds the name and
  preserves the rest verbatim.
- **Keep the form a value arrived in.** `device_version` appears as a number, as
  `"0"` and as `"0.00"`. Parsing and reprinting turns the last into the second,
  which is a change nobody asked for.

### Test file conventions

- Public tests: `*_public_test.go` in the package's `_test` package, exercising
  the exported surface. This is the default.
- Internal tests: `*_test.go` in the same package, for what the exported surface
  cannot reach.
- Suite naming: `*_public_test.go` → `{Name}PublicTestSuite`, `*_test.go` →
  `{Name}TestSuite`.
- `testify/suite` with table-driven cases.
- One suite method per function under test. Success, errors, and edge cases are
  rows in one table, not separate methods.
- `export_test.go` exposes unexported symbols to external tests, by alias or by
  setter. Do not use an alias to re-cover behavior the caller's own test already
  reaches; a helper with its own contract is what the pattern is for.

## Before committing

Run `just ready` before committing to ensure generated code, package docs,
formatting, and lint are all up to date:

```bash
just ready
```

## Branching

All changes should be developed on feature branches. Create a branch from `main`
using the naming convention `type/short-description`, where `type` matches the
[Conventional Commits] type:

- `feat/add-retry-logic`
- `fix/null-pointer-crash`
- `docs/update-api-reference`
- `refactor/simplify-handler`
- `chore/update-dependencies`

When using Claude Code's `/commit` command, a branch will be created
automatically if you are on `main`.

## Commit messages

Follow [Conventional Commits] with the 50/72 rule:

- **Subject line**: max 50 characters, imperative mood, capitalized, no period
- **Body**: wrap at 72 characters, separated from subject by a blank line
- **Format**: `type(scope): description`
- **Types**: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `chore`
- Summarize the "what" and "why", not the "how"

Try to write meaningful commit messages and avoid having too many commits on a
PR. Most PRs should likely have a single commit (although for bigger PRs it may
be reasonable to split it in a few). Git squash and rebase is your friend!

## Submitting a pull request

- **Describe your changes.** Say what changed and why. A reviewer should not
  have to read the diff to learn the reason for it.
- **Issue/PR links.** Link any previous work such as related issues or PRs.
  Please describe how your changes differ to/extend this work.
- **Examples.** Add any examples or screenshots that you think are useful to
  demonstrate the effect of your changes.
- **Draft PRs.** If your changes are incomplete, but you would like to discuss
  them, open the PR as a draft and add a comment to start a discussion. Using
  comments rather than the PR description allows the description to be updated
  later while preserving any discussions.

## AI usage

This repo is written with AI assistance. All contributions are subject to the
[AI Usage Policy](AI_POLICY.md). Disclose the tool you used, and make sure you
can explain what your change does without the aid of AI tools.

## FAQ

> I want to contribute, where do I start?

All kinds of contributions are welcome, whether it's a typo fix or a shiny new
feature. You can also contribute by upvoting/commenting on issues or helping to
answer questions.

> I'm stuck, where can I get help?

If you have questions, open a [Discussion] on GitHub.

[claude code]: https://claude.ai/code
[conventional commits]: https://www.conventionalcommits.org
[discussion]: https://github.com/retr0h/tonestack/discussions
[go]: https://go.dev
[gofumpt]: https://github.com/mvdan/gofumpt
[golangci-lint]: https://golangci-lint.run
[just]: https://just.systems
[mdformat]: https://pypi.org/project/mdformat/
[mise]: https://mise.jdx.dev
[uv]: https://docs.astral.sh/uv/
