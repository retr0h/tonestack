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
go install github.com/retr0h/tonestack@latest
```

## Usage

```bash
tonestack
```

Prints the command name. Nothing is wired to a command yet — the packages that
describe a signal chain, validate it, and find an attached device are built and
tested, but none of them is reachable from the CLI.

Planned:

```bash
tonestack devices                      # list attached Helix hardware
tonestack presets                      # list what is in each slot
tonestack show 3                       # show the chain in slot 3
tonestack apply 3 rig.hlx              # write a preset to a slot
tonestack make "a Mike Dirnt sound"    # generate a preset from a description
```

## Documentation

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
