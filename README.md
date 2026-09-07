[![release](https://img.shields.io/github/release/retr0h/tonestack.svg?style=for-the-badge)](https://github.com/retr0h/tonestack/releases/latest)
[![codecov](https://img.shields.io/codecov/c/github/retr0h/tonestack?style=for-the-badge)](https://codecov.io/gh/retr0h/tonestack)
[![go report card](https://goreportcard.com/badge/github.com/retr0h/tonestack?style=for-the-badge)](https://goreportcard.com/report/github.com/retr0h/tonestack)
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

Installs to `~/.local/bin` or `/usr/local/bin` — SHA256 checksums verified.
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

**See what your Helix holds.** Quit HX Edit first — it holds the device open.

```bash
tonestack presets list
tonestack presets list --file device.hlb    # or a backup, instead of hardware
```

**Make a preset.** Gear is named the way you say it — "Ampeg SVT", never a model
identifier.

```bash
tonestack catalog list --search ampeg --subcategory bass    # is it modelled?

tonestack recipes new \
  --id mike-dirnt --name "Mike Dirnt" --band "Green Day" \
  --instrument bass --amp "Ampeg SVT" --cab "Ampeg 8x10"

tonestack presets make --id mike-dirnt --out mike.hlx
```

Then `HX Edit → Import` and play it.

**Pull a preset back out**, as a rig you can read and edit:

```bash
tonestack presets export  --file device.hlb --slot 3 --out lead.yaml
tonestack presets compile --rig lead.yaml --out lead.hlx
```

**Iterate with Claude.** The scaffold has the gear; it does not know how the rig
should sound, and nothing here can hear. That part is a conversation:

> Fill in `character` and `technique` for `recipes/artists/mike-dirnt.yaml`.

> Too clunky. Loosen the low end and put the drive back.

Claude edits the rig, rebuilds it, and records what you asked for and what you
thought of it. That log is the only record of a human ear in the system —
[docs/workflows.md](docs/workflows.md) is the full loop.

## Documentation

- [`docs/workflows.md`](docs/workflows.md) — **start here**: what to do, in
  order, for building a rig, reading a device, and correcting a preset.
- [`docs/recipes.md`](docs/recipes.md) — every RigSpec field, and how to write
  one by hand.
- [`docs/knowledge.md`](docs/knowledge.md) — where the gear knowledge comes
  from, and how far each source can be trusted.
- [`docs/catalog.md`](docs/catalog.md) — what the device can do, and how that is
  extracted from HX Edit.
- [`docs/protocol.md`](docs/protocol.md) — the USB protocol, reverse engineered.
- [`schemas/`](schemas/) — the RigSpec contract, the generated device catalog,
  and the preset corpus.
- [`recipes/`](recipes/) — curated rigs that ship in the binary.
- [Package documentation](https://pkg.go.dev/github.com/retr0h/tonestack) on
  pkg.go.dev.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for prerequisites, setup, conventions and
the pull request workflow.

## License

The MIT License, see [LICENSE](LICENSE).
