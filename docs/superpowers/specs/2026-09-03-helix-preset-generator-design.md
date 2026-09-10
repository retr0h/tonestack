# Helix preset generator, phase 1 design

> **Superseded in part, 2026-09-06.** Written when the project was three modules
> named `helix-core` / `helix-sdk` / `helixctl`. It is now one module,
> `github.com/retr0h/tonestack`. The phase ordering, the RigSpec/catalog split
> and the validation design all still hold; the package names do not. See the
> addendum at the end of this file for the catalog change, and `CONTRIBUTING.md`
> for the current layout.

**Date:** 2026-09-03 **Status:** implemented, with the value half unbuilt
**Scope:** free text → validated `.hlx` preset, delivered as a CLI

The shape ships: a catalog, a rig, validation in four layers, and a CLI that
writes a `.hlx` HX Edit imports. Free text reaches it through an agent driving
those commands rather than through the tool itself, which is what the README
describes. What is not built is the half that decides values:
[knowledge.md](../../knowledge.md) still marks two of its four problems unbuilt,
block order and knob direction, so a generated preset is the right gear at its
factory settings.

## Problem

Building a Helix preset by hand means knowing which of several hundred amp, cab
and effect models correspond to a sound you can hear in your head, then setting
parameters you cannot name. Players who know what they want rarely know the
model catalog, and the gap is where the product is.

The eventual service: a person describes a sound on a website and downloads a
file that loads on their Stomp. This spec covers the first phase of that, same
capability, driven from a command line.

## What ships in phase 1

Free text in (`"a Mike Dirnt sound"`, `"warm jazzy clean"`), a `.hlx` file out
that HX Edit imports and the hardware loads.

Explicitly **not** in phase 1: audio analysis, YouTube ingestion, source
separation, web service, accounts, billing.

| Phase | Input                    | Ships as           |
| ----- | ------------------------ | ------------------ |
| 1     | free text                | CLI                |
| 2     | free text                | web service on EKS |
| 3     | YouTube / uploaded audio | added input path   |

Phases 2 and 3 add shells and sources around a core that does not change. That
is the design's main claim and the reason for the seams described below.

## Decisions

### Many sources, one intermediate representation

Every input path resolves to a `rig.Spec` before anything writes bytes. A source
performs I/O and returns a spec; nothing downstream knows which source produced
it. Adding an input later is one new implementation of one interface.

The alternative, each input path owning its own path to a file, was rejected. It
duplicates validation and DSP budgeting per path, and those are the parts most
likely to be wrong.

### Rig knowledge first, audio second

Considered three ways to decide what a sound *is*:

1. **Knowledge-driven.** A rig is chosen from what is known about players and
   genres; audio, when present, tunes it.
2. **Audio-first.** Measure a recording and derive a rig from the measurements
   alone.
3. **Hybrid.** Knowledge constrains the search space, audio optimises within it.

Chose (1). It ships without any audio pipeline, works on the request people
actually make ("make it sound like X"), and degrades honestly on inputs it does
not know. (2) is the harder problem and inverting distortion and cab convolution
is ambiguous in ways that would have dominated the schedule. (3) is the eventual
destination and remains reachable. It is (1) plus a measurement stage, not a
rewrite.

### Generated rigs with curated overrides

Free text cannot be served by a fixed table of recipes. A curated recipe wins
when one matches the request; otherwise Claude generates a rig, constrained by
forced structured output to identifiers that exist in the catalog.

This ships with zero authored content and lets curation become an optimisation,
author the artists worth getting exactly right, let the model handle the tail.
Curated results are deterministic and free; generated ones cost roughly $0.05
per request with the catalog in a cached prompt prefix.

Rejected: curated-only (cannot answer "warm jazzy clean"), and LLM-only (no way
to hand-fix a specific bad result except by tuning a prompt).

### Catalog derived from owned hardware

There is no published schema for `.hlx`. The container is plain JSON; the
vocabulary of model identifiers, parameter keys, ranges and DSP costs is not
documented anywhere.

The catalog is derived by diffing presets exported from hardware the author
owns. Slower than adopting a third-party catalog, but it is correct for the
firmware actually being targeted, and every entry can be verified on the device.

Third-party catalog data may seed entries, marked `inherited` and treated as
unverified until confirmed.

### Deterministic core before the model

Build order: extractor, then data model and validation, then the writer, then
the LLM source. A model in the loop during writer development makes every
failure ambiguous between "wrong rig" and "malformed file". Ordering removes the
ambiguity at no cost, since the LLM's output schema is generated from the
catalog and cannot precede it.

### Go, with one Python exception

Go for the core, CLI and eventual service: a single static binary, small
container images, and a type system suited to catalog validation.

