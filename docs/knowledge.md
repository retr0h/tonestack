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

| Problem                 | Source                                                 | State                     |
| ----------------------- | ------------------------------------------------------ | ------------------------- |
| Who plays what          | `pkg/sdk/rigs/`, hand-written                          | thin, grows by correction |
| Gear to model ID        | `resources/schemas/gear-map.json`                      | 547 models                |
| What order blocks go in | statistics over `resources/schemas/corpus/`            | added blocks placed       |
| Which way a knob moves  | parameter names, and the HX Edit manual's amp controls | cited below; not data yet |
| What values to set      | catalog defaults, corpus medians, intent               | six axes of ten           |

One specification covers all of it.
[The RigSpec design record](superpowers/specs/2026-09-06-rigspec-as-the-one-model-design.md)
says what it holds.

## 1. Who plays what

Cultural knowledge. It is not in any preset and cannot be derived from one.

A language model is good at this for well-known players and confabulates for
obscure ones. It cannot tell which it is doing. That is the largest correctness
risk in the product, and it is why every rig carries evidence per claim. See
[recipes.md](recipes.md).

### Which sources are worth what

Not equally. Ranked for a claim about gear:

| kind    | what it is                           | worth                                             |
| ------- | ------------------------------------ | ------------------------------------------------- |
| `cited` | a published rig rundown or interview | best. Somebody with access wrote it down          |
| `user`  | a forum thread, TalkBass or Reddit   | argued and corrected in public, and uneven        |
| `video` | footage                              | good for how it sounds, weak for what the box was |
| `llm`   | a model asserted it                  | a starting point, never an answer                 |

Video is last on purpose. A stage seen from forty feet says little about which
head was on it, and the description under a clip is whatever the uploader typed.
Cite it for character and technique, where hearing or seeing it is the point,
and say in the note that is what it is for.

Two practical things, both found the hard way while sourcing the shipped rigs:

**A search result is not a source.** A search summary blends several pages and
their comment sections into one answer, and it will attribute a claim to a page
that does not make it. Four of the first eight rigs cited something that was not
on the page once somebody opened it. Open the page.

**A 200 is not verification.** The Mike Dirnt example cited a bassplayer.com
article for the amplifier. The URL still answers 200, because the whole site now
redirects to a section index on another domain, and the article is gone.
Checking that a link resolves catches a 404 and nothing else. Open it and read
the sentence.

**Splitting a claim does not split its evidence.** Turning "thumping low end
under a hard top" into `loose-low-end` and `bright` left both citing the rundown
that produced the sentence, and that rundown describes the low end and says
nothing about the top. One of the two halves is usually unsourced, and
mechanically copying the citation onto both is how a guess acquires a URL.
Sharing a page between two terms is fine when each quotes a different sentence
from it. Sharing a sentence is not.

**Forums block machine readers.** Reddit refuses Anthropic's crawler outright,
and TalkBass returns 403 to anything automated. Both can be found by search and
neither can be read by an agent, so a rig citing one records that nobody checked
it. Those citations say so in a `caveat`, and they are the ones a person should
open first.

## 2. Gear to model identifier

Line 6 renames every model for trademark reasons. An Ampeg SVT ships as
`HD2_AmpSVBeastNrm`, a Klon Centaur as `HD2_DistMinotaur`, a Marshall JCM-800 as
`HD2_AmpBrit2204`. None of that is inferable from the identifier.

Knowing a player uses an SVT is worthless on its own. The mapping is what makes
artist knowledge usable, and it comes from joining two files that both ship
inside HX Edit. See [catalog.md](catalog.md).

## 3. What order blocks go in

A chain has a grammar. Compression before drive, drive before amp, amp before
cab, time-based effects last. Bass chains differ from lead chains. Some blocks
co-occur; some never do.

This is learnable from the corpus **statistically, not imitatively**. Across
4,426 presets you can measure which blocks appear together, which position each
tends to occupy, and which categories a bass chain almost always contains. One
person's bad preset barely moves an average; copying that same preset inherits
all of it.

