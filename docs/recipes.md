# Writing a rig

How to describe what somebody plays, so this project can build it.

A rig is written as a **RigSpec**, the project's only hand-authored format,
defined in
[`pkg/sdk/rig/data/rigspec.openapi.yaml`](../pkg/sdk/rig/data/rigspec.openapi.yaml).
One is a YAML file under `pkg/sdk/rigs/`, and it is the only data here that is
ours: the device catalog and the gear map are derived from Line 6's own files,
while these are written by hand.

This page is how to write one. For what each field may say, read
[`docs/rigspec.md`](rigspec.md), which is generated from the contract and lists
every field, its grammar and its allowed values.

Two rigs show the format between them, and they are worth reading together.
[`examples/rigspec/mike-dirnt.yaml`](../examples/rigspec/mike-dirnt.yaml) is
somebody saying what they want, with evidence, confidence and character.
[`examples/rigspec/dir-angl-meteor.yaml`](../examples/rigspec/dir-angl-meteor.yaml)
is a device saying what it has: produced by `presets show` from a preset an HX
Stomp wrote, committed exactly as it came out, and carrying the footswitches,
snapshots and routing no hand-written rig needs.

A test holds the two of them to the contract. Every field it declares is written
down in one of the rigs here, or named in that test with a reason nobody can
write it, so a field added without an example fails the build.

## The smallest useful rig

```yaml
schema: RigSpec
version: 2
id: mike-dirnt
subject: { kind: artist, name: Mike Dirnt, band: Green Day }
instrument: bass
chain:
  - { role: amp, gear: Ampeg SVT }
  - { role: cab, gear: Ampeg 8x10 }
```

That builds. Everything else is optional, and earns its place by making a claim
checkable later.

## Name gear the way a person would

`gear: Ampeg SVT`, never `HD2_AmpSVBeastNrm`.

Line 6 rename every model for trademark reasons and publish the mapping in their
own manual; joining the two is what [catalog.md](catalog.md) does. Naming the
identifier instead would tie a rig to one manufacturer, break when they rename a
model, and stop it being read on other hardware, which is the whole reason the
format exists.

Check a name resolves before trusting it:

```bash
tonestack catalog list --search ampeg
```

Nothing back means the device does not model that gear. Either name something it
has, or declare the gap in `requires`.

## Order is the signal path

`chain` is ordered, and the order is what the signal does. Drive ahead of an amp
overdrives its input; drive after it is a different sound entirely.

You do not have to guess the conventional order. It is measured:

```bash
tonestack corpus show --instrument bass
```

Across 4,324 real presets: bass chains hold a compressor 88% of the time, drive
sits before the amp in 88% of the chains that have one, and a cabinet follows
the amp in all but 1%.

A rig omitting something near-universal gets it added during the build, and the
build says so rather than doing it quietly.

## Settings are musical, and deliberately lossy

```yaml
settings: { drive: 0.47, bass: 0.52, mid: 0.71, treble: 0.85 }
```

A small vocabulary from 0 to 1: `drive`, `bass`, `mid`, `treble`, `presence`,
`level`, `mix`, `feel`. Each means roughly the same thing on any amplifier.

**Device controls do not belong here.** `Sag`, `Bias X`, `Ripple` and `Hum` are
one manufacturer's knobs; the compiler sets those from catalog defaults, corpus
medians and the manufacturer's own documentation of what they do. A rig carrying
them would not survive being read on other hardware.

Leaving `settings` out is fine, and often better. What happens then:

1. Line 6's stated default is the floor, and is never invalid.
2. Where the corpus shows players agreeing closely, the median replaces it. For
   an Ampeg SVT that moves `Treble` from Line 6's 0.68 to 0.845.
3. Where players disagree, the default stands, rather than an average of
   disagreement being presented as a measurement.

`tonestack corpus show --model HD2_AmpSVBeastNrm` shows the median and the
spread. The spread is the useful column: it says how much of an opinion is worth
having.

## Character describes the result, not the control

Each term is one thing, from a list that ships beside the catalog. A term is one
claim, so a rig saying two things says two of them:

```yaml
character:
  - term: mid-forward
    evidence:
      - kind: llm
  - term: short-decay
```

The words, grouped by the axis each belongs to. An axis is what makes a term
mean something: saying `mid-forward` has already said "not scooped", and a rig
claiming both has claimed nothing.

