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
md_extra_excludes := "--exclude 'docs/rigspec.md'"

# Coverage target for this repository.
#
# Not 100%, and the missing part is one file. pkg/sdk/usb.go is every call this
# project makes into libusb, one expression per method, and there is no way to
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
      # //go:build files carry the tag on line 1, blank line, then the header
      if ! head -3 "$f" | grep -q "Copyright (c)"; then
        echo "missing licence header: $f" >&2
        missing=$((missing+1))
      fi
    done < <(find . -name '*.go' -not -path './.git/*' -not -name '*.gen.go')
    if [ "$missing" -ne 0 ]; then
      echo "$missing file(s) missing the header — see CONTRIBUTING.md" >&2
      exit 1
    fi
    echo "licence header present on every .go file"

# Rebuild resources/schemas/hx-stomp.catalog.json from a local HX Edit installation
catalog:
    go run . catalog generate

# Rebuild resources/schemas/gear-map.json from a local HX Edit installation
gear-map:
    uvx --with pypdf --with fonttools python3 resources/schemas/extract_gear_map.py

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
