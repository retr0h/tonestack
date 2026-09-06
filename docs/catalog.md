# The device catalog

What an HX Stomp can do, and where that knowledge comes from.

`schemas/hx-stomp.catalog.json` is generated. Never hand-edit it.

**Ranges, defaults and DSP costs come from HX Edit's own model definitions.**
The application ships Line 6's complete model data as plain JSON in
`/Applications/Line6/HX Edit.app/Contents/Resources/*.models` — 681 models, each
with a `symbolicID`, a `name`, a DSP `load`, and every parameter's `min`, `max`,
`default` and value type. Nothing about ranges or DSP cost needs inferring.

That data is Line 6's and reaches us only because this machine has a licensed HX
Edit installation. It is **not redistributed**: the catalog is generated
locally, and a machine without HX Edit cannot build one. The generator fails
clearly rather than guessing.

**Which models the device accepts comes from the preset corpus** — see
[schemas/README.md](schemas/README.md).

Every catalog parameter carries `prov`: `official` means Line 6 stated the
value; `observed` means it was inferred from presets alone and the bounds are
only what the corpus happened to contain.
