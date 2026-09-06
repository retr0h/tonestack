#!/usr/bin/env bash
# Rebuild the preset corpus from the sources in repos.txt.
#
# Idempotent: skips repositories already present. Pass --force to refetch.
# Clones go to a temp dir and are deleted; only preset files are kept.
set -euo pipefail

cd "$(dirname "$0")"
FORCE="${1:-}"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

kept=0 skipped=0 failed=0

while IFS=$'\t' read -r repo lic; do
  [[ "$repo" =~ ^#|^$ ]] && continue
  dest="github/${repo//\//__}"

  if [[ -d "$dest" && "$FORCE" != "--force" ]]; then
    skipped=$((skipped+1)); continue
  fi

  if ! git clone --depth 1 -q "https://github.com/$repo.git" "$TMP/c" 2>/dev/null; then
    echo "  FAIL  $repo" >&2; failed=$((failed+1)); continue
  fi

  mkdir -p "$dest"
  # Preset and container files only. Nothing else from the clone is kept.
  find "$TMP/c" \( -iname '*.hlx' -o -iname '*.hls' -o -iname '*.hlb' -o -iname '*.pgs' \) \
    -exec cp -n {} "$dest/" \; 2>/dev/null || true

  n=$(find "$dest" -type f | wc -l | tr -d ' ')
  if [[ "$n" == "0" ]]; then rmdir "$dest" 2>/dev/null || true
  else echo "  ok    $repo ($n files, $lic)"; kept=$((kept+1)); fi

  rm -rf "$TMP/c"
done < repos.txt

echo
echo "fetched $kept, skipped $skipped (already present), failed $failed"
echo "next: helixctl corpus unpack   # decode setlist containers"
echo "      helixctl catalog extract # rebuild ../hx-stomp.catalog.json"
