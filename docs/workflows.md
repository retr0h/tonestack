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
own flags with `tonestack <command> --help`, and [commands.md](commands.md)
lists them all.

| I want to…                              | Go to                                          |
| --------------------------------------- | ---------------------------------------------- |
| build a preset for a player             | [Create a rig](#create-a-rig-for-a-player)     |
| see what my device holds                | [Read the device](#read-what-a-device-holds)   |
| change a rig that already exists        | [Correct a rig](#correct-a-rig-you-have-heard) |
| get a preset onto the hardware          | [Load it](#get-it-onto-the-device)             |
| switch presets, or move them around     | [Switch and rearrange](#switch-and-rearrange)  |
| add knowledge from a video or recording | [Add evidence](#add-evidence-from-a-recording) |
| measure records a player actually made  | [Measure a sound](#measure-a-players-sound)    |
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

The file goes to your own recipes directory, `$XDG_DATA_HOME/tonestack/recipes`,
or `~/.local/share/tonestack/recipes` when that variable is unset, and the
command prints its path. `recipes list`, `recipes show`, `presets make` and the
MCP server read that directory beside the rigs that ship, so the new rig builds
straight away. `--dir` writes somewhere else instead. A rig written there is
only found when you name the same directory again, with `--dir` or with
`presets make --recipes`, and the rigs that ship are read beside it either way.

`--kind` changes what a copy made with `--from` is attributed to, and is refused
without `--from`. A rig written from gear is always an artist.

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
  ●  0.0  Deluxe Comp       Line 6 Original             1.8
  ●  0.1  Ampeg SVT Brt     Ampeg SVT (bright channel)  26.6
  ●  0.2  8x10 Ampeg SVT-E                              7.2

  dsp0  ████████░░░░░░░░░░░░░░░░  35.6%

  added Deluxe Comp — almost every chain has one (88% of chains)

  heard mid-forward — Mid 0.79 to 0.89
  heard grit-on-attack — Drive 0.60 to 0.72
  heard tight-low-end — Sag 0.50 to 0.40
  heard short-decay — nothing acts on this yet
  heard audible-pick-attack — Attack 0.04 to 0.05
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
  kept ~/.local/state/tonestack/presets/07A-s0-20260910-041500.129384756.hlx

  Mike Dirnt → 07A

  written
```

A device has no undo, so whatever the slot held is read and saved first, and the
output says where. Put it back with `presets import --preset` and that file.
`--backup-dir` changes where they go. A backup never replaces a file already in
that directory. One that can't be written in full, or whose directory can't be
synced to disk, leaves no file behind, and nothing is written to the pedal.

Ctrl-C does not cut a write off halfway, because a half-sent message stalls the
pedal. The write finishes, then the command tells the pedal the session is over
and waits for it, which can take up to about 20 seconds. The first Ctrl-C prints
that the pedal is being let go safely, and the second prints that it is still
finishing. A third quits on the spot. That leaves the pedal believing an editor
is still attached, and it may need its power unplugged to recover, as
[protocol.md](protocol.md#rules-that-keep-a-device-alive) explains. Every
command that talks to the device answers Ctrl-C this way, not only `import`.

`tonestack mcp start` answers Ctrl-C the same way while it holds the pedal. It
holds the pedal from an agent's device tool call until the agent has made none
for 10 seconds. A Ctrl-C at any other time prints nothing, and the server stops
straight away.

A slot with no blocks is kept as a `.bin` file instead. It holds the bytes the
device sent, so nothing is lost. `presets import --preset <file>.bin` puts one
back: the bytes go to the slot exactly as they came off, and the slot keeps the
name it has, because a `.bin` carries none.

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
the same way `import` does. Moving a preset is a swap: when one of the two slots
holds no preset, the preset lands there and the slot it came from is emptied,
which is a move and invents nothing.

A swap of two slots that both hold no preset is refused, and nothing is kept or
written, because there is nothing to move. `copy` fills an empty slot and leaves
the source as it is.

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

## Measure a player's sound

Reading numbers off records somebody actually made, so a rig can carry
measurements rather than adjectives. Needs uv; see
[Prerequisites](../CONTRIBUTING.md#prerequisites).

The short version: a mix measures the band, so the bass has to come out of it
first. Then measure several records rather than one, because one mix is one
engineer's decisions on one day.

### 1. Collect what the player played on

Put the records in a directory. They stay there: audio is never committed, the
same way the preset corpus under `resources/schemas/corpus/` is fetched and
git-ignored rather than redistributed.

```bash
ls ~/music/mike-dirnt/
# basket-case.mp3  brain-stew.mp3  longview.mp3  when-i-come-around.mp3
```

Four is a reasonable minimum. Fewer cannot tell a player's habit from one song's
mix.

### 2. Separate the bass from each

```bash
just stems ~/music/mike-dirnt ~/stems
```

About 15 seconds a track once the model is cached, so a four-record corpus is a
minute. The recipe wraps Demucs, which is Python; that is the same category as
`ffmpeg` converting an MP3, and nothing downstream of it leaves Go.

Two files come out per track. `bass.wav` is the instrument and `no_bass.wav` is
everything it was taken out of, which the next step ignores.

**Do not skip this and filter instead.** A centre-channel extraction with a
250Hz low-pass looks like it works — 92% low, a centroid of 154Hz — and both
numbers are circular, because everything above 250Hz was thrown away. The tell
is decay, which comes back at 0.13s. A bass note does not stop that fast. Kick
drum and bass share those frequencies and no filter separates them.

### 3. Measure them together

Point `--dir` at the tree the separation wrote. Each stem is named for the
directory holding it, because four rows reading `bass` would name nothing.

```bash
tonestack measure --dir ~/stems/htdemucs
```

```text
  Each record  4 recordings

  RECORD              LOW  CENTROID  TRANSIENT   DECAY  DYNAMICS  HARMONICS
  basket-case         97%    151 Hz       0.79  1.72 s    5.3 dB  23%
  brain-stew          97%    135 Hz       0.75  0.15 s    5.9 dB  18%
  longview            91%    175 Hz       0.74  0.82 s    7.8 dB  35%
  when-i-come-around  91%    195 Hz       0.72  1.22 s    7.1 dB  20%

  Across the records  4 recordings

  MEASURE    MIDDLE  ACROSS THE RECORDS
  low           97%  91%–97%
  mid            8%  3%–9%
  high           0%  0%–1%
  centroid   175 Hz  135 Hz–195 Hz
  transient    0.75  0.72–0.79
  decay      1.22 s  0.15 s–1.72 s
  dynamics   7.1 dB  5.3 dB–7.8 dB
  harmonics     23%  18%–35%
  lean        +0.90  above the fundamental, leaning even
```

### 4. Read the width, not just the middle

The same four records measured as full mixes give centroids from 842Hz to
1338Hz. Separated, they sit between 135Hz and 195Hz. That tightening is the
whole reason for step 2.

A middle with a narrow width is a habit. A middle with a wide one is four
different decisions averaged into a number nobody played, and `decay` above is
exactly that: 0.15s to 1.72s is not a player's tendency, it is four takes
disagreeing. `brain-stew` reports 0.15s because its stem is half silence and the
loudest note is followed immediately by a rest.

With four records the ends of each width are the extreme records rather than a
tenth in from them, so one unusual take is the whole of one end. Treat a figure
that disagrees with the rest as a question rather than an answer.

### 5. Write it into the rig

Evidence carries the measurement and a link naming the recording. The audio
itself is never referenced: somebody who does not own the record can still see
what was measured and disagree, and somebody who does can re-measure.

#### Name the records first

The measurement knows what the audio sounds like and nothing about where it came
from. A manifest supplies that, and it is the half that can be committed: the
audio is somebody else's, and a link is checkable by a person who does not have
the file.

```yaml
# ~/music/mike-dirnt/corpus.yaml
artist: Mike Dirnt
tracks:
  - track: longview
    url: https://open.spotify.com/track/…
    at: "1:20-1:45"
    note: the bass carries the verse alone
  - track: basket-case
    url: https://open.spotify.com/track/…
```

`track` matches the stem directory, which is the source file's own name. Commit
this file; git-ignore the audio and the stems beside it, the same way
`resources/schemas/corpus/` keeps its fetch script and drops its payload.

An unknown field is refused rather than ignored, because `track` and `tracks`
are one letter apart and a manifest that silently measures nothing is worse than
one that stops. A timestamp that is not a timestamp is refused here too: left
until build time it surfaces as a failure against `chain[0].evidence[1].at`,
which does not say which song was wrong.

#### Then write the block

Do not type the numbers. `--evidence` writes them:

```bash
tonestack measure --dir ~/stems/htdemucs --manifest ~/music/mike-dirnt/corpus.yaml --evidence
```

Anything the two disagree about is reported on stderr, so redirecting stdout
into a rig stays clean:

```text
named in the manifest but not measured: nothing-was-separated
measured but not in the manifest, so their evidence carries no link: brain-stew
```

```yaml
# Paste under a chain entry's evidence:, and add a url: to each naming the
# recording it came from.
- kind: audio
  note: longview, bass isolated from the mix before measuring
  caveat: >-
    measures the record rather than the player: the amp, the mic, the desk and
    the master are all in these numbers
  measured:
    low: 0.91
    mid: 0.09
    high: 0
    centroid: 175
    transient: 0.74
    decay: 0.82
    dynamics: 7.8
    harmonics: 0.35
    lean: 0.73
```

One entry per record, not one for the corpus. A url is what makes a claim
checkable, and a single entry averaging four records is the one thing nobody
could check. Add the url yourself: the tool measured the audio and has no idea
which release it came from.

The keys are fixed, and that is the whole point of them. These figures exist to
be compared against the same figures taken from a generated preset, and that
comparison is only possible when both sides call a thing by the same name. The
contract accepts any key, the way it accepts any evidence `kind`; these are what
the tool writes.

`caveat` is doing real work too. These figures describe a finished record, so
they compare against another measurement. They are not knob positions.

## Ask what is possible

```bash
tonestack catalog list --category amp --subcategory bass
tonestack catalog show --model HD2_AmpSVBeastNrm
tonestack corpus show --model HD2_AmpSVBeastNrm
tonestack devices list
```

The first two are what the device *can* do; the third is what people *do* with
it. They answer different questions and neither substitutes for the other. Line
6 state a default Treble of 0.77 for the Ampeg SVT's bright channel; the median
across the presets using it is 0.85. Both are facts.

On `corpus show --model`, the **spread** is the useful column. A parameter
everybody sets the same way is one this tool can be confident about; one nobody
agrees on belongs to you.

## Use it from an agent

`tonestack mcp start` gives an agent the same operations as tools, with typed
results instead of text to parse. Claude Code starts it for you once it is
added:

```bash
claude mcp add tonestack -- tonestack mcp start
```

That offers everything except writing to a pedal. To let the agent import, copy
and swap slots, add it with `tonestack mcp start --allow-writes` instead. Each
write still saves what it replaces to a file first. Selecting a slot works
either way.

The same flag decides whether `preset_build` and `preset_export` may write over
a file already at the path the agent names. Without it they refuse, and the
refusal is part of the write: a file that appears while a build or an export is
running is kept too. With it they replace the file, as `presets make` and
`presets export` always do.

The agent sees the rigs `recipes list` shows, your own recipes beside the ones
that ship. `rigs_list`, `rig_show` and `preset_build` reach a rig you wrote with
`recipes new` as soon as the file is there.

Quit HX Edit before asking for anything that reaches the pedal.

## For agents

Two rules beyond the workflows above.

**Say which claim you have.** "The rig validates against the catalog", "HX Edit
imported the file" and "the hardware loaded it" are three different claims, and
only the first is currently possible here. Do not report one as another.

**Never invent a model identifier.** If `catalog list --search` does not find
the gear, the device does not model it. Say so and suggest what it does have,
rather than writing an identifier that looks plausible.
