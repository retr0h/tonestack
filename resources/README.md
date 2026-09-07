# resources

The data this project reads and ships, as opposed to the Go that reads it.

| Path       |                                                             |
| ---------- | ----------------------------------------------------------- |
| `schemas/` | the RigSpec contract, the generated catalog, and the corpus |
| `recipes/` | curated rigs, the only content here that is ours to publish |

Both are Go packages, which is not a design choice. `go:embed` cannot reach a
parent directory, so a file that ships inside the binary needs a `.go` file
beside it. `schemas/embed.go` and `recipes/embed.go` are that and nothing else,
one exported symbol each.

They sit here rather than under `pkg/` because people read and edit them far
more often than Go imports them. `rigspec.openapi.yaml` is the one contract
anybody hand-authors, and `recipes/artists/*.yaml` is what contributors write.
Data nobody edits by hand lives in the package that consumes it instead, which
is what `pkg/catalog/data/`, `pkg/corpus/data/`, `pkg/preset/data/` and
`pkg/sdk/wire/data/` are.

## What may be redistributed

`recipes/` is ours. So is `schemas/rigspec.openapi.yaml`.

`schemas/hx-stomp.catalog.json` and `schemas/gear-map.json` are generated from a
licensed HX Edit installation, and `schemas/corpus/` is other people's presets.
Neither travels with a release. See [schemas/README.md](schemas/README.md) for
where each came from.
