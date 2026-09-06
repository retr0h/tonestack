# The preset format

How a Line 6 `.hlx` file is laid out. Line 6 publishes no schema, so all of this
was established by reading real presets — see [corpus](../schemas/README.md).

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
  in display units — `Threshold` is `-70.0` dB. A third-party project claims
  everything is a normalised 0.0–1.0 float; the corpus disproves it. This is why
  `ParamValue` is a tagged union rather than `map[string]any`.
- `@model` is usually a symbolic ID (`HD2_AmpUSDoubleNrm`), but 13 models in the
  corpus appear under display names (`Teemah!`, `Parametric EQ`). A parser must
  not assume the `HD2_` prefix.

## Setlist containers

`.hls`, `.hlb` and `.pgs` are wrappers, not presets:

```json
{"schema":"L6Setlist","compression":{"type":"zlib"},"encoded_data":"<base64>"}
```

Base64-decode `encoded_data`, zlib-inflate, and the result is
`{"meta":…,"presets":[…]}`. Each entry is a preset's `data` object with no
`schema` or `version` of its own — wrap it in
`{"schema":"L6Preset","version":6,"data":<entry>}`. Entries whose `tone` has no
`dsp*` key are empty slots.

## Device identifiers

`data.device` names the hardware; a preset for one device will not load on
another.

| ID                | Device                                   |
| ----------------- | ---------------------------------------- |
| 2162689           | Helix Floor / Rack                       |
| 2162690           | Helix, second hardware variant           |
| 2162692           | Helix LT                                 |
| 2162693           | HX Effects                               |
| **2162694**       | **HX Stomp** — what this project targets |
| 2162695 / 2162696 | POD Go                                   |
| 2162699           | HX Stomp XL                              |
| 2162944           | Helix Native                             |
