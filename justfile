set allow-duplicate-variables

# Optional modules: mod? allows `just fetch` to work before .just/remote/ exists.
import? '.just/remote/go.just'
import? '.just/remote/md.just'
import? '.just/remote/just.just'

# No documentation site, so md formats every markdown file in the repository.
md_site_dir := ""

# Except the one nobody writes. docs/rigspec.md is generated from the RigSpec
# contract, and a test compares it against what the generator produces — so
# reflowing it here would leave the page disagreeing with its own source.
md_extra_excludes := "--exclude 'docs/rigspec.md' --exclude 'docs/commands.md'"

# Coverage target for this repository.
#
# Not 100%, and the missing part is one file. pkg/sdk/internal/device/usb_darwin.go
# is every call this project makes into IOKit, translation and nothing more, and there is no way to
# reach it without a device on the bus. Everything it forwards to — finding a
# device, choosing between two, claiming an interface, waiting on a busy one,
# framing, sequence numbers, acknowledgements — is behind an interface and
# covered.
#
# It is counted rather than excluded on purpose. An exclusion hides a file's
# size; a target says what is not reachable and gets worse if that file grows.
go_coverage_target := "99"

# --- Fetch ---

# Fetch shared justfiles from osapi-justfiles
fetch:
    mkdir -p .just/remote
    curl -sSfL https://raw.githubusercontent.com/osapi-io/osapi-justfiles/refs/heads/main/go/go.just -o .just/remote/go.just
    curl -sSfL https://raw.githubusercontent.com/osapi-io/osapi-justfiles/refs/heads/main/md/md.just -o .just/remote/md.just
    curl -sSfL https://raw.githubusercontent.com/osapi-io/osapi-justfiles/refs/heads/main/just/just.just -o .just/remote/just.just

# --- Top-level orchestration ---

# Install all dependencies
deps:
    just go-deps
    just go-mod

# Run all tests
test:
    just license-check
    just go-test

# Round-trip a preset on an attached Helix. Overwrites TONESTACK_SCRATCH_SLOT and puts it back
test-device:
    go test -tags device -count=1 -v -run TestDevicePublicTestSuite ./pkg/sdk/

# --- Data generation ---
#
# The gear map is bootstrap data: it changes only when Line 6 ships new models,
# so the result is committed and this is run by hand after an HX Edit update.
# Python because the manual's model-name column uses a subset-embedded font no
# Go PDF library decodes. The frequently regenerated artifact is the catalog,
# and that is Go.

# Every .go file must carry the MIT licence header
license-check:
    #!/usr/bin/env bash
    set -euo pipefail
    missing=0
    while IFS= read -r f; do
      # Nobody writes generated code, so nobody puts a header on it. The
      # marker is what Go tooling itself keys on, so a generated file is
      # skipped whatever it is called and a hand-written one never is.
      # mockgen puts it on line 1, oapi-codegen under a package comment.
      generated=$(head -5 "$f")
      if grep -q '^// Code generated .* DO NOT EDIT\.$' <<<"$generated"; then
        continue
      fi
      # //go:build files carry the tag on line 1, blank line, then the header
      header=$(head -3 "$f")
      if ! grep -q "Copyright (c)" <<<"$header"; then
        echo "missing licence header: $f" >&2
        missing=$((missing+1))
      fi
    done < <(find . -name '*.go' -not -path './.git/*')
    if [ "$missing" -ne 0 ]; then
      echo "$missing file(s) missing the header — see CONTRIBUTING.md" >&2
      exit 1
    fi
    echo "licence header present on every .go file"

# Refresh the embedded catalog from a local HX Edit installation; skips without one
catalog:
    go generate ./pkg/sdk/internal/catalogen/

# Refresh the embedded corpus statistics from resources/schemas/corpus; skips without it
corpus:
    go generate ./pkg/sdk/internal/corpusgen/

# Rebuild resources/schemas/gear-map.json from a local HX Edit installation
gear-map:
    uvx --with pypdf --with fonttools python3 resources/schemas/extract_gear_map.py

# Separate one instrument out of every recording in a directory, for `tonestack measure --dir`
#
# A mix measures the band, so the instrument has to come out of it before any
# number describes the player. Python because Demucs is; the same category as
# ffmpeg converting an MP3, and nothing downstream of it leaves Go.
#
# INSTRUMENT is bass or guitar, and it chooses the model as well as the stem.
# The default four-source model has no guitar in it: drums, bass, vocals and
# one bucket called "other" holding everything else. Guitar needs htdemucs_6s,
# which separates six and is slower. A guitar stem is also a worse stem than a
# bass one — six sources share the same training and a guitar overlaps the
# vocals and the keys far more than a bass does — so a figure measured from it
# carries more of the rest of the band with it.
#
# `--with numpy` is not optional: Demucs does not declare it and fails without it.
stems IN OUT INSTRUMENT="bass":
    #!/usr/bin/env bash
    set -euo pipefail
    model=htdemucs
    if [ "{{ INSTRUMENT }}" = "guitar" ]; then model=htdemucs_6s; fi
    uvx --from demucs --with numpy demucs -n "$model" \
      --two-stems={{ INSTRUMENT }} -o {{ OUT }} {{ IN }}/*
    echo "stems written to {{ OUT }}/$model — measure them with:"
    echo "    tonestack measure --dir {{ OUT }}/$model"

# Download one record into an artist's corpus, named for its manifest entry
#
# URL is the Spotify track the manifest links to, so the record measured is the
# one the evidence names. When spotdl cannot find the audio, pass
# "YOUTUBE|SPOTIFY" and it takes the audio from that video. The file is named
# TRACK because that is what `tonestack measure --manifest` matches against.
record DIR TRACK URL:
    uvx spotdl download "{{ URL }}" --output "{{ DIR }}/{{ TRACK }}.{output-ext}"
    @test -f "{{ DIR }}/{{ TRACK }}.mp3" || { echo "no {{ DIR }}/{{ TRACK }}.mp3: see docs/workflows/add-records-to-a-corpus.md" >&2; exit 1; }

# Read a TalkBass or Reddit thread as plain text, one post per author
#
# Both refuse an ordinary fetch and for different reasons: TalkBass checks the
# TLS handshake, Reddit checks the user agent. read_forum.py sends each what it
# wants, because sending both to both fails both.
forum URL:
    uvx --with curl_cffi python3 resources/read_forum.py "{{ URL }}"

# Search Reddit for threads worth reading
#
# SUB narrows it to one subreddit and may be left off. Reddit throttles hard,
# so an empty result means try again in a minute rather than that nothing is
# there. Finding the thread is the half a web search cannot do here: it has a
# session budget and reddit.com is blocked to it outright.
forum-search QUERY SUB="":
    uvx --with curl_cffi python3 resources/read_forum.py --search "{{ QUERY }}" {{ SUB }}

# Search the open web for pages worth reading
#
# For finding a thread or an article, never for citing one: nothing here is
# evidence until somebody opens it. Brave answers a plain fetch where the other
# engines refuse, and rate-limits after a few queries, so pace it. Put the site
# in the query to reach TalkBass, which has no search anybody here can use:
#
# just web "site:talkbass.com geddy lee ampeg cabinets 1977"
web QUERY:
    uvx --with curl_cffi python3 resources/read_forum.py --web "{{ QUERY }}"

# Generate code
generate:
    just go-generate

# Format, lint, and generate before committing
ready:
    just generate
    just md-fmt
    just go-fmt
    just go-vet
    just just-fmt
