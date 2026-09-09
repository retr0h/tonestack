[![release](https://img.shields.io/github/release/retr0h/tonestack.svg?style=for-the-badge)](https://github.com/retr0h/tonestack/releases/latest)
[![codecov](https://img.shields.io/codecov/c/github/retr0h/tonestack?style=for-the-badge)](https://codecov.io/gh/retr0h/tonestack)
[![license](https://img.shields.io/badge/license-MIT-brightgreen.svg?style=for-the-badge)](LICENSE)
[![build](https://img.shields.io/github/actions/workflow/status/retr0h/tonestack/go.yml?style=for-the-badge)](https://github.com/retr0h/tonestack/actions/workflows/go.yml)
[![powered by](https://img.shields.io/badge/powered%20by-goreleaser-green.svg?style=for-the-badge)](https://github.com/goreleaser)
[![conventional commits](https://img.shields.io/badge/Conventional%20Commits-1.0.0-yellow.svg?style=for-the-badge)](https://conventionalcommits.org)
[![built with just](https://img.shields.io/badge/Built_with-Just-black?style=for-the-badge&logo=just&logoColor=white)](https://just.systems)
![github commit activity](https://img.shields.io/github/commit-activity/m/retr0h/tonestack?style=for-the-badge)
[![go reference](https://img.shields.io/badge/go-reference-00ADD8?style=for-the-badge&logo=go&logoColor=white)](https://pkg.go.dev/github.com/retr0h/tonestack)

# tonestack

Describe a guitar or bass sound, get a Line 6 Helix preset.

## Install

```bash
curl -fsSL https://github.com/retr0h/tonestack/raw/main/install.sh | bash
```

Installs to `~/.local/bin` or `/usr/local/bin`, verifying SHA256 checksums.
Override with `TONESTACK_INSTALL_DIR=/some/path`, or pin a version with
`TONESTACK_VERSION=1.1.1`.

<details>
<summary>Other ways</summary>

```bash
go install github.com/retr0h/tonestack@latest
```

Released binaries are built with `CGO_ENABLED=0`, so they describe, validate and
write presets but cannot reach a device over USB. For that, build with cgo and
libusb present:

```bash
brew install libusb          # or: apt-get install libusb-1.0-0-dev
go build .
```

</details>

## Usage

Plug in the Helix and quit HX Edit. It holds the device open, and nothing else
can talk to the Helix while it runs.

**Read the device.** All three talk to the hardware over USB:

```bash
tonestack presets list                              # every slot
tonestack presets show   --slot 31A                 # one slot, as a rig
tonestack presets export --slot 31A --out lead.yaml # the same, to a file
```

Slots are addressed the way the pedal labels them, `01A` through `42C`.

**Build a preset.** Name gear the way you say it. "Ampeg SVT", never a model
identifier:

```bash
tonestack catalog list --search ampeg --subcategory bass    # is it modelled?

tonestack recipes new \
  --id mike-dirnt --name "Mike Dirnt" --band "Green Day" \
  --instrument bass --amp "Ampeg SVT" --cab "Ampeg 8x10"

tonestack presets make --id mike-dirnt --out mike.hlx
```

Then `HX Edit → Import`, and play it.

**Switch presets from here.** This loads a preset the way a footswitch does, and
writes nothing:

```bash
tonestack presets select --slot 27B
```

**Put it on the device.** The destination is overwritten and a device has no
undo:

```bash
tonestack presets import --preset mike.hlx --slot 07A
tonestack presets copy   --from 01A --to 02A
tonestack presets swap   --from 01A --to 02A
```

Every command also takes `--file` for working from an HX Edit backup with no
device attached.

**Or just ask.** An agent runs those commands for you. The part worth doing
yourself is listening:

> What's on my Helix?

> Build me a Mike Dirnt bass tone.

> Too clunky. Loosen the low end and put the drive back.

Claude edits the rig, rebuilds it, and writes down what you asked for and what
you thought of the result. That log is the only record of a human ear in the
system. [docs/workflows.md](docs/workflows.md) has the full loop.

## Documentation

Start with [`docs/workflows.md`](docs/workflows.md). It says what to do, in
order, for building a rig, reading a device, and correcting a preset.

- [`docs/recipes.md`](docs/recipes.md) covers how to write a rig by hand.
- [`docs/rigspec.md`](docs/rigspec.md) is every field of the format, generated
  from the contract: what each one holds, and what it may say.
- [`docs/knowledge.md`](docs/knowledge.md) says where the gear knowledge comes
  from and how far each source can be trusted.
- [`docs/catalog.md`](docs/catalog.md) covers what the device can do, and how
  that gets extracted from HX Edit.
- [`docs/protocol.md`](docs/protocol.md) documents the USB protocol.
- [`resources/schemas/`](resources/schemas/) holds the RigSpec contract, the
  generated device catalog, and the preset corpus.
- [`resources/recipes/`](resources/recipes/) holds the curated rigs that ship in
  the binary.
- [Package documentation](https://pkg.go.dev/github.com/retr0h/tonestack) is on
  pkg.go.dev.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for prerequisites, setup, conventions and
the pull request workflow.

## License

The MIT License, see [LICENSE](LICENSE).
