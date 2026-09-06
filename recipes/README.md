# recipes

Curated knowledge about how a sound is built: which gear a player or style uses,
and how it should behave.

This is the only data in the repository that is ours — the device catalog and
the gear map are derived from Line 6's files and cannot be shipped.

- **How to write one:** [docs/recipes.md](../docs/recipes.md)
- **The contract:** [schemas/recipe.schema.json](../schemas/recipe.schema.json),
  enforced by `just recipes-check`

```text
artists/    a named player
genres/     a style, when no particular player is meant
```

One file per subject, named for its `id`.
