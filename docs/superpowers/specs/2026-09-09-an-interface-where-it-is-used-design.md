# An interface where it is used

**Status:** implemented\
**Relates to:**
[2026-09-09-where-a-package-belongs-design.md](2026-09-09-where-a-package-belongs-design.md)

## The problem

Every dependency between packages here is a package-level function call. There
are 47 of them, and none can be stood in for. `internal/slots` calls
`catalogview.Open` and gets whatever is on disk. `internal/presets` calls
`compile.Resolve` and gets the whole resolver. A test of either reaches for the
real thing or reaches for nothing.

`pkg/sdk` is the one place that does not work this way. It declares `Editor`,
`Writer`, `Selector`, `Lister` and `Bus`, generates doubles for them with
mockgen, and the commands that read a device take one as a parameter. That is
why the device path is tested without a device on the bus, and it is the only
part of the tree with that property.

## What an interface is for here

Not every call wants one. The 47 edges divide cleanly.

**Calculations.** `catalog.Int`, every `cli` rendering helper, `slot.Label`,
`rig.GearName`, the `wire` codec. Pure functions over values. A test that stands
in for `catalog.Int` is testing nothing, and the interface would be indirection
paid for with nothing bought.

**Capabilities.** Opening a catalog touches the disk. Finding a rig reads a
directory or the binary. Resolving a chain is the policy this whole project
argues about. Standing in for one of those is how a command gets tested without
the world attached.

Four of them: `catalogview.Open`, `pkg/compile`, `pkg/editor` and
`recipes.Find`.

## Where the interface goes

In the package that uses it. That is Go's rule rather than this project's, and
the reason is that the consumer knows which methods it needs.

The first draft of this had it backwards, with the public packages declaring
their own interfaces and the internal ones declaring theirs at the point of use.
There is no such split. `pkg/sdk` returns an interface because the type behind
it is unexported and `USBLister` has two build-tag variants, so there is no
struct it could return. That is the narrow exception: return an interface only
when you cannot return the struct.

So the shape is:

| layer              | what it holds                                          |
| ------------------ | ------------------------------------------------------ |
| provider           | a struct whose methods are the package-level functions |
| consumer           | an interface naming only the methods it calls          |
| consumer's `mocks` | the generated double                                   |

`pkg/compile` carries a `Compiler` with `Lift`, `Lower`, `Resolve` and `Fit`.
Nothing wants all four. `internal/slots` declares an interface with the first
two, `internal/presets` one with the last two, and each gets a double the size
of its own use.

## How a collaborator arrives

A `Deps` struct embedded in the command's options, every field optional, a zero
value reaching the real thing.

That is what `net/http` does with a nil `Transport`, and it is what keeps this
change from touching `cmd/`: the wiring a command already has still compiles and
still means the same thing. A test names only what it wants to replace.

The alternative was a parameter on each of the 21 exported functions in
`internal/slots` and the 24 call sites in `cmd/` behind them. That is a lot of
churn for a collaborator most callers never override.

## What this does not do

It does not put an interface in front of a calculation. `cli` rendering,
`catalog` value constructors, `slot.Label` and the `wire` codec keep being
called directly, and a later change that wraps one of them should say what it
gained.

It does not change behaviour. Every method delegates to the function it is named
after, and the tests assert that reaching the work through the type and calling
it directly agree.

It does not touch `cmd/`.
