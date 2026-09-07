# docs

How tonestack works. For how to work *on* it, setup and conventions and testing,
see [CONTRIBUTING.md](../CONTRIBUTING.md). These pages cover the domain.

|                                      |                                                                                 |
| ------------------------------------ | ------------------------------------------------------------------------------- |
| [workflows.md](workflows.md)         | **Start here.** What to do, in order, for the things people come here to do     |
| [knowledge.md](knowledge.md)         | How a request becomes a signal chain, and the four problems that entails.       |
| [recipes.md](recipes.md)             | Writing a rig, the one format anybody authors by hand                           |
| [catalog.md](catalog.md)             | What a device can do, and where that knowledge comes from                       |
| [preset-format.md](preset-format.md) | How a `.hlx` file is laid out                                                   |
| [device.md](device.md)               | Reading and editing what a device holds, and what USB is for                    |
| [protocol.md](protocol.md)           | Talking to a device over USB: framing, calls, and the rules that keep one alive |

Design records live under [superpowers/](superpowers/). They are dated, and
superseded rather than rewritten. The current architecture is
[RigSpec as the one model](superpowers/specs/2026-09-06-rigspec-as-the-one-model-design.md).
One specification; everything else compiles from it.
