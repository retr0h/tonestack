# Create a rig for a player

The whole path, from a name to a file that loads.

## Where to look, and what not to accept

Search a list, not the open web. A rig is only as good as where its claims came
from, and the failure this list exists to prevent is a plausible URL off a
search page being pasted in as though somebody had read it.

The standard a source has to meet before it goes into a rig is
[Sourcing a rig](../../CONTRIBUTING.md#sourcing-a-rig). This page is the other
half of it: which places are worth searching, and what each one is good for.

Where the answers have actually come from, roughly in order of how often they
settle something:

| Source                                                                 | Good for                                                                                                                                     |
| ---------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------- |
| **web.archive.org**                                                    | dead magazines. `bassplayer.com` no longer exists and its interviews are the single richest seam here                                        |
| **Forums, searched first**: talkbass.com, reddit, rickresource.com     | people who were there. A Rickenbacker tech who had McCartney's bass on his bench answered a question no magazine had. Read with `just forum` |
| **Fan transcript archives**: `2112.net` for Rush, `ram.org` for Primus | full bodies of print articles that exist nowhere else online                                                                                 |
| **Estate and luthier sites**: `jacopastorius.com`, `ctbasses.com`      | first-party, and often carry build dates that settle which instrument was on which record                                                    |
| **Discogs and Reverb JSON APIs**                                       | sleeve credits, which are the label's own statement of who played what                                                                       |
| **premierguitar.com, guitarworld.com, mixonline.com**                  | rundowns and session pieces, and reprints of dead magazines                                                                                  |
| **Maker artist pages**, archived                                       | endorsements, where the capture date brackets the era                                                                                        |
| **Wikipedia**                                                          | never as a source, only to find the print reference underneath a claim                                                                       |

Two that are not on it. **equipboard.com** looks like a source and is not: it
scrapes Wikipedia, says so in its own text, and tags its own entries
"Unverified". **A search engine** is how you find a page on this list, not a
source in itself.

### Search the forums first, then read them

TalkBass and Reddit come before a search engine, not after it. They are where
people who were there turn up, and a search engine's job is to find a page on
this list rather than to rank the web.

```bash
just forum https://www.talkbass.com/threads/geddy-lee-amps.847050/
just forum https://www.reddit.com/r/Bass/comments/…/…/
```

`just forum` prints a thread as numbered posts, each with its author. Both sites
refuse an ordinary fetch and they refuse it differently, which is worth knowing
because the fix differs too. TalkBass checks the TLS handshake, so no user agent
gets past its Cloudflare challenge;
[resources/read_forum.py](../../resources/read_forum.py) reproduces a browser's
handshake, which is the whole trick. Reddit blocks its JSON API to anonymous
readers however you ask, but still serves RSS to anything sending a browser user
agent.

To search them, put the site in the query rather than trusting a general web
search to surface it:

```text
site:talkbass.com geddy lee ampeg cabinets 1977
site:reddit.com/r/Bass mccartney rickenbacker flatwounds wings
```

Searching Reddit goes through the MCP server that `.mcp.json` declares, which
needs no account and nothing configured: it reads Reddit's RSS feeds, which
still serve where the JSON API refuses anonymous readers. Expect a 429 if you
hurry it. See [Reading Reddit](../../CONTRIBUTING.md#reading-reddit).

Then open every thread you intend to cite. This is not a formality. Every forum
citation in this repository before `just forum` existed was entered from a
search result and never opened, and reading them found:

- a quote that **does not appear in the thread at all**. The McCartney tone
  thread was cited for "sounds like he boosts low mids and rolls off the treble
  a little". The word "boost" appears zero times across its 33 posts.
- a claim **changed in the copying**. The Geddy thread says "a bunch of Ampeg
  SVT amps" in the Fly by Night film. It was cited as an SVT-810 cabinet, which
  is a different object, and five period sources say he used neither.
- a thread about the **wrong era**. One McCartney citation came from a thread
  whose first post asks about "the 1960's", cited in a rig covering 1975 to
  1979\.

### Cite the person, not the thread

A thread holds the best and the worst evidence in this repository, often on the
same page. One rickresource.com thread carries both the tech who restrung the
instrument:

> When I worked on his bass in the mid 1970's he did NOT have rotosound strings
> of any kind on it. They were flats but not rickenbacker or roto's.

and, in the post directly above it, somebody guessing:

> Rotosound's are a possibility but I don't know.

So cite the person, not the thread. A post counts when a named person with
first-hand access is talking about something they did or saw, and it does not
count when it is opinion, however confident. Say which one you have.

Quote the post you are relying on into `note`, and name what makes the person
worth believing: the tech above is evidence because he had the instrument on his
bench, not because he posted confidently. A thread that turns out to be somebody
guessing is a thread with nothing in it, and the honest thing is to leave the
claim unsourced rather than dress the guess up in a url.

### 1. Find out whether the gear exists

Before writing anything. A rig naming an amplifier no device models is only
discovered at build time, and by then the name has usually been copied somewhere
else.

```bash
tonestack catalog list --search ampeg
tonestack catalog list --subcategory bass --category amp
```

Names are real-world gear. "Ampeg SVT", "Klon Centaur", never model identifiers.
[catalog.md](../catalog.md) explains where that mapping comes from and why it
exists.

### 2. Find out what else belongs in the chain

```bash
tonestack corpus show --instrument bass
```

This is measured over 4,324 real presets, not asserted. It says a bass chain
holds a compressor 88% of the time and that drive sits ahead of the amp in 88%
of the chains that have one. You do not have to act on it, since the build fills
in what is near-universal and says so, but it tells you what a complete chain
for that instrument looks like.

### 3. Write it

```bash
tonestack recipes new \
  --id mike-dirnt --name "Mike Dirnt" --band "Green Day" \
  --instrument bass --amp "Ampeg SVT" --cab "Ampeg 8x10"
```

Every gear name is resolved before anything is written, and near misses are
suggested when one does not resolve.

The file goes to your own recipes directory, `$XDG_DATA_HOME/tonestack/recipes`,
or `~/.local/share/tonestack/recipes` when that variable is unset, and the
command prints its path. `recipes list`, `recipes show`, `presets make` and the
MCP server read that directory beside the rigs that ship, so the new rig builds
straight away. `--dir` writes somewhere else instead. A rig written there is
only found when you name the same directory again, with `--dir` or with
`presets make --recipes`, and the rigs that ship are read beside it either way.

`--kind` changes what a copy made with `--from` is attributed to, and is refused
without `--from`. A rig written from gear is always an artist.

Then open the file. The flags write the chain; everything that makes the rig
worth reading is what you add next.

| field           | what it carries                                  | why it is not optional in practice                                                  |
| --------------- | ------------------------------------------------ | ----------------------------------------------------------------------------------- |
| `evidence`      | where each gear claim came from, per chain entry | a chain with no sources is a model's guess wearing a filename                       |
| `confidence`    | how much of the above you believe                | `low` is an honest answer and the tool displays it as unverified                    |
| `subject.era`   | when the rig applied, in words                   | a rig that does not say when it applies claims to be timeless and usually is not    |
| `subject.years` | the same period as numbers                       | nothing can be measured for this rig without it: records are held to these years    |
| `played`        | the instrument and its strings                   | a figure measured without it is attributed to the amplifier, which did not cause it |
| `technique`     | pick or fingers, and where on the string         | a picked record tuned by ear sounds dull played fingered                            |
| `character`     | how it should sound, each word with its evidence | this is what moves controls when the preset is built                                |

[recipes.md](../recipes.md) has a section per field and
[`examples/rigspec/mike-dirnt.yaml`](../../examples/rigspec/mike-dirnt.yaml)
shows all of them on one subject. Fill what you can source and leave the rest
out: an empty field is a gap somebody can close, and a filled one nobody checked
is a claim this project will repeat back as fact.

**Be honest about where the gear came from.** `kind: llm` on a piece of evidence
means a model asserted it and nobody checked. That is reliable for well-known
players, unreliable for obscure ones, and the model cannot tell which it is
doing, which makes it the largest correctness risk here.

**When the page is dead, cite the archived copy and pin it.** Most of the
interviews worth citing here are twenty years old and half the magazines that
ran them are gone. `bassplayer.com` no longer resolves and its article bodies
are only in the Internet Archive. Write the capture, not the dead original:

```text
https://web.archive.org/web/20040908171044/http://www.bassplayer.com/archive/0904/0904_Features1.htm
└─ the archive ─┘└── capture ──┘└────────── the page as it was ──────────┘
```

The fourteen digits are the capture's timestamp, `YYYYMMDDhhmmss`, so that one
is 8 September 2004. Use that form rather than the shorthand
`web.archive.org/web/2020/<url>`, which redirects to whatever capture happens to
be nearest and can therefore point somewhere else later. Resolve the shorthand
once and write down what it gave you:

```bash
curl -sI "https://web.archive.org/web/2020/<url>" | grep -i location
```

Two habits that go with this. Say in the `note` or `caveat` that the original is
gone, so a reader knows the ugly URL is the only one there is. And prefer
`https`, which the archive serves, even when the captured page inside the URL is
`http`: that inner address is a label for what was fetched, not something
anybody re-fetches.

**If this player will ever be measured**, the records go in
`resources/music/<instrument>/<id>/`, and `<id>` is this rig's identifier. That
directory name is the only thing joining a rig to its records: spell it
differently and the rig reads as one nobody has measured, while the records read
as belonging to nobody. `tonestack recipes records --corpus <dir>` says so when
it happens. [Add records to a corpus](add-records-to-a-corpus.md) is the rest.

### 4. Build it

```bash
tonestack presets make --id mike-dirnt --out mike.hlx
```

The output is the point. It reports every block chosen, what real gear each one
emulates, what it costs, how much of the processor is used, anything added that
the rig did not ask for, and what each word in the rig's `character` did:

```text
  ●  0.0  Deluxe Comp       Line 6 Original             1.8
  ●  0.1  Ampeg SVT Brt     Ampeg SVT (bright channel)  26.6
  ●  0.2  8x10 Ampeg SVT-E                              7.2

  dsp0  ████████░░░░░░░░░░░░░░░░  35.6%

  added Deluxe Comp — almost every chain has one (88% of chains)

  heard mid-forward — Mid 0.79 to 0.89
  heard grit-on-attack — Drive 0.60 to 0.72
  heard tight-low-end — Sag 0.50 to 0.40
  heard short-decay — nothing acts on this yet
  heard audible-pick-attack — Attack 0.04 to 0.05
```

[recipes.md](../recipes.md#character-describes-the-result-not-the-control)
explains the `heard` lines.

Read it before you plug anything in. A wrong amp is a bad miss that nothing
downstream recovers from, and it is visible right there.

### 5. Get it onto the device

See [below](get-it-onto-the-device.md).
