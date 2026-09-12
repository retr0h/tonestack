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

## Quickstart

The device catalog and a set of player rigs are built into the binary. Building
a preset needs no HX Edit and no Helix.

```bash
tonestack recipes list                                  # the rigs that ship
tonestack presets make --id mike-dirnt --out mike.hlx
```

```console
  Mike Dirnt

  ●  0.0  LA Studio Comp    Teletronix® LA-2A®          5.8
  ●  0.1  Ampeg SVT Brt     Ampeg SVT (bright channel)  26.6
  ●  0.2  8x10 Ampeg SVT-E                              7.2

  dsp0  █████████░░░░░░░░░░░░░░░  39.6%

  added LA Studio Comp — almost every chain has one (88% of chains)

  heard mid-forward — Mid 0.79 to 0.89
  heard grit-on-attack — Drive 0.60 to 0.76
  heard tight-low-end — Sag 0.50 to 0.40
  heard short-decay — nothing acts on this yet
  heard audible-pick-attack — the LA Studio Comp has no Attack

  [ok] wrote mike.hlx
```

Read that before you plug anything in. Each block names the real gear it models
and what it costs. `added` is what tonestack put in that the rig did not ask
for, and `heard` is what each word in the rig's description did to a knob.

Then `HX Edit → Import`. Or, with the Helix plugged in and HX Edit quit, put it
straight into a slot. Whatever the slot held is saved to a file first. This one
needs a build that can reach USB, which the released binaries are not; see
*Other ways* above.

```bash
tonestack presets import --preset mike.hlx --slot 07A
```

## Next

| To                                                    | Read                                                            |
| ----------------------------------------------------- | --------------------------------------------------------------- |
| build a preset for a player who is not in the list    | [Create a rig](docs/workflows.md#create-a-rig-for-a-player)     |
| use a plugged-in Helix                                | [Read the device](docs/workflows.md#read-what-a-device-holds)   |
| change a preset after you have played it              | [Correct a rig](docs/workflows.md#correct-a-rig-you-have-heard) |
| see every flag a command takes                        | `tonestack <command> --help`                                    |
| understand how it works                               | [docs/](docs/README.md)                                         |
| work on tonestack, including regenerating the catalog | [CONTRIBUTING.md](CONTRIBUTING.md)                              |

[docs/workflows.md](docs/workflows.md) is the usage guide, for everything past
the quickstart.

**Or ask an agent.** It runs the commands. The part worth doing yourself is
listening:

> Build me a Mike Dirnt bass tone.

> Too clunky. Loosen the low end and put the drive back.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for prerequisites, setup, conventions and
the pull request workflow.

## License

The MIT License, see [LICENSE](LICENSE).