| axis           | what it says                                        | words                                                      |
| -------------- | --------------------------------------------------- | ---------------------------------------------------------- |
| `attack`       | what the front of a note sounds like                | `audible-pick-attack` · `percussive` · `soft-attack`       |
| `decay`        | how long a note lasts once it is struck             | `long-decay` · `short-decay`                               |
| `drive`        | how hard the amplifier is being pushed              | `clean` · `grit-on-attack` · `minimal-drive` · `saturated` |
| `highs`        | the top, and how much of it there is                | `bright` · `dark` · `glassy`                               |
| `low-end`      | how the bottom behaves, not how much of it there is | `loose-low-end` · `tight-low-end`                          |
| `mids`         | where the middle of the range sits                  | `mid-forward` · `scooped`                                  |
| `movement`     | whether the tone stays put while a note lasts       | `envelope-swept` · `static`                                |
| `pickup`       | where along the string the sound is taken from      | `bridge-forward` · `neck-forward`                          |
| `space`        | what surrounds the part                             | `dry` · `roomy`                                            |
| `string-noise` | what the hands make that is not a note              | `audible-strings` · `quiet-strings`                        |

Six of those axes turn a knob. `mids`, `highs`, `drive` and `low-end` move the
first amplifier in the chain, `space` moves the first reverb and `attack` moves
the first compressor. Each word is worth one step from where the corpus left
that control, so a build reports what it heard and what it did about it:

```console
$ tonestack presets make --id mike-dirnt
  heard mid-forward — Mid 0.79 to 0.89
  heard grit-on-attack — Drive 0.60 to 0.76
  heard tight-low-end — Sag 0.50 to 0.40
  heard short-decay — nothing acts on this yet
  heard audible-pick-attack — Attack 0.04 to 0.05
```

The `short-decay` line is this project's gap: nothing anywhere acts on that
word. A word can also go unanswered because of the rig, and the build says that
differently. A rig saying `roomy` in a chain that holds no reverb gets:

```console
  heard roomy — this chain holds no reverb
```

That one is worth knowing about the rig, because adding a reverb would answer
it.

Say two words from one axis and you have answered one question twice. Applying
both lands the knob where it started, so neither is applied and the build names
the axis instead:

```console
  heard minimal-drive — another term already answered for drive, so neither moved
  heard grit-on-attack — another term already answered for drive, so neither moved
```

There is a third thing a word can do, which is nothing at all because the chain
is already what it asked for:

```console
  heard dry — this chain has no reverb, so it is already dry
```

The other four axes, `decay`, `string-noise`, `pickup` and `movement`, describe
the player and the instrument rather than the rig. They are recorded and move
nothing.

A word that is not on the list is reported and not refused:

```console
$ tonestack presets make --id mine
  note no such character term "tight low end" — did you mean tight-low-end?

  [ok] wrote mine.hlx
```

The preset is written, because refusing a word would be refusing you the right
to describe a sound. If the word you want is missing, add it to
[`pkg/sdk/internal/compile/data/character-terms.json`](../pkg/sdk/internal/compile/data/character-terms.json)
with a sentence saying what it means. The rigs this repository ships are held to
the list by a test, which is what keeps the examples from drifting back into
sentences.

## Say where each claim came from

This is what makes a rig correctable rather than merely a guess.

```yaml
- role: amp
  gear: Ampeg SVT
  evidence:
    - { kind: cited, url: "…", note: "Bass Player interview" }
    - { kind: video, url: "…", at: "1:42", note: "SVT visible on stage" }
  confidence: high
```

Evidence attaches to a **claim**, not to the document, because the amp may come
from an interview and a drive figure from measuring a corpus. `kind` is open:
`llm`, `cited`, `video`, `audio`, `corpus`, `measured`, `user`.

Two things to be clear about.

**A URL does not make a claim true.** It makes it *checkable*. That is what lets
somebody correct one line instead of re-deriving a rig, and it is why a
correction is a small reviewable diff.

**`llm` means nobody checked.** A language model is good at well-known players
and confabulates for obscure ones, and cannot reliably tell which it is doing.
That is the largest correctness risk in this project. A rig sourced that way
should carry `confidence: low` and be shown as unverified whatever it claims
about itself.

## Declare what the device does not ship with

```yaml
requires:
  - { kind: ir, name: Ownhammer SVT 8x10, slot: 82, url: "…" }
```

