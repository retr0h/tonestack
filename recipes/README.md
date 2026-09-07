# recipes

Curated knowledge about how a sound is built: which gear a player or style uses,
and how it should behave. Each file is a RigSpec, the same format a preset reads
back as and the same one that compiles to a device.

This is the only data in the repository that is ours. The device catalog and the
gear map come from Line 6's files, so we cannot ship them.

[docs/recipes.md](../docs/recipes.md) says how to write one.
[schemas/rigspec.openapi.yaml](../schemas/rigspec.openapi.yaml) is the contract,
and `pkg/rig` refuses to load a rig that violates it.

```text
artists/    a named player
genres/     a style, when no particular player is meant
```

One file per subject, named for its `id`.
