# schemas

What a device can do, which gear each of its models emulates, and the presets
all of that was learned from.

How the `.hlx` format works, and how the catalog is generated, is in
[CONTRIBUTING.md](../CONTRIBUTING.md). This file describes what is in this
directory.

| Path                    |               |                                                    |
| ----------------------- | ------------- | -------------------------------------------------- |
| `hx-stomp.catalog.json` | **generated** | Which blocks an HX Stomp has and what each accepts |
| `gear-map.json`         | **generated** | Which real-world gear each Line 6 model emulates   |
| `extract_gear_map.py`   | hand-written  | Writes `gear-map.json`, and lives beside it        |
| `corpus/`               | collected     | ~4,400 real presets                                |

The RigSpec contract is not here. It is embedded in the package that reads it,
at `pkg/sdk/rig/data/rigspec.openapi.yaml`, and
[docs/recipes.md](../../docs/recipes.md#where-the-contract-lives) says why.

## What the catalog and the gear map are for

```text
RigSpec                   gear-map + catalog             .hlx
"Ampeg SVT"          ──►  HD2_AmpSVBeastNrm        ──►   blocks, params, positions
"mid-forward"             bass amps only, real ranges    what the device loads
what a person means       what the device understands    what the file needs
```

The curated rigs in `pkg/sdk/rigs/` are the only gear knowledge that is ours and
publishable. The catalog and the gear map come from a licensed HX Edit
installation, so we do not redistribute them.

## hx-stomp.catalog.json

Generated, never hand-edited. 681 blocks with every parameter's real range,
default and DSP cost.

Two sources, described in [CONTRIBUTING.md](../CONTRIBUTING.md): Line 6's own
model definitions from a local HX Edit installation supply the values; the
corpus supplies which models an HX Stomp actually accepts.

## corpus/

Real presets, used to establish device support and to see what real chains look
like. Reference data only. Nothing this project ships redistributes it.

```text
corpus/
  repos.txt              sources, one per line: <owner/repo><TAB><licence>
  fetch.sh               rebuilds github/ from repos.txt, idempotent
  github/SOURCES.md      per-repo attribution and licences
  customtone/            Line 6's official library, see its SOURCES.md
  setlists_extracted/    presets unpacked from container files
```

```text
4,426  presets
  721  HX Stomp
2,112  of those unpacked from setlist containers
  366  models confirmed in real HX Stomp use
```

Add a source by appending to `repos.txt` and running `fetch.sh`. Record the
licence accurately, including "no license file". That record is why `SOURCES.md`
exists.

**Line 6's CustomTone library is not fetchable.** Downloads sit behind a login
with a registered product serial, and the download control renders as a dead
`login-modal` link for anonymous clients across every device family. See
`corpus/customtone/`.
