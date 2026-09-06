# schemas

The data contracts, the generated catalog, and the presets everything was
learned from.

How the `.hlx` format works, and how the catalog is generated, is in
[CONTRIBUTING.md](../CONTRIBUTING.md). This file describes what is in this
directory.

| Path                    |               |                                                    |
| ----------------------- | ------------- | -------------------------------------------------- |
| `rigspec.schema.json`   | hand-written  | The RigSpec contract — a signal chain              |
| `recipe.schema.json`    | hand-written  | The Recipe contract — how a sound is built         |
| `hx-stomp.catalog.json` | **generated** | Which blocks an HX Stomp has and what each accepts |
| `gear-map.json`         | **generated** | Which real-world gear each Line 6 model emulates   |
| `corpus/`               | collected     | ~4,400 real presets                                |

## Recipe and RigSpec are the two ends of generation

```text
Recipe                    gear-map + catalog             RigSpec
"Ampeg SVT"          ──►  HD2_AmpSVBeastNrm        ──►   blocks, params, positions
"mid-forward"             bass amps only, real ranges    validated, writable
what a person means       what the device understands    what the file needs
```

A **Recipe** is an input, written by a person, naming real-world gear. A
**RigSpec** is an output, produced by the generator, naming device models.

They are separate contracts because they answer different questions and change
for different reasons: a Recipe changes when someone learns something about a
player, a RigSpec changes when the device does. Both are versioned
independently.

`recipes/` is the only data here that is genuinely ours and publishable. The
catalog and the gear map are derived from a licensed HX Edit installation and
are not redistributed.

## rigspec.schema.json

RigSpec is this project's own invention. The Line 6 format has no equivalent —
it stores blocks under `dsp0`/`block0` keys and has no abstraction over where a
chain came from. RigSpec exists so every input path converges on one validated
shape before anything writes a file.

The schema is the contract; the Go types in `pkg/rig` implement it, and
`pkg/rig/schema_conformance_public_test.go` pins them to it. That test earns its
keep — it caught a nil `Params` map marshalling as `null` on its first run.

### Why RigSpec does not enumerate models

RigSpec is the *shape* of a signal chain and is stable across devices and
firmware. Which models exist belongs to a device at a firmware version, and that
is the catalog's job.

Enumerating models inside RigSpec would tie its version to the firmware, make a
preset using an unlisted model unrepresentable, and produce a schema tens of
thousands of lines long. A per-device schema with the model enum inlined can be
*generated* from the two when strict validation is wanted.

### Generating clients

The schema is the source for anything that needs to speak RigSpec:

```bash
npx json-schema-to-typescript schemas/rigspec.schema.json > rigspec.d.ts
```

Go types are hand-written rather than generated, and this was tested rather than
assumed. `go-jsonschema` renders the `paramValue` union as `interface{}`, which
throws away the one distinction the type exists to preserve. `oapi-codegen` does
better — a real union with typed accessors — but names them positionally
(`AsParamValue0`) and drops the zero-value guard. Neither is worth trading a
tagged union for. Other languages should generate; Go is the exception.

## hx-stomp.catalog.json

Generated, never hand-edited. 681 blocks with every parameter's real range,
default and DSP cost.

Two sources, described in [CONTRIBUTING.md](../CONTRIBUTING.md): Line 6's own
model definitions from a local HX Edit installation supply the values; the
corpus supplies which models an HX Stomp actually accepts.

## corpus/

Real presets, used to establish device support and to see what real chains look
like. Reference data — nothing shipped by this project redistributes it.

```text
corpus/
  repos.txt              sources, one per line: <owner/repo><TAB><licence>
  fetch.sh               rebuilds github/ from repos.txt, idempotent
  github/SOURCES.md      per-repo attribution and licences
  customtone/            Line 6's official library — see its SOURCES.md
  setlists_extracted/    presets unpacked from container files
```

```text
4,426  presets
  721  HX Stomp
2,112  of those unpacked from setlist containers
  366  models confirmed in real HX Stomp use
```

Add a source by appending to `repos.txt` and running `fetch.sh`. Record the
licence accurately, including "no license file" — that record is why
`SOURCES.md` exists.

**Line 6's CustomTone library is not fetchable.** Downloads are gated behind a
login with a registered product serial; the download control renders as a dead
`login-modal` link for anonymous clients across every device family. See
`corpus/customtone/`.