Only what a catalog cannot see. Whether a model exists on a device tier, or
needs newer firmware, is already known, because the catalog carries the
supported device list and the release it came from.

The case nothing can know is an impulse response. A preset stores the **slot
number**, never the audio, so a rig depending on slot 82 sounds like whoever
made it only if the same IR is loaded there. That is why generated chains never
reach for a user IR block, and why one is flagged when reading somebody else's
preset.

## Record what you thought of it

```yaml
mutations:
  - ask: make it clunkier
    changed:
      - { path: chain[1].settings.drive, from: 0.47, to: 0.58 }
    reason: >-
      Line 6 document Sag as "lower values offer tighter responsiveness…
      higher values provide more touch dynamics & sustain". Read "clunky" as
      a looser power-amp feel rather than more gain.
    verdict: closer, but muddy now, so keep the feel and put the drive back
```

Nothing in this project can hear. Every other input is a measurement or an
assertion, and **the verdict is the only place a human ear is written down**. It
is also the only thing unrecoverable later: a catalog can be regenerated next
year, and nobody can go back and ask themselves what they thought of round
three.

Four fields, four jobs:

- `ask` is your words, verbatim. "Clunky" is not a parameter, and normalising it
  away loses the question.
- `changed` is what moved.
- `reason` is how the ask was interpreted, cited. If the reading was wrong, this
  is the line that shows it, rather than only that the value was.
- `verdict` is what it sounded like. Absent means not yet heard, which is useful
  state.

Append-only, never replayed: `chain` always holds the current state.
Reconstructing a rig from its history would be more elegant and much worse to
read, and a person reads this file.

## One artist, several rigs

A player's rig changes by era and by song, and two can differ at the amp, which
makes them siblings, not variations. Each is a complete RigSpec with its own
`id`; one carries `default: true`, because asking for "a Mike Dirnt sound" with
no qualifier has to land somewhere. Use `extends` only where a rig genuinely is
a small departure from another.

### `extends` records lineage, and nothing merges

This is the part people expect to work the other way, so it is worth saying
plainly: **a rig that extends another still holds everything itself.** Nothing
is inherited, nothing is looked up at build time, and deleting the parent leaves
the child working. All `extends` does is record where the rig came from, which
is what lets `recipes show` list a rig's variants underneath it.

The reason is that a rig is meant to be read. If a file only held its
differences, the rig that compiled would not be the rig on the page, and
answering "why is this amp here" would mean opening two files and knowing the
merge rules. Duplication across nine rigs is cheaper than that.

So copy the parent and edit the copy:

```bash
tonestack recipes new --from flea \
  --id flea-under-the-bridge --kind song --name "Under the Bridge"
```

That writes a whole rig: the chain, the character, the comments and every
citation, with `extends: flea` recorded and the parent's `aliases` and `default`
dropped, since those belong to the parent alone.

The citations coming across is the point and also the trap. A claim sourced for
one rig is not evidence for another, so anything you change loses its evidence
with it. The file says so at the top, and the honest move is to drop what you
cannot stand behind rather than leave a citation pointing at a rig that no
longer exists.

## Checking your work

```bash
tonestack recipes show --id mike-dirnt        # read it back
tonestack catalog list --search "ampeg svt"   # does the gear resolve?
tonestack corpus show --instrument bass       # what else belongs in the chain?
tonestack presets make --id mike-dirnt --out mike.hlx
```

The build reports every block it chose, what real gear each emulates, what it
costs, and anything it added the rig did not ask for. A wrong amp should be
visible before anyone plugs in rather than after.

## Rigs read off a device carry more

Everything above describes a rig somebody writes. One lifted from a preset by
`presets export` carries three more fields, and they exist so that reading a
preset into a rig and writing it back gives the preset it came from. That is
asserted over every HX Stomp preset in the corpus.

```yaml
- role: amp
  gear: Ampeg SVT® (bright channel)
  models: { HX Stomp: HD2_AmpSVBeastBrt }
  position: 1
  params:
    Drive: 0.53
    Sag: 0.5
    "@type": 7
```

`models` records what the gear actually resolved to, keyed by device. This is
not belt and braces: **665 models share only 469 names**, and "Ampeg SVT"
matches both channels, so a rig carrying the name alone would rebuild into a
different preset. A device with no entry falls back to the name, which is the
portable behaviour and why the name is still required.

