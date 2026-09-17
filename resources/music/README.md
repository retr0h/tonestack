# music

Records measured to describe how somebody plays, one directory per artist.

Nothing here travels. The audio is somebody else's and the stems are cut from
it, so both are git-ignored. What is committed is `corpus.yaml`: which songs
were measured, and a link so a person who does not own them can still check what
was claimed. That is the same split [schemas/corpus/](../schemas/corpus/) uses
for presets, where the fetch script and the attribution are kept and the payload
is not.

```text
resources/music/flea/
  corpus.yaml                 committed
  Some Record.mp3             ignored
  stems/htdemucs/...          ignored
```

## Measuring one

```bash
just stems resources/music/flea resources/music/flea/stems
tonestack measure --dir resources/music/flea/stems/htdemucs \
  --manifest resources/music/flea/corpus.yaml
```

`track` in the manifest matches the stem directory, which is the source file's
own name without its extension. A name that matches nothing is reported rather
than ignored, and so is a recording nothing names.

[workflows/measure-a-players-sound.md](../../docs/workflows/measure-a-players-sound.md)
is the full procedure.

## Adding one

```bash
just record resources/music/flea aeroplane https://open.spotify.com/track/0VLdJcQUsqHBBwqPp4CIKJ
```

[workflows/add-records-to-a-corpus.md](../../docs/workflows/add-records-to-a-corpus.md)
says how to pick the record and check the file is the right recording.

## Choosing records

Three or four each, with the bass prominent enough to separate cleanly, and from
**different productions**. One album's tracks share a room, an engineer and a
master, so what they have in common may be the studio rather than the player.
Spreading the records is what leaves the player as the thing they share.
