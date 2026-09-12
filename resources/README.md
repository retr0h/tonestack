# resources

The data this project reads and ships, as opposed to the Go that reads it.

| Path       |                                                                   |
| ---------- | ----------------------------------------------------------------- |
| `schemas/` | the generated catalog and gear map, and the corpus they come from |

Nothing here is embedded in the binary. What ships lives beside the package that
reads it: `pkg/sdk/catalog/data/`, `pkg/sdk/corpus/data/`,
`pkg/sdk/preset/data/`, `pkg/sdk/rig/data/`, `pkg/sdk/internal/compile/data/`
and `pkg/sdk/internal/wire/data/`. `go:embed` cannot reach a parent directory,
and a package that needs a file from elsewhere is a package nobody can move.

This tree is the working material those files are built from. The RigSpec
contract, the one format anybody hand-authors, is
`pkg/sdk/rig/data/rigspec.openapi.yaml`. The curated rigs are in
`pkg/sdk/rigs/`.

## What may be redistributed

The curated rigs in `pkg/sdk/rigs/` are ours.

`schemas/hx-stomp.catalog.json` and `schemas/gear-map.json` are generated from a
licensed HX Edit installation, and `schemas/corpus/` is other people's presets.
Neither travels with a release. See [schemas/README.md](schemas/README.md) for
where each came from.
