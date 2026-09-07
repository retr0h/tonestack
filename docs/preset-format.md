# The preset format

How a Line 6 `.hlx` file is laid out. Line 6 publishes no schema, so all of this
was established by reading real presets. See
[corpus](../resources/schemas/README.md).

`.hlx` files are plain JSON. Line 6 publishes no schema, so everything here was
established by reading real presets.

```json
{"schema":"L6Preset","version":6,
 "data":{"device":2162694,"device_version":57737216,
         "meta":{"name":"…"},
         "tone":{"dsp0":{"block0":{"@model":"HD2_AmpUSDoubleNrm","Drive":0.23}},
                 "dsp1":{…},"controller":{…},"snapshot0":{…}}}}
```

- Two processors, `dsp0` and `dsp1`, each holding `block0`…`blockN` plus
  structural entries (`cab0`, `inputA`, `split`, `join`, `outputA`).
- Block attributes are `@`-prefixed: `@model`, `@position`, `@enabled`, `@path`,
  `@stereo`, `@type`. Everything else is a parameter.
- Eight fixed snapshot slots, `snapshot0`…`snapshot7`.
- `controller` entries carry a parameter's true `@min` and `@max` when someone
  assigned a controller to it.
- **Values are not normalised.** They mix floats, ints and bools, and some are
  in display units, so `Threshold` is `-70.0` dB. A third-party project claims
  everything is a normalised 0.0–1.0 float; the corpus disproves it. This is why
  `ParamValue` is a tagged union rather than `map[string]any`.
- `@model` is usually a symbolic ID (`HD2_AmpUSDoubleNrm`), but 13 models in the
  corpus appear under display names (`Teemah!`, `Parametric EQ`). A parser must
  not assume the `HD2_` prefix.

## Setlist containers

`.hls`, `.hlb` and `.pgs` are wrappers, not presets:

```json
{
  "schema": "L6Setlist",
  "encoding": "Base64",
  "compression": { "type": "zlib", "crc32": 1115417524, "decompressed_size": 1916328 },
  "encoded_data": "<base64>"
}
```

Base64-decode `encoded_data`, zlib-inflate, and the result depends on the
schema:

| Schema           | Extension | Payload                         | Holds         |
| ---------------- | --------- | ------------------------------- | ------------- |
| `L6Setlist`      | `.hls`    | `{meta, presets: […]}`          | 128 slots     |
| `L6PresetBundle` | `.hlb`    | `{setlists: [{meta, presets}]}` | 8 × 128 slots |

Each entry is a preset's `data` object with no `schema` or `version` of its own
by wrapping it in `{"schema":"L6Preset","version":6,"data":<entry>}`. Entries
whose `tone` has no `dsp*` key are empty slots, and a device-written setlist
always holds all 128 of them, most untouched.

`compression.crc32` and `compression.decompressed_size` describe the inflated
payload and must be recomputed on write. A stale checksum is rejected by
whatever loads the file next, and the message it gives blames the wrong thing.

A `.hlb` is what HX Edit writes when it backs a device up, which makes it the
only file stating everything the hardware currently holds. That is what
[device.md](device.md) builds on.

## Nothing may be dropped on a rewrite

A preset read and written back is byte-for-byte the preset that was read, and a
test asserts it over the whole corpus. Making that true meant fixing four ways
it silently was not, and each is a rule for anything added later.

- **A field that can legitimately be empty must not be `omitempty`.** An
  explicit `""` is a different document from a missing field.
- **Whatever is not modelled is kept.** Presets carry `song`, `band`, `author`,
  `tnid`, and an `appVersion` spelled two ways. Metadata holds the name and
  preserves the rest verbatim; a block holds the attributes it models and
  preserves the others.
- **A value keeps the form it arrived in.** `device_version` appears as a
  number, as `"0"` and as `"0.00"`. Parsing and reprinting turns the last into
  the second.
- **A block's key and its `@position` are independent.** `block5` can carry
  `@position: 6`. Deriving either from the other moves blocks around a preset
  nobody asked to change.

## Device identifiers

`data.device` names the hardware; a preset for one device will not load on
another.

| ID                | Device                                  |
| ----------------- | --------------------------------------- |
| 2162689           | Helix Floor / Rack                      |
| 2162690           | Helix, second hardware variant          |
| 2162692           | Helix LT                                |
| 2162693           | HX Effects                              |
| **2162694**       | **HX Stomp**, what this project targets |
| 2162695 / 2162696 | POD Go                                  |
| 2162699           | HX Stomp XL                             |
| 2162944           | Helix Native                            |
