# A grammar for every field

**Status:** proposed\
**Depends on:** checking a rig as it was written, not as it decoded (#37)

## The problem

RigSpec has 68 fields. Twenty-three of them accept any string at all, and
nothing anywhere says what belongs in one.

The two that prompted this are in the example everybody copies:

```yaml
character:
  - mid-forward, not scooped
  - minimal drive, grit only on hard attack

technique: pick, near the bridge
```

`character` requires a non-blank string. `technique` requires nothing. The
schema descriptions carry the intent, "describe the result, not the control",
and no code enforces it. Neither field is read by anything: `resolve` and `lift`
ignore both, and `recipes show` prints them. They exist to be read by a person
and by a model, which is exactly the audience a shape check does not serve.

Three measurements, taken rather than assumed:

| question                                    | answer                    |
| ------------------------------------------- | ------------------------- |
| fields with no enum, pattern or format      | 23 of 68                  |
| fields never mentioned in `docs/recipes.md` | 13, including `technique` |
| fields no example exercises                 | 33                        |

The third number is the one that matters. Half the format has never been written
down in a file anybody can copy, so the only description of those fields is
prose in a schema most people never open.

## What a grammar means here

Not an enum on every field. Every field belongs in one of four buckets, and the
job is that none is left in none of them.

| bucket        | enforced by                          | already used for                                            |
| ------------- | ------------------------------------ | ----------------------------------------------------------- |
| **closed**    | `enum` in the contract               | `role`, `instrument`, `kind`, `confidence`, `evidence.kind` |
| **looked up** | checked against data at load         | gear names, today, through the catalog                      |
| **shaped**    | `pattern` or `format`                | `id`                                                        |
| **open**      | declared as prose, nothing parses it | nothing, currently. Some fields are prose and never say so. |

RigSpec does the first bucket well and the other three not at all. Assigning
every unconstrained field:

### Looked up

The vocabulary exists already, in this repository, and is never consulted.

| field                  | checked against                                         |
| ---------------------- | ------------------------------------------------------- |
| `footswitch.led`       | the colour list `catalogen` reads out of HX Edit        |
| `controller.parameter` | the parameters the catalog gives that model             |
| `footswitch.gear`      | the gear map, the same one a chain entry's `gear` uses  |
| `target.device`        | the devices the catalog names                           |
| `extends`              | a rig identifier that resolves                          |
| `character`            | a term list shipped beside the catalog. New, see below. |

`led: violet` is checked against nothing today while `catalogen.readLEDColours`
sits three packages away holding the answer.

### Shaped

| field                                              | shape                                                      |
| -------------------------------------------------- | ---------------------------------------------------------- |
| `evidence.url`, `requirement.url`                  | `format: uri`                                              |
| `evidence.at`, `mutation.at`                       | a timestamp, `1:42`, or a range                            |
| `change.path`                                      | a path into a rig, `chain[1].settings.drive`               |
| `subject.era`                                      | a year, a range, or a record title                         |
| `target.catalog`                                   | a name and a version, `HX Edit 3.82`                       |
| `snapshot.name`, `footswitch.label`, `device.name` | what the hardware can store: a length, and a character set |

The last row is worth its own sentence. A device truncates a label it cannot
hold, so a rig carrying a longer one describes a preset the device will not
produce, and the round trip stops being a round trip.

### Open

`evidence.note`, `evidence.caveat`, `mutation.reason`, `mutation.verdict`,
`subject.name`, `subject.band`. Commentary and proper nouns. These stay free and
the contract says so, which is the part missing today: a reader cannot currently
tell `note` from `led`, and one of them has an answer.

`mutation.verdict` is a candidate for closing later, once there are enough
mutations to see what people write in it. Guessing the vocabulary before the
data exists is how the first two got this way.

## The decisions

### 1. `technique` becomes structured

A sentence with a grammar is a sentence somebody has to parse. Three orthogonal
things are being said, so say three things:

```yaml
technique:
  attack: pick        # pick | fingers | slap | thumb | hybrid
  position: bridge    # bridge | middle | neck
  muting: palm        # none | palm
```

Three enums, nothing free. It prints back as "pick, near the bridge, palm muted"
wherever a sentence reads better, which is a rendering decision rather than a
storage one.

`target.technique` takes the same type. It exists so a rig tuned from a picked
recording can be compensated when the person playing it uses fingers, and that
comparison needs both sides to be the same kind of value. Two sentences cannot
be compared. Two `attack` fields can.

### 2. `character` becomes a shipped term list

Terms live in `resources/schemas/character-terms.json`, beside `gear-map.json`,
and are checked at load. An unknown term is refused the way unknown gear already
is, with the near misses named:

```console
$ tonestack presets make --id mike-dirnt
[err] no such character term "chimey"
      did you mean: glassy, bright?
```

Not an enum in the contract. The list will churn for months, every term needs a
sentence of definition that an OpenAPI enum has nowhere to put, and this project
already has the pattern for exactly this: a data file, a fuzzy match, a
suggestion. Adding a term becomes a data change with a definition beside it, not
a schema edit and a regeneration.

Seeded from what the two rigs in the tree already say, plus the opposites that
make each an axis rather than a word:

```json
{
  "mids":    ["mid-forward", "scooped"],
  "lowEnd":  ["tight-low-end", "loose-low-end"],
  "decay":   ["short-decay", "long-decay"],
  "drive":   ["clean", "minimal-drive", "grit-on-attack", "saturated"],
  "attack":  ["audible-pick-attack", "soft-attack"],
  "highs":   ["dark", "bright", "glassy"]
}
```

Each term carries a definition in the file. The axis grouping is what lets the
compiler eventually act on a term, and what stops a rig saying `mid-forward` and
`scooped` in the same breath.

### 3. The reference is generated

A hand-written grammar document drifts, and `docs/recipes.md` proves it: 13
fields it never mentions, including the one this design started from.

`docs/rigspec.md` is generated from `rigspec.openapi.yaml` and
`character-terms.json` by a generator that lives beside them and runs in
`just generate`. One row per field: which bucket, the allowed values in full,
and the description already written in the contract. Adding a term to the
vocabulary updates the page. Nobody types it, so it cannot lie.

`docs/recipes.md` keeps what it is good at, which is how to write a rig and what
a good one looks like, and stops trying to be a field reference.

### 4. The example is exercised

A test walks the contract's fields and asserts each appears in at least one file
under `examples/rigspec/`. A field added without an example fails the build.

Thirty-three fields do not appear today. Some belong in a second example rather
than in Mike Dirnt: a rig read off a device carries footswitches, snapshots and
controllers, and that is a different document with a different point to make.
Where a field genuinely suits no example, the exemption is declared in the test
with a reason. The number stops being an accident.

### 5. Continuous integration checks what is generated

`just generate` runs locally, inside `just ready`. Nothing in continuous
integration regenerates and compares, so a generated file drifts the moment
somebody edits a source and does not run it.

Without this, the generated reference inherits the problem it was built to
solve. Regenerate in continuous integration and fail on a dirty tree.

## Migration

Two files carry `technique`, and one carries `character`:

- `resources/recipes/artists/mike-dirnt.yaml`
- `examples/rigspec/mike-dirnt.yaml`

Four `technique` values and four `character` phrases, all in this repository.
Nothing outside it consumes the format yet.

So the old spelling is refused rather than deprecated. No compatibility shim, no
accepting both for a release, no schema version bump. A format with two files
and no external consumers should break cleanly while that is still free.

The four phrases become terms:

| was                                       | becomes                           |
| ----------------------------------------- | --------------------------------- |
| `mid-forward, not scooped`                | `mid-forward`                     |
| `minimal drive, grit only on hard attack` | `minimal-drive`, `grit-on-attack` |
| `tight low end, short decay`              | `tight-low-end`, `short-decay`    |
| `pick attack audible`                     | `audible-pick-attack`             |

"not scooped" disappears, because an axis makes it redundant: a rig saying
`mid-forward` has already said it.

## Where the checks live

Shape checks belong in the contract and run in `rig.Load`, which has the
document.

Look-ups cannot. `rig.Validate` takes a rig and nothing else, and adding a
catalog to it would put a device dependency in the package that defines the
device-independent format. That is the wrong direction, and it is the mistake
[2026-09-06-rigspec-as-the-one-model-design.md](2026-09-06-rigspec-as-the-one-model-design.md)
was written to undo.

So look-ups run where gear names are already resolved, in `internal/resolve`,
and report the same way. A rig with an unknown `led` is a valid rig that this
catalog cannot build, which is exactly what an unknown amplifier already is.

## What this does not do

It does not make `character` affect the preset. Terms are checked, and nothing
compiles them into a chain. Acting on `mid-forward` means deciding what it does
to an EQ block, and that is a question about tone rather than about format.

It does not close `mutation.verdict`, for want of data.

It does not touch the catalog, the corpus, or anything under `pkg/sdk`.

## Order of work

1. Shapes and look-ups for the fields that already have a vocabulary. No format
   change, so no migration.
2. `technique` becomes structured. Two files move.
3. `character-terms.json`, the check, and the suggestion.
4. The generated reference, and the continuous integration check that keeps it
   honest.
5. The example coverage test, and a second example carrying what a device
   writes.

Each step lands on its own. The first is worth doing even if the rest is
rejected.

## Corrections found while implementing

Added rather than rewritten, so the record shows what the design got wrong.

**`format` is not enforced.** kin-openapi treats a string format as an
annotation unless the format is registered globally, and registering one from a
library mutates a table shared with everything else in the binary. So the shaped
fields use `pattern`, which is enforced. Measured: a rig carrying
`url: "not a url at all"` under `format: uri` compiled without complaint.

**`mutation.at` is a date, not a place in a recording.** It records when a
correction was made, "so a run of corrections can be read in order". Only
`evidence.at` is a timestamp into a source. The two were grouped together in the
table above and take different shapes: `2026-09-06` against `1:42` or
`1:20-1:45`.

**`subject.era` has no shape.** It holds "1994" or "American Idiot", a year or
the record a rig belongs to, and a pattern admitting both admits everything. It
moves to open.

**`target.catalog` moves to open as well.** It is written by the catalog
generator rather than by hand, so a pattern would check this project's own
output against itself.

**Label lengths are deferred.** A device truncates a label it cannot hold, and
the limit is not written down anywhere here. Guessing a `maxLength` would refuse
presets the hardware accepts. It needs measuring against a device first.

## What step 2 did

`technique` is three enums now, in both places it appears: the rig's own, and
the one under `target` saying how the person using it plays.

Five files carried the old spelling rather than the two this design counted. The
other three are test fixtures, which is a reminder that a format change costs
whatever exercises the format, not whatever ships it.

**A `$ref` cannot carry a description in OpenAPI 3.0.** Wrapping it in `allOf`
is the usual way round that, and it costs the error message: a rig with
`attack: plectrum` was refused with "technique doesn't match all schemas from
allOf", which names neither the field nor the value. With a plain `$ref` it
reads
`technique.attack value is not one of the allowed values ["pick", "fingers", "slap", "thumb", "hybrid"]`.
The wording that sat beside the `$ref` moved into the component, which is the
only place 3.0 will keep it.

**`muting: none` earns its place.** The first draft said to leave it out when
nothing damps the string, which makes `none` a second way to say what omission
already says. It is worth stating when the notes ringing on is part of the sound
rather than a decision nobody made, and the example says so on the one rig in
the tree.

**The sentence is still there.** `recipes show` prints "pick, near the bridge,
palm muted" from the three fields, so what a person reads did not change and the
test that asserted that line still passes untouched. `none` prints as nothing,
because an unmuted note is what every note sounds like unless something damps
it.

## A second axis: whether a field is a claim

The four buckets say what a field may hold. They say nothing about how it came
to hold it, and that turns out to be the more valuable question for exactly the
fields this design started from.

**A field that asserts something about the world must be able to carry evidence.
A field that describes what a preset holds must not, because citing it would be
citing ourselves.**

Applied to the tree as it stood, two claims could be cited and two could not:

| field              | could cite | can now |
| ------------------ | ---------- | ------- |
| `chain[].gear`     | yes        | yes     |
| the rig as a whole | yes        | yes     |
| `technique`        | no         | yes     |
| `character[]`      | no         | yes     |

The two that could not are the two nothing can measure. A corpus can be asked
what parameter a model usually carries; nothing anywhere says how somebody
picks, or that a tone is mid-forward. Those claims are asserted or somebody
listened, and the gap between those is the whole reason the format records
provenance at all. `Evidence` says as much in its own description, that it is
attached per claim rather than per document, and then the two loudest claims in
the file had only the per-document bucket to sit in.

The other side of the rule is what it excludes. `snapshots`, `footswitches`,
`controllers` and the `device` block are read off hardware or chosen by whoever
wrote the rig. They are not claims about a player, and hanging evidence on them
would be ceremony.

`subject.era` is the one left undecided. "Dookie through American Idiot" is a
claim, and making it citable means turning a string into an object for a field
nothing reads yet.

### What it cost

`character` had to stop being a list of strings, which is the shape change step
3 was going to make anyway for the term list. Doing it here means the format
breaks once rather than twice, and step 3 is left with only the vocabulary: a
term is free text until `character-terms.json` exists, and the contract says so
where a reader will see it.
