# Workflows

What to actually do, in order, for the things people come here to do.

This is the usage guide. Everything else under [docs/](README.md) explains how
one piece works; this says which pieces to use and when. If you are an agent
being asked for help with any of the tasks below, start here and follow the
links rather than reading everything.

Building a preset needs no device and no HX Edit, because the catalog, the
corpus statistics and the rigs are built into the binary. Only the commands that
read or write a device need a Helix plugged in, HX Edit quit, and a build with
USB support, which the released binaries do not have. The
[README](../README.md#install) says how to build one. Every command explains its
own flags with `tonestack <command> --help`.

| I want to…                              | Go to                                          |
| --------------------------------------- | ---------------------------------------------- |
| build a preset for a player             | [Create a rig](#create-a-rig-for-a-player)     |
| see what my device holds                | [Read the device](#read-what-a-device-holds)   |
| change a rig that already exists        | [Correct a rig](#correct-a-rig-you-have-heard) |
| get a preset onto the hardware          | [Load it](#get-it-onto-the-device)             |
| switch presets, or move them around     | [Switch and rearrange](#switch-and-rearrange)  |
| add knowledge from a video or recording | [Add evidence](#add-evidence-from-a-recording) |
| know what the device can do at all      | [Ask the catalog](#ask-what-is-possible)       |

## Create a rig for a player

The whole path, from a name to a file that loads.

### 1. Find out whether the gear exists

Before writing anything. A rig naming an amplifier no device models is only
discovered at build time, and by then the name has usually been copied somewhere
else.

```bash
tonestack catalog list --search ampeg
tonestack catalog list --subcategory bass --category amp
```

Names are real-world gear. "Ampeg SVT", "Klon Centaur", never model identifiers.
[catalog.md](catalog.md) explains where that mapping comes from and why it
exists.

### 2. Find out what else belongs in the chain

```bash
tonestack corpus show --instrument bass
```

This is measured over 4,324 real presets, not asserted. It says a bass chain
holds a compressor 88% of the time and that drive sits ahead of the amp in 88%
of the chains that have one. You do not have to act on it, since the build fills
in what is near-universal and says so, but it tells you what a complete chain
for that instrument looks like.

### 3. Write it

```bash
tonestack recipes new \
  --id mike-dirnt --name "Mike Dirnt" --band "Green Day" \
  --instrument bass --amp "Ampeg SVT" --cab "Ampeg 8x10"
```

Every gear name is resolved before anything is written, and near misses are
suggested when one does not resolve.

Then open the file and fill in what the flags cannot express: `character`,
`technique`, and honest `provenance`. [recipes.md](recipes.md) covers each field
and [`examples/rigspec/mike-dirnt.yaml`](../examples/rigspec/mike-dirnt.yaml)
shows all of them on one subject.

**Be honest about where the gear came from.** `source: llm` means a model
asserted it and nobody checked. That is reliable for well-known players,
unreliable for obscure ones, and the model cannot tell which it is doing. That
makes it the largest correctness risk here. Say `confidence: low` and let the
tool display it as unverified.

### 4. Build it

```bash
tonestack presets make --id mike-dirnt --out mike.hlx
```

The output is the point. It reports every block chosen, what real gear each one
emulates, what it costs, how much of the processor is used, anything added that
the rig did not ask for, and what each word in the rig's `character` did:

```text
  ●  0.0  LA Studio Comp    Teletronix® LA-2A®          5.8
  ●  0.1  Ampeg SVT Brt     Ampeg SVT (bright channel)  26.6
  ●  0.2  8x10 Ampeg SVT-E                              7.2

  dsp0  █████████░░░░░░░░░░░░░░░  39.6%

  added LA Studio Comp — almost every chain has one (88% of chains)

  heard mid-forward — Mid 0.79 to 0.89
  heard grit-on-attack — Drive 0.60 to 0.76
  heard tight-low-end — Sag 0.50 to 0.40
  heard short-decay — nothing acts on this yet
  heard audible-pick-attack — the LA Studio Comp has no Attack
```

[recipes.md](recipes.md#character-describes-the-result-not-the-control) explains
the `heard` lines.

Read it before you plug anything in. A wrong amp is a bad miss that nothing
downstream recovers from, and it is visible right there.

### 5. Get it onto the device

See [below](#get-it-onto-the-device).

## Read what a device holds

```bash
tonestack presets list
```

That reads the attached device over USB. **Quit HX Edit first.** It claims the
editor interface exclusively and nothing else can talk to the device while it
runs.

It is read-only: it asks the device to describe a setlist and nothing more.
Nothing is selected, loaded or written. [protocol.md](protocol.md) covers how,
and carries the rules that keep a device alive if you are working on that code.

`presets show` and `presets export` read the hardware too, and give the slot
back as a rig:

```bash
tonestack presets show   --slot 31A
tonestack presets export --slot 31A --out lead.yaml
```

A slot is addressed the way the pedal labels it, `01A` through `42C`. A bare
number works too, for scripts.

Every reading command also takes `--file`, for a backup HX Edit wrote when no
device is attached:

```bash
tonestack presets list --file device.hlb
tonestack presets show --file device.hlb --slot 31A
```

Putting a preset *onto* a device is the part that is not live. See
[Get it onto the device](#get-it-onto-the-device).

A rig read off the device carries its routing, so compiling one puts the
device's own inputs, outputs, split and join back. Controller assignments are
the exception: nothing has decoded them yet, and a rig read over USB carries
none. Reading the same slot out of a backup carries them.
[protocol.md](protocol.md) states what the device answers today and what is
still unknown.

## Move a rig between formats

RigSpec is what this project speaks, so it is what `export` writes. Reading a
slot still means reading a backup rather than the device:

```bash
tonestack presets export --file device.hlb --slot 3 --out slot3.yaml
tonestack presets compile --rig slot3.yaml --out slot3.hlx
```

Out and back. A preset read into a rig and compiled again is the preset it came
from. That is asserted over every HX Stomp preset in the corpus, so it is a
measurement rather than a claim.

Two things make that work, and both matter if you hand-edit a rig in between.

`slot3.yaml` is a rig, the same format `recipes new` writes and `presets make`
reads. A lifted rig records `models: { HX Stomp: HD2_... }`, the exact model
each piece of gear resolved to. **665 models share only 469 names**, and "Ampeg
SVT" matches both channels, so a rig carrying the name alone would rebuild into
a different preset. Delete that line and compiling falls back to resolving the
name, which is right for a rig you wrote and wrong for one you lifted.

Compiling writes the chain into an untouched preset the device itself wrote, so
the result carries the inputs, outputs, split and join a device expects. 98.6%
of real presets have them and one assembled from nothing has none. `--template`
uses a particular preset as that base instead.

For a faithful copy rather than a reading, `--as hlx` writes the device's own
file, which also carries the routing and snapshots a rig models but nobody
chooses.

## Get it onto the device

Three ways, depending on what you have open.

**Straight to the device.** Plug in the Helix and quit HX Edit:

```bash
tonestack presets import --preset mike.hlx --slot 07A
```

```console
  kept ~/.local/state/tonestack/presets/07A-20260910-041500.hlx

  Mike Dirnt → 07A

  written
```

A device has no undo, so whatever the slot held is read and saved first, and the
output says where. Put it back with `presets import --preset` and that file.
`--backup-dir` changes where they go.

**Through HX Edit.** `HX Edit → Import` and choose the `.hlx`.

**Into a backup**, with no device attached:

```bash
tonestack presets import --file device.hlb --preset mike.hlx \
  --slot 07A --out edited.hlb
# HX Edit → Restore
```

Nobody has yet confirmed that a device loads a generated preset and it sounds
right. The build validating against the catalog is the only claim this project
can make today; [device.md](device.md) covers what is known about writing.

## Switch and rearrange

These need the Helix plugged in and HX Edit quit, or `--file` to work on a
backup instead.

```bash
tonestack presets select --slot 27B              # load it, like a footswitch
tonestack presets copy   --from 01A --to 02A     # 02A becomes a copy of 01A
tonestack presets swap   --from 01A --to 02A     # exchange the two
```

`select` writes nothing. `copy` and `swap` keep what the destination held first,
the same way `import` does. Moving a preset is a swap: leaving the source empty
would mean inventing an empty preset, and an empty preset still carries routing
that differs by model and firmware.

## Correct a rig you have heard

This is the loop that matters, because **nothing here can hear**. Every other
input is a measurement or an assertion; you are the only thing that can say
whether it sounds right.

1. Play it.
2. Say what is wrong in your own words. "Too clunky", "the drive is muddy".
3. Change the rig.
4. **Record what you asked for, what changed, and what you thought of it.**

That last step is the one people skip and the only one that cannot be recovered
later. A catalog can be regenerated next year; nobody can go back and ask
themselves what they thought of round three.

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

`reason` is where an agent records its *interpretation*, cited. If it read
"clunkier" as drive when you meant sag, that line is what shows the reading was
wrong rather than only the value.

If several rounds of settings changes all come back negative, stop turning
knobs. Repeated failure at that layer is evidence against something higher up,
usually the amp.

## Add evidence from a recording

A rig gets more trustworthy by having its claims made checkable, not by someone
asserting them more confidently.

```yaml
- role: amp
  gear: Ampeg SVT
  evidence:
    - { kind: cited, url: "…", note: "Bass Player interview, 2004" }
    - { kind: video, url: "…", at: "1:42", note: "SVT visible on stage" }
  confidence: high
```

Evidence attaches to a claim, not to the document, because the amp may come from
an interview and a drive figure from measuring a corpus.

**A URL does not make a claim true.** It makes it checkable, which is what lets
somebody correct one line instead of re-deriving a rig.

Audio evidence carries measurements rather than conclusions, and that
distinction is load-bearing: measuring a record measures the bass, the player,
the amp, the mic, the desk, the master and the encoder. Those numbers compare
against the same measurement taken from a generated preset, giving a direction.
They are not knob positions. The `caveat` field is where a rig says which of the
two it means.

## Ask what is possible

```bash
tonestack catalog list --category amp --subcategory bass
tonestack catalog show --model HD2_AmpSVBeastNrm
tonestack corpus show --model HD2_AmpSVBeastNrm
tonestack devices list
```

The first two are what the device *can* do; the third is what people *do* with
it. They answer different questions and neither substitutes for the other. Line
6 state a default Treble of 0.68 for an Ampeg SVT; the median across every
measured use is 0.845. Both are facts.

On `corpus show --model`, the **spread** is the useful column. A parameter
everybody sets the same way is one this tool can be confident about; one nobody
agrees on belongs to you.

## For agents

Two rules beyond the workflows above.

**Say which claim you have.** "The rig validates against the catalog", "HX Edit
imported the file" and "the hardware loaded it" are three different claims, and
only the first is currently possible here. Do not report one as another.

**Never invent a model identifier.** If `catalog list --search` does not find
the gear, the device does not model it. Say so and suggest what it does have,
rather than writing an identifier that looks plausible.
