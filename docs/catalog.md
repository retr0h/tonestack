# The device catalog

What a device can do, and where that knowledge comes from.

The catalog is generated. Never hand-edit it. Change the generator, or the data
it reads, and run `just catalog`.

## It ships in the binary

`pkg/sdk/catalog/data/hx-stomp.json.gz` is committed and embedded: 665 blocks,
1.5MB of JSON compressed to about 59KB.

**Nobody needs HX Edit to use this project.** Generating a catalog does; using
one does not, and that distinction is the whole reason the file is committed
rather than built on demand. A `--catalog` flag overrides the built-in one for
anybody who has generated their own.

## Two sources, joined

Neither half is enough alone, and the join is what makes a rig buildable.

### What a model is, from HX Edit's own data

`/Applications/Line6/HX Edit.app/Contents/Resources/*.models` holds 19 files of
681 models between them, as plain JSON. Each carries a `symbolicID`, a `name`, a
DSP `load`, a `cablink` naming the cabinet Line 6 voiced the amp with, and every
parameter's `min`, `max`, `default` and value type.

So ranges and DSP costs are **stated, not inferred**. A real entry, unedited:

```json
{ "symbolicID": "HD2_AmpGermanMahadeva", "name": "German Mahadeva",
  "category": 1, "cablink": "HD2_Cab1x12Lead80", "load": 28.27, "params": [...] }
```

**Which device carries a model comes from the same files.** Each model lists the
devices that support it, and that list is what filters 681 models down to the
665 an HX Stomp has. A model listing no devices at all is taken as universal.

### What a model *is*, from the manual

The `.models` files never say what real gear a model imitates.
`HD2_AmpSVBeastNrm` does not mention Ampeg anywhere, and that is deliberate:
Line 6 rename every model for trademark reasons.

The mapping exists in one place, the "Based On" column of the Pilot's Guide,
which ships in the same folder. `resources/schemas/extract_gear_map.py` reads
it, sitting beside the `resources/schemas/gear-map.json` it writes. It is the
exception to the no-Python rule in [CONTRIBUTING.md](../CONTRIBUTING.md). The
model-name column uses a subset-embedded font no Go PDF library decodes, and it
runs once per Line 6 release rather than on every build.

Joining the two gives **547 of 665 models mapped to gear a person recognises**,
and that is what makes a rig naming "Ampeg SVT" mean anything at all.

## A catalog says which release it came from

```json
{ "device": "HX Stomp", "device_id": 2162694,
  "schema_version": 6, "source": "HX Edit 3.82" }
```

Read from the application bundle, because the `.models` files carry no version
of their own. It matters more now that the catalog ships in the binary: a
catalog is only true of the models one release knew about, and a device on older
firmware may not have all of them. A catalog that cannot name its source
displays as `source unknown` rather than being presented as authoritative.

## Categories are what a block does

The generator maps each `.models` filename onto a category, and a rig uses the
same words for a block's role, so one resolves to the other without translation.

| category                       | from                                               |
| ------------------------------ | -------------------------------------------------- |
| `amp`                          | `amp`, `preamp`                                    |
| `cab`                          | `cab`, `cabmicirs`, `cabmicirswithpan`             |
| `drive`                        | `distortion`                                       |
| `comp`                         | `compressor`                                       |
| `gate`                         | `gate`                                             |
| `eq`, `delay`, `reverb`, `mod` | the file of that name                              |
| `wah`, `pitch`, `filter`       | `wah`, `pitch-synth`, `filter`                     |
| `utility`                      | `volumepan`, `sendreturn`, `io`, `fixed`, `looper` |
| `other`                        | anything else                                      |

`utility` is plumbing: volume, gain, a send, a looper. Nobody chooses one for
how it sounds, so it is excluded from anything measuring what a chain is made
of, while still being nameable in a rig because a real chain contains them.

A noise gate is **not** a compressor. Counting them together overstated how
often players compress, which is exactly the kind of thing the corpus statistics
are supposed to measure honestly.

## Every figure says how far to trust it

Each block and each DSP cost carries a `prov`:

| value      | means                                                                          |
| ---------- | ------------------------------------------------------------------------------ |
| `official` | Line 6 stated it                                                               |
| `observed` | inferred from presets, so the bounds are only what the corpus happened to hold |
| `assumed`  | neither, so a guess                                                            |

This is not decoration. A chain is never filled with a block whose DSP cost is
`assumed`: budgeting a rig on a guess produces one that validation refuses a
moment later, blaming a block nobody asked for.

## Reading it

```bash
tonestack catalog list --search ampeg
tonestack catalog list --category amp --subcategory bass
tonestack catalog show --model HD2_AmpSVBeastNrm
```

What the device can do is a different question from what people do with it. See
[corpus statistics](knowledge.md). Line 6 state a default Treble of 0.68 for an
Ampeg SVT; the median across every measured use is 0.845. Both are facts, and
the catalog only knows the first.

## Regenerating it

```bash
just gear-map    # once per Line 6 release; needs HX Edit and Python
just catalog     # reads the models and the gear map, writes the catalog
```

Model names, parameter ranges and artwork are Line 6's. They are read from a
local installation at generation time and the results are committed here as
device knowledge, in the same way the corpus is collected rather than
redistributed wholesale.
