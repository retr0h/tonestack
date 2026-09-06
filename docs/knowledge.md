# How a sound is constructed

Turning a request like *"a Mike Dirnt sound"* into a signal chain. This is the
product's actual problem; everything else is plumbing.

## Constructing is not copying

The corpus holds a preset called "Basket Case". Copying it would inherit one
person's opinion, including their mistakes, and would answer nothing for a
player nobody has made a preset for.

The goal is a system that knows *how a chain is built*. That decomposes into
four problems with four different sources, and conflating them is why generated
tones come out generic.

| Problem                 | Source                                         | State                     |
| ----------------------- | ---------------------------------------------- | ------------------------- |
| Who plays what          | `recipes/` — hand-written                      | thin, grows by correction |
| Gear to model ID        | `schemas/gear-map.json`                        | 547 models                |
| What order blocks go in | statistics over `schemas/corpus/`              | not built                 |
| What values to set      | catalog defaults, corpus distributions, intent | not built                 |

## 1. Who plays what

Cultural knowledge. It is not in any preset and cannot be derived from one.

A language model is genuinely good at this for well-known players and will
confabulate for obscure ones — **and cannot reliably tell which it is doing.**
That is the largest correctness risk in the product, and it is why every recipe
carries `provenance`. See [recipes.md](recipes.md).

## 2. Gear to model identifier

Line 6 renames every model for trademark reasons. An Ampeg SVT ships as
`HD2_AmpSVBeastNrm`, a Klon Centaur as `HD2_DistMinotaur`, a Marshall JCM-800 as
`HD2_AmpBrit2204`. None of that is inferable from the identifier.

Knowing a player uses an SVT is worthless on its own. The mapping is what makes
artist knowledge actionable, and it comes from joining two files that both ship
inside HX Edit — see [catalog.md](catalog.md).

## 3. What order blocks go in

A chain has a grammar. Compression before drive, drive before amp, amp before
cab, time-based effects last. Bass chains differ from lead chains. Some blocks
co-occur; some never do.

This is learnable from the corpus **statistically, not imitatively**. Across
4,426 presets you can measure which blocks appear together, which position each
tends to occupy, and which categories a bass chain almost always contains. One
person's bad preset barely moves an average; copying that same preset inherits
all of it.

Not built. The corpus is collected; the measurements are not taken.

## 4. What values to set

The difference between "an SVT" and "Mike Dirnt's SVT". Three inputs, in order
of authority:

1. **The catalog's official defaults.** Line 6 states a `default` for every
   parameter. This is the anchor and is never wrong.
2. **Corpus distributions.** The median `Drive` across every preset using this
   model beats a guess, and the spread says how much it varies in practice.
3. **Intent.** A recipe's `character` lines — "mid-forward, not scooped", "grit
   only on hard attack" — become directional moves against the catalog's real
   ranges.

Not built.

## Artist or song?

Both, at different layers. A rig is not one thing, and treating it as one is
part of why generated tones sound generic.

| Layer       | Stability        | Example                 |
| ----------- | ---------------- | ----------------------- |
| Instrument  | career-long      | Precision Bass          |
| Amp and cab | career-long      | Ampeg SVT into an 8x10  |
| Technique   | career-long      | pick, near the bridge   |
| Settings    | per era or album | drive amount, EQ curve  |
| Effects     | per song         | the octave on one track |

*"A Mike Dirnt sound"* resolves to the characteristic rig: the stable layers
plus median settings. *"The Longview bass tone"* keeps the same instrument and
amp and moves only the settings.

Recipes record this split as `rig` and `variants`, so a refinement moves only
what should move. Getting the amp right and the drive wrong is a fixable near
miss; getting the amp wrong is not.

## Nothing here can hear

No part of this system can judge whether a preset sounds right, and no quantity
of corpus data changes that. **The evaluator is a person.** The architecture
assumes it:

- **The correction loop must be cheap.** Generate, push to the device, listen,
  fix one line of a recipe, never hear that mistake again. This is why
  [`pkg/sdk`](device.md) matters more than more corpus — it removes a manual HX
  Edit import from every iteration.
- **Decisions must be inspectable.** A generated rig should record why each
  block was chosen and how confident that choice was, so a wrong amp is visible
  before anyone plugs in rather than after.
- **Corrections must be permanent.** A fix belongs in a recipe, where it
  outranks generated knowledge for good.

## The pipeline

```text
request      "a Mike Dirnt sound"
   │
   ▼
recipe       recipes/artists/mike-dirnt.yaml        who plays what
   │         amp: Ampeg SVT
   ▼
gear map     schemas/gear-map.json                  gear to model
   │         HD2_AmpSVBeastNrm
   ▼
catalog      schemas/hx-stomp.catalog.json          what the device accepts
   │         Drive 0.0–1.0, default 0.39, DSP 28.27
   ▼
grammar      statistics over schemas/corpus/        what order      [not built]
   │
   ▼
values       defaults + distributions + character   what to set     [not built]
   │
   ▼
RigSpec      validated against the catalog
   │
   ▼
.hlx         written, pushed, heard, corrected                      [not built]
```

Steps three and four are the unbuilt middle. Everything above them exists;
nothing below them does.
