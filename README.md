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

```bash
tonestack recipes list                       # what gear knowledge exists
tonestack recipes show --id mike-dirnt       # one recipe in full
tonestack catalog list --subcategory bass    # what the device can do
tonestack catalog show --model HD2_AmpSVBeastNrm
tonestack devices list                       # attached hardware
tonestack corpus show --instrument bass      # what real chains contain
tonestack presets make --id mike-dirnt --out mike.hlx
```

Building a preset reports every decision, including the ones nobody asked for. A
recipe names an amp; a rig is several blocks, and the rest come from what the
corpus shows chains of that kind almost always hold:

```
  Mike Dirnt

  ●  0.0  LA Studio Comp    Teletronix® LA-2A®          5.8
  ●  0.1  Ampeg SVT Brt     Ampeg SVT (bright channel)  26.6
  ●  0.2  8x10 Ampeg SVT-E                              7.2

  dsp0  █████████░░░░░░░░░░░░░░░  39.6%

  added LA Studio Comp — almost every chain has one (88% of chains)

  [ok] wrote mike.hlx
```

### What the device already holds

Point the same commands at a backup HX Edit wrote — a `.hls` setlist or a `.hlb`
of the whole device — and a slot reads back as the same kind of chain:

```bash
tonestack presets list   --file device.hlb                 # every slot
tonestack presets show   --file device.hlb --slot 3        # one chain
tonestack presets export --file device.hlb --slot 3 --out lead.hlx
tonestack presets import --file device.hlb --preset mike.hlx --slot 7 --out edited.hlb
tonestack presets copy   --file device.hlb --from 1 --to 2 --out edited.hlb
tonestack presets swap   --file device.hlb --from 1 --to 2 --out edited.hlb
```

```
  Songs  128 slots · 115 in use

  SLOT  NAME              CHAIN
  01A   Claptone          drive → mod → amp → cab → delay → utility → mod → reverb
  01B   Divider there2    drive → mod → amp → cab → utility → mod → reverb
```

Restore the edited backup with HX Edit. Presets do not travel over USB —
[docs/device.md](docs/device.md) explains why.

## Documentation

- [`docs/`](docs/) — how it all works. Start with
  [knowledge.md](docs/knowledge.md) for how a request becomes a signal chain, or
  [recipes.md](docs/recipes.md) to write one yourself.
- [Package documentation](https://pkg.go.dev/github.com/retr0h/tonestack) on
  pkg.go.dev.
- [`docs/`](docs/) — how it works: turning a request into a chain, the device
  catalog, the preset format.
- [`schemas/`](schemas/) — the RigSpec and Recipe contracts, the generated
  device catalog, the preset corpus.
- [`recipes/`](recipes/) — curated knowledge about players and styles.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for prerequisites, setup, conventions and
the pull request workflow.

## License

The MIT License, see [LICENSE](LICENSE).
