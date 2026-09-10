# rigs

Curated knowledge about how a sound is built: which gear a player or style uses,
and how it should behave. Each file is a RigSpec, the same format a preset reads
back as and the same one that compiles to a device.

This is the only data in the repository that is ours. The device catalog and the
gear map come from Line 6's files, so we cannot ship them.

Here rather than under `resources/` because it is the first of the things this
project has to know — [docs/knowledge.md](../../../docs/knowledge.md) lists it
above the gear map and the catalog — and every other one of those already
travels inside the library. A rig this project ships is library knowledge in the
same way the catalog is.

[docs/recipes.md](../../../docs/recipes.md) says how to write one.
[../rig/data/rigspec.openapi.yaml](../rig/data/rigspec.openapi.yaml) is the
contract, and `pkg/sdk/rig` refuses to load a rig that violates it.

```text
artists/    a named player
genres/     a style, when no particular player is meant
```

One file per subject, named for its `id`.