Partly built. The corpus is measured per instrument: across 159 bass chains, 88%
hold a compressor and 61% hold drive, which sits before the amp 88% of the time.
A build uses that to add the blocks a chain almost always holds, three chains in
four or more, with the model that instrument's players use most, on the side of
the amp where they put it. Blocks a rig names keep the order the rig gives them;
nothing reorders a whole chain by the grammar yet.

## 4. What values to set

The difference between "an SVT" and "Mike Dirnt's SVT". Three inputs, in order
of authority:

1. **The catalog's official defaults.** Line 6 states a `default` for every
   parameter. This is the anchor and is never wrong.
2. **Corpus distributions.** The median `Drive` across every preset using this
   model beats a guess, and the spread says how much it varies in practice.
   Across 54 SVT instances the median `Treble` is 0.845 where Line 6's stated
   default is 0.68. The factory default is measurably not what players use.
   `Bass` sits in 0.50–0.53 and `Drive` spans 0.28–0.60, so the spread also says
   how much of an opinion is worth having.
3. **Intent.** A rig's `character` words become moves against the catalog's real
   ranges. Each word is worth one step from where the corpus left that control,
   and six axes act: `mids`, `highs`, `drive` and `low-end` on the amp, `space`
   on the reverb and `attack` on the compressor.
   [recipes.md](recipes.md#character-describes-the-result-not-the-control) lists
   the words.

Built, for those six axes. What is not built is a general answer to which way
any control moves, which is what turning a measured difference into a change
would need.

### Which way a knob moves

Mostly the name says. The catalog has 641 parameter names across 5,602 controls,
and 46% of those controls carry a name whose direction needs no explaining:
`Level`, `Treble`, `Drive`, `Mix`, `Feedback`, `Decay`.

Line 6 publishes no table per model. The HX Edit manual documents the amp
controls a name does not explain, in one list headed *Common Amp Settings*:

| control         | lower                                                    | higher                                          |
| --------------- | -------------------------------------------------------- | ----------------------------------------------- |
| `Master`        | less power amp distortion, and less effect from the rest | more power amp distortion                       |
| `Sag`           | *"tighter" responsiveness for metal and djent*           | *more touch dynamics & sustain*                 |
| `Hum`, `Ripple` | less heater hum and AC ripple                            | more; *"at higher settings, things get freaky"* |
| `Bias`          | *a "colder" Class AB biasing*                            | at maximum, Class A                             |
| `Bias X`        | *a tighter feel*                                         | *more tube compression*                         |

`Sag`, `Hum`, `Ripple`, `Bias` and `Bias X` are the largest unclear controls in
the catalog, about 650 of them together. Most of what is left is not a tone knob
but a switch or a placement, such as `TempoSync`, `Mic`, `Position`, `Angle` and
`Pan`, and has no direction to find.

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
  fix one line of a rig, never hear that mistake again. This is why
  [`pkg/sdk`](device.md) matters more than more corpus. It removes a manual HX
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
recipe       pkg/sdk/rigs/artists/mike-dirnt.yaml             who plays what
   │         amp: Ampeg SVT
   ▼
gear map     resources/schemas/gear-map.json                  gear to model
   │         HD2_AmpSVBeastNrm
   ▼
catalog      pkg/sdk/catalog/data/hx-stomp.json.gz            what the device accepts
   │         Drive 0.0–1.0, default 0.39, DSP 28.27
   ▼
grammar      pkg/sdk/corpus/data/hx-stomp.stats.json.gz       what a chain almost always holds
   │
   ▼
values       corpus medians + character                       what to set
   │
   ▼
RigSpec      validated against the catalog
   │
   ▼
.hlx         written, and put on a device over USB
   │
   ▼
a person     listens, and corrects the rig                    nothing above can hear
```

Every step above the person exists. What is still missing is narrower than it
was: a grammar that orders a whole chain rather than placing what it adds, and
values that come from measuring a recording rather than from words.
