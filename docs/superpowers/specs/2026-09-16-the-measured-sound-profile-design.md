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

### The input is somebody's records, and separation is required

Not a DI recording. Nobody hands a tool a clean bass track: the real request is
"I want to sound like Mike Dirnt, here are his songs, work it out". So the input
is a corpus of finished records per artist, and isolating the bass from a full
mix is a requirement rather than an option.

An earlier draft of this record treated separation as undecided and pointed at
DI recordings as the place to start. That was building for the convenience of
whoever writes the code rather than for the person asking the question.

A corpus per artist helps more than it costs. One mix is one engineer's
decisions on one day; several records by the same player, measured and
aggregated, separate what the player does from what a mastering chain did to
them. That is the same move the preset corpus already makes, where 721 presets
give a median rather than one file giving a number.

DI still matters, for one thing only: closing the loop. Playing a known signal
through a generated preset and measuring the return needs a signal whose source
is not in question.

**What separation costs.** Every serious option is a Python machine-learning
model: Demucs, Spleeter, Open-Unmix. There is no Go library, and writing one is
not an afternoon. This is the same bargain the gear map strikes with Python,
scaled up: shell out for the job Go cannot do, keep the measurement here.

Crude alternatives do not work, and this was measured rather than assumed. A
centre-channel extraction with a 250Hz low-pass reads 92% low and a centroid of
154Hz, which sounds like success until you notice both numbers are circular:
everything above 250Hz was filtered out, so of course what is left is low. The
number that gives it away is decay, which drops to 0.13s. That is far too short
for a bass note ringing, and it reads like kick drum transients dominating the
band. Kick and bass share those frequencies and a filter cannot tell them apart.

### WAV only

One format, because a decoder can be trusted with it in a few lines. Anything
else is converted on the way in by whoever has ffmpeg. That is the same bargain
the gear map strikes with Python: shell out for the one job Go should not be
doing, and keep the part that matters here.

## What is not decided

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

## Amended: a measurement nobody could take

Two of the six have no answer for some audio, and both returned a number anyway.

A recording already at its loudest in the first frame holds no attack. A
generated tone is one, and so is a note with no silence in front of it.
`transient` reported the largest wobble in the level after that, which measures
2.5% of the peak for a held sine against 2.9% for a swell. One of those never
started and the other arrived over a second, and by that number they are the
same figure.

A note that never falls to a quarter of its peak has no decay. `decay` returned
whatever was left of the recording, so a tone read as ringing for exactly as
long as somebody happened to record it.

`Profile.Transient` and `Profile.Decay` are a `Reading` now, carrying whether
the recording answered at all. `Across` gathers only the records that did and
counts them, so a middle taken over two of four says so. `Measured()` leaves the
key out instead of writing a figure, and the absent key is what separates a
measurement of zero from one nobody took.

What tells the two attack cases apart is where the signal starts rather than how
far it rose. Across the generators, everything already at level begins at 0.92
to 1.00 of its peak, and everything with a start to find begins under 0.01.
Nothing lands between.

The rest at the end of a note stays a measurement. Reaching one means the player
stopped while the note was still sounding, which is something observed rather
than the recording running out.

Checked against the four bass stems afterwards: all four still report both
figures, because real playing starts from silence.

## Amended: how a measurement becomes a setting

This record left the map unwritten. It is written here, and most of it is a
statement of what cannot be done yet rather than what can.

### A figure on its own names nothing

91% of the energy below 250Hz is not "scooped". Every isolated bass stem is
mostly low, so that figure describes the instrument rather than the player. A
measurement becomes a word only by sitting somewhere in a population of
measurements taken the same way.

So the map is not `centroid 175Hz` to `Treble 0.6`. It is: measure an artist's
records, place that artist against the other artists measured the same way, and
let the position choose the term. Nothing new is needed after that. A term
already moves a control from where the corpus left it, by the corpus spread,
capped at a quarter of the range.

That also answers which half of the system measurement replaces. Gear still
comes from sourced knowledge, cited. Measurement replaces the adjective somebody
chose, and the adjective already knows how to move a knob.

### Four of the six acting axes have a measurement behind them

| axis   | control                   | measurement          | direction                                            |
| ------ | ------------------------- | -------------------- | ---------------------------------------------------- |
| mids   | Mid                       | mid band share       | more is mid-forward, less is scooped                 |
| highs  | Treble                    | high share, centroid | higher is bright, lower is dark                      |
| drive  | Drive                     | harmonics            | energy above the fundamental is what distortion adds |
| attack | Attack, on the compressor | transient            | sharper is percussive, softer is soft-attack         |

Each of those is the same quantity the control acts on, which is why the
direction needs no argument.

### Two do not, and no threshold should be invented for them

**Sag** is touch response and sustain. No band share, centroid or harmonic
figure describes it, and `tight-low-end` against `loose-low-end` cannot be read
off any of them.

**Reverb Mix** is how much room is on the part. Nothing measured separates the
room from the rest of a finished master.

Those two stay words somebody chose, until there is a measurement that means
them. Deriving them from the figures that happen to be available would be
fitting a number to a control because both exist, which is the thing this whole
record was written to avoid.

### What blocks the rest is data, not design

One artist has been measured: four Green Day bass stems. A population of one has
no positions in it, so any threshold derived from it today would be a number
written to fit one player and nobody else.

The next step is measuring a second artist, and a third. That is the same
experiment this record already names above under whether the character words
survive, and it answers both questions at once: if the numbers do not separate
two players the rigs describe differently, no threshold drawn from them was ever
going to mean anything.

### The loop is the other route, and it needs the pedal

Closing the loop needs no population. Playing a signal through a generated
preset, recording the return and measuring it the same way compares a preset
against a target measurement directly, and the difference says which way to move
without anybody deciding what a word means. It stays undecided here for the
reason given above: it needs the pedal connected for every run.
