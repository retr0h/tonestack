# recipes

Curated knowledge about how a sound is built: which gear a player or style uses,
and how it should behave. Each file is a RigSpec — the same format a preset
reads back as, and the same one that compiles to a device.

This is the only data in the repository that is ours — the device catalog and
the gear map are derived from Line 6's files and cannot be shipped.

- **How to write one:** [docs/recipes.md](../docs/recipes.md)
- **The contract:**
  [schemas/rigspec.openapi.yaml](../schemas/rigspec.openapi.yaml), enforced by
  `pkg/rig`, which refuses to load a rig that violates it

```text
artists/    a named player
genres/     a style, when no particular player is meant
```

One file per subject, named for its `id`.
