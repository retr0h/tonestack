# music

Records measured to describe how somebody plays: one directory per instrument,
then one per artist.

```text
resources/music/bass/flea/
```

The instrument is not tidiness. A word is earned by sitting clear of the other
players in the tree, and a guitar's centre of gravity sits an octave above a
bass guitar's. One guitarist in a corpus of bassists would earn every bassist
`dark` and every guitarist `bright`, and both words would stop meaning anything.
So the directory that holds a comparison is the instrument, and
`measure --corpus` is pointed at one of them.

Nothing here travels. The audio is somebody else's and the stems are cut from
it, so both are git-ignored. What is committed is `corpus.yaml`: which songs
were measured, and a link so a person who does not own them can still check what
was claimed. That is the same split [schemas/corpus/](../schemas/corpus/) uses
for presets, where the fetch script and the attribution are kept and the payload
is not.

```text
resources/music/bass/flea/
  corpus.yaml                 committed
  Some Record.mp3             ignored
  stems/htdemucs/...          ignored
```

## Keeping it

Git will not bring this back. The audio and the stems are ignored, so nothing
here is in a commit, and the only copy is the one on your disk.

Two things delete it, and neither announces itself:

- `git clean -fdx`, which removes ignored files. That is the whole corpus.
- Removing a git worktree you filled. A worktree has its own `resources/music/`,
  and audio written there is inside that temporary directory rather than beside
  the manifests. It goes when the worktree does.

So fetch, separate and measure from the main checkout. Nine records were lost
this way once, and refetching them took longer than the separation had.

Keep the `.mp3` beside the stems rather than deleting it once the stems exist.
`measure` reads the stems, so a missing source file is silent until somebody
wants to separate it again with different settings, and then it is gone.

To see what the manifest names but the disk does not hold:

```bash
tonestack measure --dir resources/music/bass/flea/stems/htdemucs \
  --manifest resources/music/bass/flea/corpus.yaml
```

It reports `named in the manifest but not measured` for each one.

## Measuring one

```bash
just stems resources/music/bass/flea resources/music/bass/flea/stems
tonestack measure --dir resources/music/bass/flea/stems/htdemucs \
  --manifest resources/music/bass/flea/corpus.yaml
```

`track` in the manifest matches the stem directory, which is the source file's
own name without its extension. A name that matches nothing is reported rather
than ignored, and so is a recording nothing names.

[workflows/measure-a-players-sound.md](../../docs/workflows/measure-a-players-sound.md)
is the full procedure.

## Adding one

```bash
just record resources/music/bass/flea aeroplane https://open.spotify.com/track/0VLdJcQUsqHBBwqPp4CIKJ
```

[workflows/add-records-to-a-corpus.md](../../docs/workflows/add-records-to-a-corpus.md)
says how to pick the record and check the file is the right recording.

## Choosing records

Three or four each, with the bass prominent enough to separate cleanly, and from
**different productions**. One album's tracks share a room, an engineer and a
master, so what they have in common may be the studio rather than the player.
Spreading the records is what leaves the player as the thing they share.
