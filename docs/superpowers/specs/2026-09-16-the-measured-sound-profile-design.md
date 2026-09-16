# The measured sound profile

A design record. Dated, and superseded rather than rewritten.

## The goal

"Here is a Nirvana song, make me sound like the bass player."

Everything built before this is preparation for that sentence. A rig names gear
and moves knobs by words somebody chose; nothing yet listens to a recording and
says what it is.

## The problem with words

A rig says `percussive` or `warm`. Two people mean different things by either,
and neither can be checked. The corpus work made the knob positions measurable,
and the character terms honest about being somebody's judgement, but the input
is still a word.

A measurement is not. A spectral centroid of 410Hz is the same number for
everyone who computes it, and two recordings can be compared without either of
us describing them.

So the direction is: **measurements are the input, and words become labels
derived from them**, if they survive at all.

## What this record decides

Measure first. Nothing about rig generation is settled here, deliberately,
because the vocabulary question has to be answered with data before anything is
built on top of it.

### The measurements

Six, chosen for a bass guitar rather than for a full mix.

| measure       | unit    | what it separates                         |
| ------------- | ------- | ----------------------------------------- |
| band energy   | share   | where the sound sits: low, mid, high      |
| centroid      | hertz   | its centre of gravity                     |
| transient     | 0 to 1  | a pick or a slap from fingers or a swell  |
| decay         | seconds | muted playing from a ringing open string  |
| dynamic range | dB      | how compressed the playing is             |
| harmonics     | share   | how much distortion, and even against odd |

Band edges are 250Hz and 2kHz. A four-string's open E is about 41Hz and its
highest fretted note is under 400Hz, so the lower edge sits above the
fundamentals and below the body, and the upper above anything the instrument
sounds without a pick or a fret buzzing.

### Nothing here decides what a measurement means

`pkg/sdk/audio` returns numbers. Turning 410Hz into "warm" is a judgement and
belongs where judgements are made. Mixing the two would leave nowhere to stand
when they disagree, which is the same reason the catalog records what Line 6
states and the corpus records what players do, separately.

The renderer follows the same rule: it says what each number counts, never what
it means.

### Built on signals whose answers are known

Every measurement was checked against synthetic signals before any music was
involved: a sine sits at its own frequency and holds nothing else, a square
carries odd harmonics at a third and a fifth of the fundamental, a plucked note
decays and a held one does not, silence measures as nothing.

This caught a real error. The first transient measurement read how much the
rises in level varied among themselves, which scored a cleanly struck note at
**zero**, because one rise has nothing to vary against. Measured across a struck
note, a swelled note, a held tone, a square and noise, the number that actually
separates them is the largest single rise against the peak: struck notes land
above 0.94 and everything gradual below 0.1. The definition was replaced on that
evidence rather than tuned until a test passed.

### WAV only

One format, because a decoder can be trusted with it in a few lines. Anything
else is converted on the way in by whoever has ffmpeg. That is the same bargain
the gear map strikes with Python: shell out for the one job Go should not be
doing, and keep the part that matters here.

## What is not decided

**Stem separation.** Measuring a bass part inside a full mix needs the part
isolated first, and every good option is a Python machine-learning model. That
is a far heavier dependency than a PDF reader, and it deserves its own decision
rather than being smuggled in. Not needed for a DI recording or a bass-forward
track, which is where this starts.

**Whether the character words survive.** The point of measuring first is to find
out whether the numbers separate the artists the rigs already describe. If a
measured profile tells Flea from Geddy Lee, words may become labels derived from
measurements. If it does not, the words were carrying something the measurements
miss, and that is worth knowing before rewriting them.

**How measurements become settings.** Gear still comes from sourced knowledge:
who played what, cited. Measurements should drive the settings, and the map from
one to the other is unwritten.

**Closing the loop.** A Helix is a USB audio interface. Playing a DI track
through a generated preset, recording the return and measuring it the same way
turns "sounds right" into a number before a human listens. That is the check
that would make the whole thing self-correcting, and it needs the pedal
connected for every run.

## Where it lives

`pkg/sdk/audio`, because measuring a recording passes all three questions in
[where a package belongs](2026-09-09-where-a-package-belongs-design.md): a
hosted service, an agent on somebody's desktop and a stranger writing their own
tool would each call it.

`tonestack measure --file take.wav` prints a profile. It reads a file and prints
numbers; it does not touch a device.
