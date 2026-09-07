# schemas

The data contracts, the generated catalog, and the presets everything was
learned from.

How the `.hlx` format works, and how the catalog is generated, is in
[CONTRIBUTING.md](../CONTRIBUTING.md). This file describes what is in this
directory.

| Path                    |               |                                                     |
| ----------------------- | ------------- | --------------------------------------------------- |
| `rigspec.openapi.yaml`  | hand-written  | The RigSpec contract, the only format anybody types |
| `hx-stomp.catalog.json` | **generated** | Which blocks an HX Stomp has and what each accepts  |
| `gear-map.json`         | **generated** | Which real-world gear each Line 6 model emulates    |
| `corpus/`               | collected     | ~4,400 real presets                                 |

## RigSpec is the only contract

```text
RigSpec                   gear-map + catalog             .hlx
"Ampeg SVT"          ──►  HD2_AmpSVBeastNrm        ──►   blocks, params, positions
"mid-forward"             bass amps only, real ranges    what the device loads
what a person means       what the device understands    what the file needs
```

There used to be two contracts, one for what a person writes and one for what
the generator produces. They were the same document at two levels of detail, so
now there is one. A rig is sparse when somebody types it and full once it has
been compiled or lifted from a preset.

`recipes/` is the only data here that is ours and publishable. The catalog and
the gear map come from a licensed HX Edit installation, so we do not
redistribute them.

## rigspec.openapi.yaml

RigSpec is this project's own invention. The Line 6 format has no equivalent. It
stores blocks under `dsp0`/`block0` keys with no abstraction over where a chain
came from. RigSpec exists so every input converges on one validated shape before
anything writes a file.

The schema is the contract in both senses. `pkg/rig/gen` is generated from it by
`oapi-codegen`, and `pkg/rig.Validate` checks a document against the same file
rather than against a second copy of the rules written in Go. Two copies drift:
a constraint added to one becomes a type nothing enforces, or a check nothing
asked for.

### Why RigSpec does not enumerate models

RigSpec is the *shape* of a signal chain and is stable across devices and
firmware. Which models exist belongs to a device at a firmware version, and that
is the catalog's job.

Enumerating models inside RigSpec would tie its version to the firmware, make a
preset using an unlisted model unrepresentable, and produce a schema tens of
thousands of lines long. A per-device schema with the model enum inlined can be
*generated* from the two when strict validation is wanted.

### Generating clients

The schema is the source for anything that needs to speak RigSpec, including
this project's own Go types:

```bash
just generate                                    # regenerates pkg/rig/gen
npx openapi-typescript schemas/rigspec.openapi.yaml -o rigspec.d.ts
```

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
