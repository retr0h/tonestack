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

## Features

| Feature                                                                     | Description                                                                                                                                                                                                                                                                              |
| --------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [RigSpec](docs/rigspec.md)                                                  | One file describes a tone. It names the gear in the chain the way a player would, says how it should sound and how the player plays it. Every field follows an [OpenAPI contract](pkg/sdk/rig/data/rigspec.openapi.yaml), and tonestack refuses a rig that breaks it                     |
| Shareable rigs                                                              | A rig names "Ampeg SVT", not a Line 6 model ID. Export a slot as a rig, send the file to someone, and `presets compile` builds the preset on their end                                                                                                                                   |
| [Every claim sourced](docs/recipes.md#say-where-each-claim-came-from)       | Each piece of gear and each description records where it came from: an interview, a video timestamp, a forum thread, measured presets. A claim an AI made says so                                                                                                                        |
| [Your agent tunes it](docs/workflows.md#correct-a-rig-you-have-heard)       | Your agent builds a rig for the player you name. Play it, say what is wrong, and it rebuilds. The rig keeps each round: what you asked, what changed, why, and your verdict. The next session starts from what worked                                                                    |
| Talks to your Helix                                                         | Read, write, copy, swap and select presets over USB. tonestack saves a slot to a file before it overwrites it. Device access is one backend per operating system, and only macOS has one so far. Building presets works everywhere, and the HX Stomp is the device it has been tested on |
| [MCP server](docs/workflows.md#use-it-from-an-agent)                        | `tonestack mcp` gives an agent the catalog, the corpus, building and the pedal as tools with typed results. It cannot write to a pedal unless you start it with `--allow-writes`                                                                                                         |
| [Go SDK](CONTRIBUTING.md#what-to-import-if-you-are-using-this-as-a-library) | The CLI is flags over `pkg/sdk`. Import it to build presets, read and write a device, or look up what a device can do from your own Go program                                                                                                                                           |

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

Released binaries reach a Helix over USB on macOS. On Linux they build, validate
and write presets, and the device commands say they are not supported yet.

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

  ●  0.0  Deluxe Comp       Line 6 Original             1.8
  ●  0.1  Ampeg SVT Brt     Ampeg SVT (bright channel)  26.6
  ●  0.2  8x10 Ampeg SVT-E                              7.2

  dsp0  ████████░░░░░░░░░░░░░░░░  35.6%

  added Deluxe Comp — almost every chain has one (88% of chains)

  heard mid-forward — Mid 0.79 to 0.89
  heard grit-on-attack — Drive 0.60 to 0.76
  heard tight-low-end — Sag 0.50 to 0.40
  heard short-decay — nothing acts on this yet
  heard audible-pick-attack — Attack 0.04 to 0.05

  [ok] wrote mike.hlx
```

Read that before you plug anything in. Each block names the real gear it models
and what it costs. `added` is what tonestack put in that the rig did not ask
for, and `heard` is what each word in the rig's description did to a knob.

Have a rig file of your own, one you wrote or one somebody shared? Build it the
same way:

```bash
tonestack presets compile --rig mine.yaml --out mine.hlx
```

### Put it on the pedal

With the Helix plugged in and HX Edit quit, tonestack writes the preset straight
into a slot, and saves what that slot held to a file first:

```bash
tonestack presets import --preset mike.hlx --slot 07A
```

Device access is macOS only for now. Anywhere else, import the file with HX
Edit.

## Next

| To                                                    | Read                                                                  |
| ----------------------------------------------------- | --------------------------------------------------------------------- |
| build a preset for a player who is not in the list    | [Create a rig](docs/workflows.md#create-a-rig-for-a-player)           |
| use a plugged-in Helix                                | [Read the device](docs/workflows.md#read-what-a-device-holds)         |
| change a preset after you have played it              | [Correct a rig](docs/workflows.md#correct-a-rig-you-have-heard)       |
| see every command and flag                            | [docs/commands.md](docs/commands.md), or `tonestack <command> --help` |
| understand how it works                               | [docs/](docs/README.md)                                               |
| work on tonestack, including regenerating the catalog | [CONTRIBUTING.md](CONTRIBUTING.md)                                    |

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