`params` are device parameters under their own names, `Sag` and `Bias X`, as
distinct from `settings`, which is the small musical vocabulary that means
something anywhere. They are here because somebody dialling `Sag` by ear is
producing the one kind of knowledge nothing else can, and dropping it to stay
portable would throw away exactly what is worth keeping.

**A rig stating parameters means exactly those.** No catalog defaults are added
on top, because the block was described completely. A rig stating none is
describing gear rather than a block, and takes Line 6's defaults.

`position` is where a block sits on the device's grid, which is not always its
order in the chain. A preset can hold `block5` whose position is 6.

## What none of this can tell you

Whether it sounds right. That is a person with the preset loaded, and the answer
belongs in `mutations` so the next round starts from it rather than from
nothing. [knowledge.md](knowledge.md) explains why the architecture is shaped
around that.

## What a rig keeps

A rig describes a sound, and it also carries everything else the preset it came
from held. That is what makes it safe to be the only thing this project
exchanges: a rig read out of a preset rebuilds that preset exactly, with the
original file gone.

| field          | what it holds                                             |
| -------------- | --------------------------------------------------------- |
| `chain`        | the gear, in order, with settings and evidence            |
| `snapshots`    | what each footswitch recalls: name, tempo, block states   |
| `footswitches` | what the pedal prints under each switch, and its colour   |
| `device`       | everything else, verbatim: routing, controllers, metadata |

`snapshots` and `footswitches` are modelled rather than kept verbatim, because
they are musical decisions somebody made and might want to change. `device` is
the remainder: entries a person would not hand-edit, kept exactly as they
arrived so a field nobody has modelled yet is not a field this drops.

Only a lifted rig carries `device`. One somebody typed has none, and compiling
it uses an untouched preset the device itself wrote.

### How that is known to hold

Three assertions over every HX Stomp preset in the corpus, all of which must
pass:

1. **A preset becomes a rig and is written back into itself**, byte for byte.
2. **A preset becomes a rig and is built into an untouched preset**, byte for
   byte, which is the path a rig takes when somebody shares it.
3. **A rig becomes a preset and is read back as a rig**, the same rig.

The third is the one that catches a field this format reads but never writes.
Such a field survives the first two, because the preset underneath still holds
it, and disappears in the third because the rig is all there is.

## A footswitch's colour

Stored as a packed number, and not a free choice. Measured over 17,665
assignments in the corpus, a switch takes the colour of the block it works on
unless somebody sets it: red for an amp or a cabinet, amber for drive, green for
delay, blue for modulation, purple for a filter, pitch block or wah, orange for
reverb, lime for a compressor or EQ.

Each is one hue at two brightnesses, bright while the block is engaged and dim
while it is bypassed, which is why a palette covering twelve categories holds
twenty-four values.

## Where the contract lives

The schema is at
[`pkg/sdk/rig/data/rigspec.openapi.yaml`](../pkg/sdk/rig/data/rigspec.openapi.yaml),
embedded into the package that reads it. It sits beside that code rather than in
a directory of its own so the library can be lifted out without somebody
remembering to bring a file along.

RigSpec is this project's own invention. The Line 6 format has no equivalent. It
stores blocks under `dsp0`/`block0` keys with no abstraction over where a chain
came from. RigSpec exists so every input converges on one validated shape before
anything writes a file.

It is the contract in both senses. `pkg/sdk/rig/gen` is generated from it by
`oapi-codegen`, and `rig.Validate` checks a document against the same file
rather than against a second copy of the rules written in Go. Two copies drift:
a constraint added to one becomes a type nothing enforces, or a check nothing
asked for.

There used to be two contracts, one for what a person writes and one for what
the generator produces. They were the same document at two levels of detail, so
now there is one. A rig is sparse when somebody types it and full once it has
been compiled or lifted from a preset.

### Why RigSpec does not enumerate models

RigSpec is the *shape* of a signal chain and is stable across devices and
firmware. Which models exist belongs to a device at a firmware version, and that
is the catalog's job.

Enumerating models inside RigSpec would tie its version to the firmware, make a
preset using an unlisted model unrepresentable, and produce a schema tens of
thousands of lines long. A per-device schema with the model enum inlined can be
*generated* from the two when strict validation is wanted.

### Generating clients

The schema is the source for anything that needs to speak RigSpec, including
this project's own Go types:

```bash
just generate    # regenerates pkg/sdk/rig/gen
npx openapi-typescript pkg/sdk/rig/data/rigspec.openapi.yaml -o rigspec.d.ts
```
