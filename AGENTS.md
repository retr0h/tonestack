# AGENTS.md

Test: `just test` | Before committing: `just ready`

Read [CONTRIBUTING.md](CONTRIBUTING.md) first. It covers prerequisites, setup,
package structure, code standards and testing. All of it applies to agents
exactly as it applies to people. This file carries only what is specific to
agents.

## Running tools

Invoke tools through `mise`, not from your path:

```bash
mise exec -- just test
```

`mise` is active in a person's shell and supplies the versions `.mise.toml`
declares. An agent's shell has no activation, so a bare `just` resolves to
whatever is installed globally, usually an older version.

The symptom is a check that fails here and passes in continuous integration, on
a file nobody edited. When that happens, establish which version ran before
treating the failure as real.

## CONTRIBUTING is not optional reading

[CONTRIBUTING.md](CONTRIBUTING.md) is the source of truth for layout,
conventions, testing and the licence header every file carries. It applies to
agents exactly as it applies to people, and none of it is repeated here.

Two of its rules are easy to skip and worth naming: run `just ready` before
committing, and put every markdown change through the unslop skill first. See
[Prose](CONTRIBUTING.md#prose).

## Finding your way around the domain

[docs/](docs/) covers what the code is *for*, which is not derivable from the
code. Read the one that matches the task rather than all of them:

| Task                                                                                                  | Read                                                                                                                    |
| ----------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------- |
| **Somebody asks for help doing something**: build a rig for a player, read a device, correct a preset | [docs/workflows.md](docs/workflows.md), the step-by-step, linking onward to whichever reference it needs                |
| Understanding why any of this is shaped as it is                                                      | [docs/knowledge.md](docs/knowledge.md), how a request becomes a signal chain and which of the four problems is unsolved |
| Writing or changing a rig                                                                             | [docs/recipes.md](docs/recipes.md), with [examples/rigspec/mike-dirnt.yaml](examples/rigspec/mike-dirnt.yaml) beside it |
| **Whether a field may say a thing**: what is allowed, and what is refused                             | [docs/rigspec.md](docs/rigspec.md), generated from the contract and never hand-edited                                   |
| Anything touching models, parameters or DSP cost                                                      | [docs/catalog.md](docs/catalog.md)                                                                                      |
| Reading or writing a `.hlx`                                                                           | [docs/preset-format.md](docs/preset-format.md)                                                                          |
| Reading or editing what a device holds                                                                | [docs/device.md](docs/device.md)                                                                                        |
| Touching USB                                                                                          | [docs/protocol.md](docs/protocol.md), **including the rules that keep a device alive**                                  |
| Changing the shape of the system                                                                      | [docs/superpowers/specs/](docs/superpowers/specs/), dated design records, superseded rather than rewritten              |

The RigSpec contract is
[`pkg/sdk/rig/data/rigspec.openapi.yaml`](pkg/sdk/rig/data/rigspec.openapi.yaml),
embedded in the package that reads it. It is the only hand-authored format;
everything else is compiled from it. The generated catalog, the gear map and the
corpus are in [resources/schemas/](resources/schemas/), and
[resources/README.md](resources/README.md) says what else is in that tree and
which of it may be redistributed.

## Say which claim you have

"The rig validates against the catalog", "HX Edit imported the file" and "the
hardware loaded it" are three different claims. Only the first is currently
possible in this repository.

Do not report one as another, and do not describe work as verified on evidence
you did not gather. If you did not run it, say you did not run it.

## Commit trailer

When committing via Claude Code, end the message with:

```
🤖 Generated with [Claude Code](https://claude.ai/code)

Co-Authored-By: Claude <noreply@anthropic.com>
```

## Re-running the scaffold

`retemplate-go` brings this project up to a newer template. It reads
`.swamp-template.json`, which records what the last run wrote and the hash of
each file, and decides per file:

- A file **unchanged since generated** gets the current template's version.
- A file **edited here** is left alone, and named in the output.
- A file that is **absent** gets written.

Keep `.swamp-template.json` in the repository. Without it every file looks
edited, and a retemplate can only skip.

### When a file is reported as an orphan

An orphan is a file an earlier template generated, this one no longer generates,
and nobody has edited. An `internal/cli/` left behind when the entry point moved
to `cmd/` is one. It gets reported, never deleted:

```
orphan internal/cli/cli.go: generated by an earlier template, no longer
       part of this one, and unchanged since. Safe to delete.
```

Check nothing imports it, then delete it. Left in place it compiles and passes
the gate while being unreachable, which is how it goes unnoticed.

### When a file is held

Some files only make sense together: `main.go` imports `cmd`, `cmd` imports
`internal/<pkg>`; the library stub's test calls a function its source defines.
If one of them has been edited, the others are held rather than written, and the
output says so:

```
HOLD cmd/root.go: the entrypoint files this project already has came from an
     earlier template.
```

The run cannot merge your edit into the new shape. Either port the edit by hand,
or move those files aside and re-run to get the current set.
