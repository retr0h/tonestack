# Each instrument its own crowd

**Status:** implemented\
**Scope:** `pkg/sdk/corpus`, `pkg/sdk/internal/corpusgen`,
`pkg/sdk/internal/compile`\
**Builds on:**
[Character moves a knob](2026-09-12-character-moves-a-knob-design.md)

## The problem

Four of the nine shipped rigs describe their attack, and all four ended the same
way:

```text
flea          heard percussive — the LA Studio Comp has no Attack
geddy-lee     heard percussive — the LA Studio Comp has no Attack
les-claypool  heard percussive — the LA Studio Comp has no Attack
mike-dirnt    heard audible-pick-attack — the LA Studio Comp has no Attack
```

None of them asked for a compressor. The build adds one because nearly every
bass chain holds one, and it adds the model the corpus used most. The LA Studio
Comp is an LA-2A emulation, an opto, with no attack control.

## What was actually wrong

"The model the corpus used most" was counted across every preset, and guitar
chains outnumber bass twenty to one. The corpus's grammar already kept guitar
and bass apart, because averaging them describes neither. The model counts did
not.

Counted per instrument, over the 159 bass and 3,468 guitar chains in the corpus,
the compressors come out in a different order:

| model          | bass chains | guitar chains | Attack |
| -------------- | ----------- | ------------- | ------ |
| Deluxe Comp    | 72          | 604           | yes    |
| LA Studio Comp | 61          | 689           | no     |
| Red Squeeze    | 4           | 379           | no     |

Bass players reach for the Deluxe Comp. The LA Studio Comp was the guitar
players' choice, handed to bass rigs because guitar outvoted them.

## Decision

The corpus counts, for each instrument, how many of its chains hold each model.
The build adds the model players of the rig's instrument hold most often. A
model no chain for that instrument held is never added, however popular it is
elsewhere.

That is the rule `fill` always claimed to follow, counted in the right crowd. It
fixes the four rigs as a side effect: the Deluxe Comp has an Attack control, so
their attack words find one.

Statistics measured before the count existed carry no per-instrument figures,
and fall back to the total across every preset, which is what they have.

## Rejected: letting the word pick the model

The first proposal was that when a word needs a control the favourite lacks, the
build adds the most-used model that has it. Two reasons against, once the counts
were split:

- It is not needed. No shipped rig needs a word to steer the choice once the
  choice is made in the right crowd.
- It would make a gear decision on a word, and the attack words describe the
  hands: `percussive` is "the string against the fretboard is part of the
  sound", `audible-pick-attack` is "the pick is a sound of its own". Every
  shipped rig asserts them with `kind: llm`. Choosing a block is a bigger claim
  than turning a knob, and an unverified description of a player's hands is not
  enough to make it.

## What is still counted across instruments

Parameter medians and spreads. A bass Deluxe Comp still starts from how every
player sets one. Splitting those by instrument is the same change applied to
`Stats.Models[...].Params`, and waits until something shows it matters: many
models are guitar-only or bass-only already, and the rest need enough bass
samples to be worth a median of their own.

## Regenerating

The embedded statistics were regenerated from the same corpus. Every model's use
count and every median is unchanged. One chain moved from guitar to bass, 158 to
159 bass chains, which moves the bass shares by under one percent.

## Testing

- `corpusgen`: per-instrument counts, once per chain, and presets belonging to
  no instrument are not counted.
- `fill`: the instrument's favourite beats the overall one; a model no chain for
  the instrument held is skipped; a convention whose models the instrument never
  held adds nothing; statistics without counts behave as before.
- Every shipped rig built before and after, with the differences in the pull
  request.

Nobody has listened to the result. That a Deluxe Comp sounds more like Flea than
an LA-2A is not a claim this record makes.
