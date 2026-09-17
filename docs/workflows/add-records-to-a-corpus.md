# Add records to a corpus

Fetching the records a measurement needs into `resources/music/`, for when
somebody asks an agent to grow a player's corpus. The agent does the whole
thing, from choosing the record to checking the file, but only when asked. Needs
uv and ffmpeg; see [Prerequisites](../../CONTRIBUTING.md#prerequisites).

This page is step 1 of [Measure a player's sound](measure-a-players-sound.md)
done by a machine. Every rule in
[resources/music/README.md](../../resources/music/README.md) about choosing
records still applies, and the download is the easy part. Most of what goes
wrong is a file that downloads cleanly and is the wrong recording.

### When a corpus needs more

- `tonestack measure --manifest` reports
  `named in the manifest but not measured`. The manifest names a record whose
  audio is not on disk.
- An entry has no `url`. The record cannot be fetched again and nobody can check
  what was measured, so the entry needs a link before it needs anything else.
- The artist has fewer than three records, or they all come from one production.
- A width is too wide to call a habit, and one record sits alone at one end of
  it. A fourth record says whether that one is the outlier.

An agent that sees one of these says so and names the records it would add. It
downloads them when asked to.

### 1. Check the player played the bass on that recording

Do this before anything else, because no later step catches it. A song credited
to a band is not proof of who played its bass, or that a bass guitar is on it at
all.

Stevie Wonder's "Higher Ground" has no bass guitar. The line is a Moog
synthesiser, played by Wonder. Downloaded into Flea's corpus by title, it would
have measured a synthesiser as a bassist, with transients no bass guitar
produces. Flea's record is the Red Hot Chili Peppers cover on *Mother's Milk*.
Check the credits, and take most care with covers, where the title names two
recordings.

### 2. Name it in the manifest first

Add the entry to `resources/music/<artist>/corpus.yaml` before downloading, with
the Spotify track link as its `url`:

```yaml
  - track: silly-love-songs
    url: https://open.spotify.com/track/…
    note: Wings at the Speed of Sound, 1976
```

The link does two jobs. It is the evidence a person without the file can check,
and it is what the next step downloads, so the record measured is the record the
evidence names.

Every entry carries a `url`. An entry without one is not finished, whatever else
it has. Flea, Mike Dirnt and Paul McCartney each got three tracks and no links,
and once the audio was gone nobody could fetch those records again or check
which takes had been measured. The album in `note` does not stand in for the
link. "Mother's Milk, 1989" does not say which of the several Spotify tracks
called "Higher Ground" was meant, and step 1 is about the one that would be
wrong.

`ReadManifest` refuses an entry with no url, the way it refuses one with no
track name, so a manifest missing a link fails where it is read rather than
years later when somebody wants the record back.

Use the link, never a search. Asked for a song that does not exist, spotdl
downloaded "ZzONE - NOBODY" rather than failing. A real title gets the same
treatment, and a wrong match is harder to spot.

Check the link before writing it down.
`uvx spotdl save "<url>" --save-file song.spotdl` prints the artist, album, year
and duration Spotify holds for it, which is what tells a search result apart
from the record you wanted. Searching for "Suck My Kiss" returned two links that
were both the song "Blood Sugar Sex Magik", and one "Welcome to Paradise" was a
1994 Chicago live broadcast.

### 3. Download it under the manifest's name

```bash
just record resources/music/paul-mccartney silly-love-songs \
  https://open.spotify.com/track/…
```

The url is the one the manifest already holds. Copy it from there rather than
finding it a second time, or the file on disk and the evidence beside it stop
being the same recording. An entry that also carries a `source` is fetched from
both, source first, which is what step 4 writes down. Reading every track and
the string that fetches it:

```bash
awk '/^  - track:/{ if(t!="") print t, (s!=""? s"|"u : u); t=$3; u=""; s="" }
     /^    url:/{ u=$2 } /^    source:/{ s=$2 }
     END{ if(t!="") print t, (s!=""? s"|"u : u) }' \
  resources/music/flea/corpus.yaml
```

Reset `u` and `s` on every new track. An awk that only tracks the current name
prints the first entry's link against every later one, and the recipe then
writes the wrong recording under a name that looks right. That happened here:
"Aeroplane" landed as `higher-ground.mp3` and only the length check caught it.

The recipe runs `uvx spotdl` and writes `silly-love-songs.mp3`. The name matters
because `track` in the manifest matches the file's name without its extension.
spotdl's own naming, `Wings - Silly Love Songs - 2014 Remaster.mp3`, matches
nothing, and `measure` reports it as unnamed.

The recipe fails if no file appears. Do not read success from spotdl's output.

### 4. When spotdl cannot find the audio

spotdl reads the song from Spotify and fetches audio from YouTube. Two failures
are routine and neither is fatal.

`YouTube Music returned no usable results … after 3 attempts` on its own is
fine. spotdl falls back to YouTube and usually downloads anyway.

`AudioProviderError: YT-DLP download error` usually passes. Run the recipe again
once.

If it still fails, find the video yourself and pass both links, YouTube first:

```bash
uvx yt-dlp --flat-playlist \
  --print "%(id)s | %(title)s | %(duration_string)s" \
  "ytsearch6:Wings Silly Love Songs"

just record resources/music/paul-mccartney silly-love-songs \
  "https://www.youtube.com/watch?v=…|https://open.spotify.com/track/…"
```

Pick the result whose length matches the Spotify track and whose title says
nothing about live, cover, remix or a performance name like *Rockshow*. The
Spotify half still supplies the tags. Never pass the YouTube link alone: spotdl
then searches Spotify for the video's title, and one link for "Aeroplane" came
back tagged as a Tape B song.

Write the link that worked into the entry's `source`:

```yaml
  - track: higher-ground
    url: https://open.spotify.com/track/3gDxDN503aZr5aeFtn6VwI
    source: https://www.youtube.com/watch?v=3jFHoGaempM
    note: Mother's Milk, 1989
```

`url` stays the evidence, and is what a rig quotes. `source` is only the
downloader's way back to the same audio, so the next person to want this record
runs step 3 and gets it rather than repeating the search. Both Flea records need
one: every Spotify link tried for them failed here, four apiece, while the other
seven in the corpus came down from their url alone.

Trying another Spotify link first is still worth it, and cheaper than this. It
did not help for those two, but the link a search returns is often a live take
or a compilation edit, and a different one for the same recording downloads
fine.

### 5. Check it is the recording

A wrong file measures without complaint, so compare lengths. Spotify's is in
what spotdl saves:

```bash
uvx spotdl save "https://open.spotify.com/track/…" --save-file song.spotdl
# song.spotdl is JSON: "duration" is in seconds, beside "album_name" and "year"
```

The file's length, and any silence at its end:

```bash
ffprobe -v error -show_entries format=duration \
  resources/music/paul-mccartney/silly-love-songs.mp3
ffmpeg -hide_banner -i resources/music/paul-mccartney/silly-love-songs.mp3 \
  -af silencedetect=n=-45dB:d=1 -f null - 2>&1 | grep silence_
```

Subtract trailing silence from the file's length and it should land within a few
seconds of Spotify's. Lyric videos pad the end: the "Silly Love Songs" file was
6:05, with 12.7 seconds of silence, leaving 5:52 of music against the album's
5:54. A difference the silence does not explain means a different recording.
Delete it and go back to step 4.

Compilations are fine when they reuse the recording. "Higher Ground" came back
tagged *Greatest Hits*, 2003, which carries the *Mother's Milk* take.

### 6. Separate and measure

Carry on from step 2 of
[Measure a player's sound](measure-a-players-sound.md#2-separate-the-bass-from-each).
The audio stays git-ignored. Commit only the manifest change.
