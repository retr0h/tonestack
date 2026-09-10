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

| Problem                 | Source                                      | State                     |
| ----------------------- | ------------------------------------------- | ------------------------- |
| Who plays what          | `resources/recipes/`, hand-written          | thin, grows by correction |
| Gear to model ID        | `resources/schemas/gear-map.json`           | 547 models                |
| What order blocks go in | statistics over `resources/schemas/corpus/` | not built                 |
| Which way a knob moves  | the Pilot's Guide parameter tables          | not built                 |
| What values to set      | catalog defaults, corpus medians, intent    | not built                 |

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

Not built. The corpus is collected; the measurements are not taken. They are
cheap once taken: across 169 bass-amp chains, 89% hold a compressor and 63% hold
drive, which sits *before* the amp 89% of the time.

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
3. **Intent.** A rig's `character` lines, "mid-forward, not scooped" or "grit
   only on hard attack", become directional moves against the catalog's real
   ranges. Which direction is not guesswork either: the Pilot's Guide documents
   the controls that cannot be inferred, in the same language a recipe uses. Of
   `Sag` it says *"lower values offer tighter responsiveness … higher values
   provide more touch dynamics & sustain"*; of `Bias X`, *"set low for a tighter
   feel"*. It says nothing about Drive, Bass, Mid or Treble, because those need
   no explaining.

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
recipe       resources/recipes/artists/mike-dirnt.yaml        who plays what
   │         amp: Ampeg SVT
   ▼
gear map     resources/schemas/gear-map.json                  gear to model
   │         HD2_AmpSVBeastNrm
   ▼
catalog      resources/schemas/hx-stomp.catalog.json          what the device accepts
   │         Drive 0.0–1.0, default 0.39, DSP 28.27
   ▼
grammar      statistics over resources/schemas/corpus/        what order      [not built]
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
