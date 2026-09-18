# Workflows

What to actually do, in order, for the things people come here to do.

This is the usage guide. Everything else under [docs/](README.md) explains how
one piece works; this says which pieces to use and when. If you are an agent
being asked for help with any of the tasks below, start here and follow the
links rather than reading everything.

Commands are written as `tonestack ...`, which is how somebody with it installed
runs them. From a checkout, run `go run main.go ...` instead: it builds what is
in front of you, where an installed release does not carry anything added since
it shipped.

Building a preset needs no device and no HX Edit, because the catalog, the
corpus statistics and the rigs are built into the binary. Only the commands that
read or write a device need a Helix plugged in, HX Edit quit, and a build with
USB support, which the released binaries do not have. The
[README](../README.md#install) says how to build one. Every command explains its
own flags with `tonestack <command> --help`, and [commands.md](commands.md)
lists them all.

| I want to…                              | Go to                                                      |
| --------------------------------------- | ---------------------------------------------------------- |
| build a preset for a player             | [Create a rig](workflows/create-a-rig-for-a-player.md)     |
| see what my device holds                | [Read the device](workflows/read-what-a-device-holds.md)   |
| change a rig that already exists        | [Correct a rig](workflows/correct-a-rig-you-have-heard.md) |
| get a preset onto the hardware          | [Load it](workflows/get-it-onto-the-device.md)             |
| switch presets, or move them around     | [Switch and rearrange](workflows/switch-and-rearrange.md)  |
| add knowledge from a video or recording | [Add evidence](workflows/add-evidence-from-a-recording.md) |
| measure records a player actually made  | [Measure a sound](workflows/measure-a-players-sound.md)    |
| fetch more records for a player         | [Add records](workflows/add-records-to-a-corpus.md)        |
| know what the device can do at all      | [Ask the catalog](workflows/ask-what-is-possible.md)       |

## Create a rig for a player

The whole path, from a name to a file that loads: check the gear exists, see
what else belongs in the chain, write it, build it.

Read
[workflows/create-a-rig-for-a-player.md](workflows/create-a-rig-for-a-player.md).

## Read what a device holds

List, show and export the slots on an attached Helix, or on a backup HX Edit
wrote. Read only, and it never selects or writes anything.

Read
[workflows/read-what-a-device-holds.md](workflows/read-what-a-device-holds.md).

## Move a rig between formats

Export a slot as a rig and compile it back. A preset read into a rig and built
again is the preset it came from, asserted over every HX Stomp preset in the
corpus.

Read
[workflows/move-a-rig-between-formats.md](workflows/move-a-rig-between-formats.md).

## Get it onto the device

Write a preset into a slot, and what tonestack saves before it overwrites.

Read [workflows/get-it-onto-the-device.md](workflows/get-it-onto-the-device.md).

## Switch and rearrange

Select a preset on the pedal, and copy or swap slots around.

Read [workflows/switch-and-rearrange.md](workflows/switch-and-rearrange.md).

## Correct a rig you have heard

Play it, say what is wrong, rebuild. The rig keeps each round so the next
session starts from what worked.

Read
[workflows/correct-a-rig-you-have-heard.md](workflows/correct-a-rig-you-have-heard.md).

## Add evidence from a recording

Record where a claim came from: an interview, a video timestamp, a forum thread,
a measurement.

Read
[workflows/add-evidence-from-a-recording.md](workflows/add-evidence-from-a-recording.md).

## Measure a player's sound

Separate the bass out of records somebody actually made, measure them together,
and write the figures into a rig as evidence.

Read
[workflows/measure-a-players-sound.md](workflows/measure-a-players-sound.md).

## Add records to a corpus

Download the records a measurement is short of, under the names the manifest
gives them, and check each is the recording the player is on.

Read
[workflows/add-records-to-a-corpus.md](workflows/add-records-to-a-corpus.md).

## Ask what is possible

What the device can do, and what people actually do with it. Two different
questions, and neither substitutes for the other.

Read [workflows/ask-what-is-possible.md](workflows/ask-what-is-possible.md).

## Use it from an agent

`tonestack mcp start` gives an agent the same operations as tools, with typed
results instead of text to parse.

Read [workflows/use-it-from-an-agent.md](workflows/use-it-from-an-agent.md).

## For agents

What an agent should read first, and the claims it must not overstate.

Read [workflows/for-agents.md](workflows/for-agents.md).
