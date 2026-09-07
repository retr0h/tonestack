# RigSpec as the one model

**Status:** accepted, not yet implemented\
**Supersedes:** the two-schema split in
[2026-09-03-helix-preset-generator-design.md](2026-09-03-helix-preset-generator-design.md)

## The problem

The project shipped two hand-authored schemas, and their names were backwards.

| file                   | holds                                        | actually is                          |
| ---------------------- | -------------------------------------------- | ------------------------------------ |
| `recipe.openapi.yaml`  | "Ampeg SVT", character lines, provenance     | the abstraction                      |
| `rigspec.openapi.yaml` | `HD2_AmpSVBeastBrt`, `Bias: 0.74`, `dsp0/p1` | compiler output, named as if it were |

`recipe.openapi.yaml` states it outright: *"A Recipe is an input to preset
generation; a RigSpec is the output."* So the document named for the abstraction
was the device-bound artifact, and the two sat at nearly the same level. RigSpec
was `.hlx` with the routing stripped out, not a layer above it.

The cost was not cosmetic. Nobody could reason about the system out loud.

## The decision

**There is one specification. Mike Dirnt is an instance of it.**

```text
RigSpec          the one tier-1 spec. Device-independent. Authored, shared, published.
   │ compile(rigspec, catalog)
   ▼
chain            an internal Go struct. Not a format. No schema. No OpenAPI.
   │
   ▼
.hlx             Line 6's format, not ours.
```

The compiled chain exists for the few hundred lines between "resolve against a
catalog" and "write a preset". Nothing outside the binary consumes it, nobody
authors it, and it is never exchanged. It had a schema only because one was
written for it, and that was the mistake.

It runs backwards too, which is what makes RigSpec *the* standard rather than
merely the input format: read a `.hlx`, lift each `HD2_` identifier through the
catalog's `BasedOn`, and the result is a RigSpec. So `presets show` emits one,
`presets make` consumes one, and recipes are RigSpecs on disk. **Everything
entering or leaving the system is the same kind of thing.**

## One schema, progressively filled

Authored by hand, sparse:

```yaml
schema: RigSpec
id: mike-dirnt
subject: { kind: artist, name: Mike Dirnt, band: Green Day }
instrument: bass
chain:
  - { role: amp, gear: Ampeg SVT }
  - { role: cab, gear: Ampeg 8x10 }
character: ["mid-forward, not scooped"]
```

The same document after compiling for a device, same schema and more fields
present:

```yaml
chain:
  - role: amp
    gear: Ampeg SVT
    settings: { drive: 0.47, bass: 0.52, mid: 0.71, treble: 0.85 }
```

That is what keeps it one format instead of two.

### The resolved layer is never written back

An earlier draft of this design called the resolved layer a lockfile and
proposed persisting it. That was wrong. npm needs a lockfile because resolution
is expensive and drifts over time; ours is deterministic and takes milliseconds.
Persisting it would buy nothing and cost two things that matter: every compile
would dirty the file, and the diffs would fill with generated noise. Those diffs
are how corrections get reviewed.

Compile on demand, show it in output, never write it back. Then there is
genuinely one artifact.

## What the model holds, and why each part exists

### `subject`, not `artist`

People ask for a named player, a song, a genre, or a bare description. Making
artist the primary axis makes the other three second-class.
`subject: {kind, name, band?}` costs nothing now and avoids a migration.

### Sibling rigs, not a base plus deltas

An artist owns several rigs, by era and by song, and they can differ at the amp,
not merely in settings. Modelling them as deltas off a canonical rig assumes a
shared spine that may not exist. Each rig is complete; one is marked default,
because *"a Mike Dirnt sound"* with no qualifier has to resolve to something.
`extends: <id>` is available where a rig genuinely is a delta.

### Evidence is per claim

A flat `source: llm` on a whole document cannot say that the amp came from an
interview and the drive figure from corpus statistics. Different claims have
different support:

```yaml
- role: amp
  gear: Ampeg SVT
  evidence:
    - { kind: cited, url: "…", note: "Bass Player interview, 2004" }
    - { kind: video, url: "…", at: "1:42", note: "SVT visible on stage" }
    - { kind: audio, url: "…", at: "1:20-1:45", stem: bass, method: htdemucs,
        measured: { mid_ratio: 0.61 }, caveat: "live, room and PA included" }
  confidence: high
```

