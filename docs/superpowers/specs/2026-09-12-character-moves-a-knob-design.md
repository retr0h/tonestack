# Character moves a knob

**Status:** implemented\
**Scope:** `pkg/sdk/internal/compile`, and what a build reports

## The problem

A rig can say how it should sound:

```yaml
character:
  - term: mid-forward
  - term: saturated
```

and the preset that comes out is byte for byte the preset that comes out without
it. The vocabulary file says so in its own first line: *"Nothing compiles these
into a chain yet."*

Twenty-four terms across ten axes, validated, spell-checked and suggested when
misspelled, acting on nothing. This project's one sentence is "describe a guitar
or bass sound, get a preset", and the describing half is inert.

## What a term does

A term moves one parameter of the amplifier, from where the corpus says players
put it, by how much the corpus says players disagree.

```text
Mid on an SVT
  corpus median  0.52   p25 0.48  p75 0.56   spread 0.08
  mid-forward    0.52 + 0.08 = 0.60
  scooped        0.52 - 0.08 = 0.44
```

The spread is the step because it is the only honest one available. A parameter
every player sets the same way is one nobody has an opinion about, and a term
should barely move it; a parameter players disagree about is one where an
opinion is worth having, and a term should move it further. Line 6's range would
say the same thing about both.

Where the corpus has too few samples to measure a spread, the step is a tenth of
the catalog's stated range. Values clamp to that range.

## Which knobs

Six axes, because six have a direction that is not a guess. Each names the kind
of block it asks of, not just the control.

| axis      | block  | parameter | why that direction                                                                                                   |
| --------- | ------ | --------- | -------------------------------------------------------------------------------------------------------------------- |
| `mids`    | amp    | `Mid`     | needs no explaining                                                                                                  |
| `highs`   | amp    | `Treble`  | needs no explaining                                                                                                  |
| `drive`   | amp    | `Drive`   | needs no explaining                                                                                                  |
| `low-end` | amp    | `Sag`     | the Pilot's Guide: *lower values offer tighter responsiveness … higher values provide more touch dynamics & sustain* |
| `space`   | reverb | `Mix`     | how much of the room is in the signal                                                                                |
| `attack`  | comp   | `Attack`  | how fast the compressor closes on a note's front edge                                                                |

The remaining four, `decay`, `string-noise`, `pickup` and `movement`, stay
inert, and not for want of effort. Their own definitions describe the player and
the instrument rather than the rig: `quiet-strings` is a left hand, and
`bridge-forward` is a switch position on a guitar this project never sees.
`decay` is the near miss, with three plausible controls and no rule for choosing
between them.

## When a word turns nothing

Three reasons, and calling any of them by another's name is a lie.

Nothing acts on `short-decay`, anywhere, in any chain. That is this project's
gap, and the build says *nothing acts on this yet*.

`audible-pick-attack` is different. It has a control behind it, and this chain
has nowhere to put it: the LA Studio Comp is an opto emulation with no attack
knob at all. The build says *the LA Studio Comp has no Attack*, which is a fact
about the rig somebody can act on, and the same shape covers a chain asking for
room while holding no reverb.

`dry` is neither. Mix at zero and no reverb at all are the same signal, so a rig
asking to stay dry in a chain holding no reverb asked for what it already has.
Nothing was turned and nothing is missing. A `turn` says where that is true, per
axis, because it does not generalise: a chain with no compressor is not a chain
with a soft attack.

A block that lacks the parameter is skipped rather than failed. Not every
amplifier models sag.

## Pairs and scales

Two shapes of axis, and they need different arithmetic.

`mids` is a pair: `mid-forward` and `scooped` are one step either side of the
median. `drive` is a scale: `clean`, `minimal-drive`, `grit-on-attack` and
`saturated` are four positions on one line, so each carries its own multiple of
the step rather than a direction.

## Two words, one question

An axis is what makes a term mean something: saying `mid-forward` has already
said `not scooped`. A rig naming two terms from one axis has answered one
question twice, and the two cancel — apply both and the parameter lands where it
started, which reads as though the rig said nothing at all.

So neither is applied, and the build says which axis was spoken for twice. Said
rather than refused, because that is what this format already does with a word
it does not recognise: refusing a preset over a description would make the
format hostile to the person it exists for.

This is not hypothetical. `mike-dirnt`, shipped, claims both `minimal-drive` and
`grit-on-attack` — two adjacent points on one scale from no breakup to breakup
throughout. Before this was detected they cancelled silently.

## One block per axis

An axis names one kind of block and takes the first of that kind in the chain.
`mid-forward` moves the amplifier's `Mid` and nothing else, even where an EQ
block has one too, and `roomy` moves the first reverb even where the chain holds
two. Two blocks arguing over one axis needs a rule nobody has written, and the
first of a kind is where the described character mostly lives.

## What a build says

`Moved` alongside `Added`, reported the same way: a block put in the chain that
nobody asked for is already named, and a knob turned by a word should be too. A
caller sees which term moved which parameter, from what to what, and which terms
moved nothing.