Phase 3's measurement work is fine in Go: spectral analysis, envelope detection,
autocorrelation. Source separation is not; it is a PyTorch model with no
credible Go equivalent. It becomes a separate service on GPU nodes, which is the
right boundary regardless of language, since nothing else in the system shares
its scaling profile.

## Architecture

Full detail in `docs/architecture.md`. In brief:

- `helix-core` holds catalog, rig, synth, sources and typed errors. No output,
  no I/O outside a source, typed values at every boundary.
- `helixctl` is a formatter over the core, holding no logic worth testing.

Five independent validation layers: structural, parametric, budget, topological,
envelope. Separate functions, separate error types, separate tests.

`ParamValue` is a tagged union rather than `map[string]any`, because Helix
parameters mix floats, ints, bools and enum strings in one object and the write
boundary is exactly where discarding type information causes files that look
correct and do not load.

Two separate enums, deliberately not merged. Catalog `Provenance` (`observed` /
`measured` / `inherited` / `assumed`) records how an entry was learned and gates
delivery: a preset containing a block with an `assumed` DSP cost is not returned
to a user. Rig `Origin` (`curated` / `llm` / `audio`) records how a rig was
decided, and is what explains to a user where their preset came from.

## Testing

- Validation layers unit-tested independently, because a combined test cannot
  say which layer rejected a spec.
- Synth output compared byte-for-byte against golden fixtures.
- Round-trip property: parsing an exported preset and re-writing it reproduces
  the input, which is the strongest available check with no format
  documentation.
- Hardware verification is a distinct claim from schema validation. Both are
  recorded; neither substitutes for the other.

## Open questions

Answerable only from real exported files. Nothing above is designed around a
guess on any of these:

1. Are parameter values normalised `0.0`–`1.0` or stored in display units? A
   third-party project asserts normalised; unverified.
2. Are parameter keys stable across firmware versions?
3. The actual `device`, `version` and `modeldata_version` integers for the
   target firmware.
4. Whether JSON key order matters to the parser. Go sorts map keys on
   marshalling; exports are not sorted.
5. How snapshots are encoded.

Blocked on these: the synth writer's envelope, and the extractor. Not blocked:
`ParamValue`, the catalog and rig types, and four of the five validation layers.

## Risks

**DSP budget is the likeliest visible failure.** Costs cannot be derived from
preset diffing, only measured. Provenance gating is the mitigation; it means
early catalog coverage will be narrower than the model catalog.

**Generated rigs are non-deterministic.** Identical requests can drift between
runs. Caching by normalised prompt and an eval harness over known artists are
the mitigations. Curated recipes exist partly for this reason.

**Firmware drift.** A catalog is correct for one firmware version. The envelope
validation layer exists to fail loudly rather than emit a file that silently
will not load.

**Ingestion legality (phase 3).** Server-side extraction of YouTube audio
conflicts with their terms and is a poor foundation for a paid service. Deferred
with the phase, and likely resolved by accepting user-supplied audio instead.

______________________________________________________________________

## Addendum, 2026-09-06: the catalog problem is solved

This spec's Risks section names DSP budget as "the likeliest visible failure",
on the grounds that costs "cannot be derived from preset diffing, only
measured". That reasoning was sound and the conclusion is now obsolete.

HX Edit ships Line 6's complete model definitions as plain JSON inside its app
bundle: 681 models, every parameter with a real `min`, `max` and `default`, and
645 models with a DSP `load`. Nothing about ranges or DSP cost needs to be
inferred any more.

What changes:

- **The DSP risk is retired.** Costs are stated by the vendor.
- **`prov` gains `official`** and most parameters carry it. The `assumed` gate
  in `ValidateBudget` stays as a guard but should rarely fire.
- **The catalog is generated locally**, from a licensed HX Edit install, and is
  not shipped inside a hosted service. `helixctl catalog extract` must read the
  local app bundle, and must fail clearly on a machine without it.
- **The corpus keeps a narrower job.** Establishing which of the 681 models an
  HX Stomp accepts (366 confirmed), and what real presets look like.

What does not change: the phase ordering, the RigSpec/catalog split, and the
requirement that a preset be validated before it reaches anyone.

One correction to this spec's testing section: it specifies golden files
compared byte for byte. That is not achievable, because re-serialising a parsed
float does not reliably reproduce the original literal, and roughly 40% of
corpus float values are not exactly float32-representable. Round-trip tests must
compare semantically.

One new defect to fix in code: `catalog.Provenance` defines `measured`,
`observed`, `inherited` and `assumed`. The generated catalog now emits
`official`, which no constant matches. The extractor task must reconcile them.