`kind` is an open enum, so a new source of knowledge is a new value rather than
a new format. This is also what makes a correction a one-line diff reviewable by
someone who knows the band.

A URL does not make a claim true. It makes it *checkable*. Confidence stays
human-set, and the tool should show the gap between a claimed confidence and the
evidence supporting it. An LLM-sourced entry with no citations reads as
unverified regardless of what it says about itself.

### `requires` covers only what the catalog cannot see

Most dependency questions are already answered. Line 6's `.models` files carry a
per-model `devices` list, which is how the catalog filters to one device, and a
catalog now records the release it came from.

| case                          | who knows                | declare? |
| ----------------------------- | ------------------------ | -------- |
| model absent on a device tier | catalog `devices` list   | no       |
| model needs newer firmware    | catalog `source` version | no       |
| third-party IR pack           | nobody                   | **yes**  |
| purchased or non-stock model  | nobody                   | **yes**  |

```yaml
requires:
  - { kind: ir, name: "Ownhammer SVT 8x10", slot: 82, url: "…" }
```

A device modelling nothing close to the named gear is not a dependency; it is a
**substitution**, recorded by the compiler with a reason and a lowered
confidence, in compile output rather than in the shared file.

### `target` discloses, it does not restrict

```yaml
target: { device: HX Stomp, catalog: "HX Edit 3.82" }
```

If a RigSpec pinned a device, a Helix Floor owner could not use it and
portability would be lost, and portability is the whole reason the shareable
layer is abstract. `target` says *"these values were arrived at here"*, so
somebody on other hardware knows to re-tune rather than trust. Advisory, never
gatekeeping.

### `mutations` is where the human ear gets written down

The system's founding constraint is that nothing in it can hear; the evaluator
is a person. The mutation log is the only place that person's judgment is
recorded.

```yaml
mutations:
  - ask: "make it clunkier"
    changed:
      - { path: chain[1].settings.feel, from: tight, to: loose }
    reason: >
      Line 6 documents Sag as "lower values offer tighter responsiveness…
      higher values provide more touch dynamics & sustain", and Bias X as
      "set low for a tighter feel". Read "clunky" as looser power-amp feel,
      not more gain.
    verdict: "closer, but muddy now, so keep the feel and put the drive back"
```

Four fields doing four jobs. `ask` keeps the human's words verbatim, because
"clunky" is not a parameter and normalising it away loses the question.
`changed` is the machine-readable delta. `reason` records the *interpretation*,
cited, so a later session can see that the reading was wrong rather than only
that the value was. `verdict` is what the human said after hearing it, and an
entry with no verdict is "not yet evaluated", which is useful state.

**Mutations are memory, not deltas.** `chain` always holds the current state;
the log is never replayed to reconstruct it. Event sourcing would be more
elegant and much worse to read, and a human reads this file.

It lives in the file rather than in git because the transport is a conversation.
A file pasted into a chat carries no history but its own.

What it is for, concretely:

1. **Not repeating a rejected move.** Scan for the same `path` with a negative
   verdict before proposing a change.
2. **Bracketing instead of oscillating.** 0.58 was muddy and 0.40 was thin, so
   try 0.48. The log is the search state.
3. **Learning one person's vocabulary.** Across enough rigs, `ask` → `changed` →
   `verdict` becomes a dictionary of what *this* person means by "gnarly".
4. **Knowing when the knobs are not the problem.** Eight failed settings rounds
   is evidence against a higher-confidence claim upstream, usually the amp.

Exclusions must be scoped to the same gear and should decay. A move that failed
on one rig is not universally wrong.

### Settings are deliberately lossy

`drive: 0.47` does not mean the same thing on a Helix, a Kemper, and a real SVT.
Three options were considered:

1. No settings in RigSpec. Portable, but discards corpus medians, which are the
   difference between "an SVT" and "an SVT set the way people set one".
2. Raw device values. Accurate, not portable, defeats the premise.
3. **A small musical vocabulary.**
   `drive, bass, mid, treble, presence, level, feel`, accepted as approximate.

