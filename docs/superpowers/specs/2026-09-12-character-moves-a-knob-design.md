# Character moves a knob

**Status:** proposed\
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

Four axes, because four have a direction that is not a guess.

| axis      | parameter | why that direction                                                                                                   |
| --------- | --------- | -------------------------------------------------------------------------------------------------------------------- |
| `mids`    | `Mid`     | needs no explaining                                                                                                  |
| `highs`   | `Treble`  | needs no explaining                                                                                                  |
| `drive`   | `Drive`   | needs no explaining                                                                                                  |
| `low-end` | `Sag`     | the Pilot's Guide: *lower values offer tighter responsiveness … higher values provide more touch dynamics & sustain* |

The other six — `decay`, `attack`, `space`, `string-noise`, `pickup`, `movement`
— are not amplifier controls. They stay inert, and a build says which terms it
acted on and which it only recorded, so nobody reads a preset believing `glassy`
did something.

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

## The amplifier only

First pass. `mid-forward` moves the amplifier's `Mid` and nothing else, even
where an EQ block in the chain has one too. Two blocks arguing over one axis
needs a rule nobody has written, and the amplifier is where the described
character mostly lives.

## What a build says

`Moved` alongside `Added`, reported the same way: a block put in the chain that
nobody asked for is already named, and a knob turned by a word should be too. A
caller sees which term moved which parameter, from what to what, and which terms
moved nothing.
