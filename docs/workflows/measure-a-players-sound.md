# Measure a player's sound

Reading numbers off records somebody actually made, so a rig can carry
measurements rather than adjectives. Needs uv; see
[Prerequisites](../../CONTRIBUTING.md#prerequisites).

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

An agent can fetch them itself.
[Add records to a corpus](add-records-to-a-corpus.md) covers choosing,
downloading and checking each one.

Run every step here from the main checkout, not a git worktree. The corpus is
ignored by git, so a worktree's copy of `resources/music/` is a separate
directory that is deleted with the worktree, taking the audio with it. See
[Keeping it](../../resources/music/README.md#keeping-it).

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

### 6. Compare them against the other players

A figure on its own is a fact about a recording. A word is a claim about a
player, and it is earned by sitting clear of everybody else:

```bash
tonestack measure --corpus resources/music/bass
```

```
  What the records say  9 players

  PLAYER          RECORDS  EARNS                AGAINST THE OTHERS
  bootsy-collins        4  mid-forward          mid 23% against 4%
  flea                  3  clean                harmonics 12% against 24%
  geddy-lee             4  mid-forward          mid 14% against 4%
  jaco-pastorius        4  mid-forward, bright  mid 39% against 4%; centroid 259 Hz against 170 Hz
  les-claypool          4  nothing
  mike-dirnt            3  nothing
  paul-mccartney        3  clean                harmonics 10% against 24%
  pino-palladino        3  dark                 centroid 96 Hz against 176 Hz
  tim-commerford        3  nothing
```

One directory per player, each holding that player's separated records. This is
the only mode that produces words, because one player has nobody to be clear of.

`nothing` is the ordinary answer and not a failure. A player earns a term only
where their whole range sits outside the middle half of the others, so a player
whose records disagree with each other earns nothing, and so does one who sits
where everybody else sits. Les Claypool reads that way today: one of his three
records is far darker than the other two.

Mike Dirnt earned `mid-forward` against four players and earns nothing against
eight. Nothing about his records changed. Bootsy Collins and Jaco Pastorius sit
further into the mids than he does, so the middle of the population moved and
what used to be clear of it no longer is. A word lost this way was never
evidence; it was a small population.

It moves both ways. Geddy Lee earned nothing against seven players and earns
`mid-forward` against eight, because Tim Commerford arrived at 2% and pulled the
middle down. The three of them claim the same amplifier, an Ampeg SVT into an
8x10, and read 2%, 8% and 14%: the same rig measured three ways, which is the
answer to whether this corpus measures the player or the box.

Three axes are derived — `mid-forward`/`scooped`, `bright`/`dark` and
`saturated`/`clean` — and they are the three whose measure is the same quantity
the control acts on. `attack` and `decay` are measured but not derived, for
reasons written down in `pkg/sdk/audio/derive.go`.

A word this earns is a word for the rig's `character`, and `--evidence` writes
the block to paste:

```bash
tonestack measure --corpus resources/music/bass --evidence
```

```yaml
flea:
  - term: clean
    evidence:
      - kind: audio
        measured: { harmonics: 0.12 }
        against: { harmonics: 0.24 }
```

Both sides, because the gap between them is what decides how far the word moves
a control when the preset is built. Half of what everybody else reads is half a
step, not a knob at zero.

Nothing writes it into the rig for you. The terms are an argument, and putting
one in a file is still somebody deciding to believe it.