Option 3. RigSpec carries what a musician would say out loud. Device particulars
(`Sag`, `Bias X`, `Ripple`, `Hum`) are *not* in RigSpec; the compiler sets them
from catalog defaults, corpus medians, and the manual's directional guidance. A
mutation therefore reads `feel: tight → loose`, not `Sag: 0.5 → 0.7`, which
keeps the log portable.

Round-tripping a preset through RigSpec is lossy by design and must not claim
otherwise.

## The four sources of knowledge

Naming these separately is what stops the system guessing where it could
measure.

| question                             | source                             | state          |
| ------------------------------------ | ---------------------------------- | -------------- |
| Which gear does this player use?     | cultural knowledge, an LLM         | the only guess |
| What is that gear called on a Helix? | HX Edit `.models` ⋈ Pilot's Guide  | built          |
| What else belongs in the chain?      | corpus statistics                  | not built      |
| Which way does a knob move?          | the Pilot's Guide parameter tables | not built      |
| Does it sound right?                 | a person                           | irreducible    |

Measurements taken from the corpus while writing this, as evidence that rows
three and four are real rather than aspirational:

- 89% of 169 bass-amp chains contain a compressor; 63% contain drive, and drive
  sits before the amp in 89% of those.
- Across 54 SVT instances, the median `Treble` is 0.845 where Line 6's stated
  default is 0.68. The factory default is measurably not what players use.
- `Bass` sits in 0.50–0.53 (consensus) while `Drive` spans 0.28–0.60 (taste).
  **The spread says how much of an opinion is worth having**, and a wide one
  should defer to a character line or the human.

The Pilot's Guide holds seven `Parameter Description` tables documenting exactly
the controls that cannot be guessed, `Master`, `Sag`, `Hum`, `Ripple Bias` and
`Bias X`, in directional, intent-mapped language. It omits Drive, Bass, Mid and
Treble because those are self-evident.

## Audio analysis is a comparator, not an extractor

The naive version, analysing a track and extracting knob values, does not work,
and it is the version everyone assumes will.

Measuring the Longview bass measures the bass, the player, the amp, the mic, the
DI, the console EQ, the mix compressor, the master, and the encoder. Set
`Treble` to match that spectrum and the mix engineer's decisions are baked into
the amp.

What works is measuring both sides the same way:

```text
reference stem      ──measure──┐
                               ├─► delta ─► "4 dB darker than the reference"
own DI through the preset ─────┘
```

The question stops being *"what Treble value equals this spectrum"*, which is
unanswerable, and becomes *"am I moving toward it"*, which is not. That makes
audio analysis a **verdict generator** feeding the mutation log as
`verdict: {kind: measured}` beside the human ones. It still cannot say whether
something sounds *good*. Spectral tilt and mid/scoop ratio survive a mix
reasonably; absolute level, compression, and drive-versus-tape-saturation do
not, and the system should report which is which.

Sources, ranked by how much they lie: official isolated stems, then bass
playthroughs, then live footage (best evidence for `gear` claims, since the amp
is visible), then Demucs separation of studio tracks. Spotify is a dead end: DRM
prevents audio access, and its analysis API returns timbre vectors of the full
mix.

The pipeline is a **separate binary**. It needs ffmpeg and torch, and nobody
generating a preset should pay for that. It writes evidence blocks into a
RigSpec; `tonestack` only ever reads them. That preserves the single static
binary with an embedded catalog.

## Deliberately not decided

Whether a correction edits the canonical file (one Mike Dirnt improving over
time) or publishes a fork (competing versions). The first needs no identity or
versioning fields; the second needs author, version, and a parent pointer. There
is one recipe today, and the answer will be obvious after twenty. Nothing above
depends on it.

## Sequencing

The artist-to-gear mapping is the thinnest and most easily reproduced layer in
the system: one search, or one LLM call. The compiler is the part almost nobody
can build. Work goes there first.

1. The Pilot's Guide parameter tables. A PDF parse, and it unblocks every
   character line.
2. Corpus statistics: chain grammar and parameter distributions with spreads.
3. The RigSpec schema and the code behind it.
4. The audio pipeline, once a generated preset is good enough to compare
   against.

Sharing, importing and any registry come after all of it. They distribute the
cheap half.
