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

| Problem                 | Source                                                 | State                                       |
| ----------------------- | ------------------------------------------------------ | ------------------------------------------- |
| Who plays what          | `pkg/sdk/rigs/`, hand-written                          | thin, grows by correction                   |
| Gear to model ID        | `resources/schemas/gear-map.json`                      | 547 models                                  |
| What order blocks go in | statistics over `resources/schemas/corpus/`            | added blocks placed; a rig's own order kept |
| Which way a knob moves  | parameter names, and the HX Edit manual's amp controls | cited below; not data yet                   |
| What values to set      | catalog defaults, corpus medians, intent               | six axes of ten                             |

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

Not equally. Strongest first, the order `tonestack recipes list` uses to name a
rig's source:

| kind     | what it is                            | worth                                                    |
| -------- | ------------------------------------- | -------------------------------------------------------- |
| `heard`  | a person played the rig and judged it | best. Nothing else here can hear                         |
| `cited`  | a published rig rundown or interview  | somebody with access wrote it down                       |
| `audio`  | figures measured from a record        | anybody with the record can check, but it holds a studio |
| `user`   | a forum thread, TalkBass or Reddit    | argued and corrected in public, and uneven               |
| `video`  | footage                               | good for how it sounds, weak for what the box was        |
| `corpus` | what other people's presets do        | says what is common, not what this player did            |
| `llm`    | a model asserted it                   | a starting point, never an answer                        |

Video sits low on purpose. A stage seen from forty feet says little about which
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

A chain's order is its signal path. Drive ahead of an amp overdrives its input,
and drive after it is a different sound, so the order a rig gives is a choice,
and a build keeps it.

What the corpus measures is narrower than a full grammar. Across the 4,324
presets measured it records, for each kind of block, how often it sits before
the amp and how often after, per instrument: across 159 bass chains, 88% hold a
compressor and 62% hold drive, which sits before the amp 88% of the time.

A build uses that for the blocks it adds. Those are the ones nearly every chain
holds, three in four or more, and each goes in as the model that instrument's
players use most, on the side of the amp where they put it. Blocks a rig names
stay where the rig put them.

It does not say which of two effects comes first. A compressor on bass sits
before the amp 52% of the time, which is not a convention to impose on anybody.
Ordering several effects against each other would need pairs measured, and that
matters only once a build generates a chain of several effects, which nothing
does yet.

## 4. What values to set

The difference between "an SVT" and "Mike Dirnt's SVT". Three inputs, in order
of authority:

1. **The catalog's official defaults.** Line 6 states a `default` for every
   parameter. This is the anchor and is never wrong.
2. **Corpus distributions.** The median `Drive` across every preset using this
   model beats a guess, and the spread says how much it varies in practice.
   Across 26 presets using the Ampeg SVT's bright channel, the median `Treble`
   is 0.85 where Line 6's stated default is 0.77. The factory default is
   measurably not what players use. On its normal channel `Bass` sits in
   0.50–0.52 and `Drive` spans 0.27–0.60, so the spread also says how much of an
   opinion is worth having.
3. **Intent.** A rig's `character` words become moves against the catalog's real
   ranges. Each word is worth one step from where the corpus left that control,
   never more than a quarter of its range, and six axes act: `mids`, `highs`,
   `drive` and `low-end` on the amp, `space` on the reverb and `attack` on the
   compressor.
   [recipes.md](recipes.md#character-describes-the-result-not-the-control) lists
   the words.

Built, for those six axes.

Four of them now have a measurement behind the word rather than somebody's
judgement: `mids` from the mid band share, `highs` from the high share and the
centroid, `drive` from how much energy sits above the fundamental, and `attack`
from how sharply notes start. Those need no general answer to which way a
control moves, because each is the same quantity the control acts on.

The other two do not, and no threshold is invented for them. `low-end` is Sag,
which is touch response and sustain, and `space` is how much room is on the
part. Nothing measured describes either, so both stay words somebody chose.

What is still missing is a population. A figure names nothing on its own: 91% of
the energy below 250Hz is not "scooped", because every isolated bass stem is
mostly low. A measurement becomes a word by sitting somewhere among other
artists measured the same way, and one artist has been measured so far.
[The measured sound profile](superpowers/specs/2026-09-16-the-measured-sound-profile-design.md)
carries the whole argument.

### Which way a knob moves

Mostly the name says. The catalog has 641 parameter names across 5,602 controls,
and the most common of them carry a name whose direction needs no explaining:
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
   │         Drive 0.0–1.0, default 0.53, DSP 26.67
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

Every step above the person exists, and so does measuring a recording:
`tonestack measure` reads an artist's records as numbers, and those numbers
reach a rig as `kind: audio` evidence carrying the figures they were measured
as.

The step from figures to words exists too, for three axes. `tonestack measure`
compares an artist against the others measured the same way, and earns them a
term only where their whole spread sits clear of the rest: mid-forward or
scooped from the mid band, bright or dark from the centroid, saturated or clean
from the harmonics. The music corpus names 5 players, which is what makes the
comparison possible at all: one artist has nothing to be clear of.

Three axes rather than the six a rig can say. `attack` is not among them,
because what is measured is how fast the level rises and the click of a pick is
spectral: the bass stems carry almost nothing above 1 kHz, so the figure that
axis needs is not in the recording. `decay` is not among them either, because it
moves by a third of a second depending on which tracks were picked, which
measures the choice rather than the player. Both findings sit beside the code
that would have used them, in `pkg/sdk/audio/derive.go`.

The records are not in this repository and neither are the measurements: the
stems are somebody's own audio files, so a figure in a rig is a claim about a
recording nobody else here can replay.

What is still missing is the step from a word to a value. A term says which way
to move a control. How far is still somebody's judgement.
