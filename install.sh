#!/usr/bin/env bash
#
# tonestack installer
# Usage: curl -fsSL https://github.com/retr0h/tonestack/raw/main/install.sh | bash
#
# Env overrides:
#   TONESTACK_VERSION       install a specific version (e.g. 1.1.1) instead of latest
#   TONESTACK_INSTALL_DIR   force install destination, skipping the default rules

set -euo pipefail
APP=tonestack
REPO=retr0h/tonestack

# Mirrors the roles in internal/cli/theme.go so the installer and the
# installed program paint with the same palette. ACCENT is the amber a tube
# glows (#ffa032) as a truecolor escape; MUTED is the dim attribute lipgloss
# uses for its Mute role, so both agree without naming a grey.
MUTED='\033[0;2m'
RED='\033[0;31m'
ACCENT='\033[38;2;255;160;50m'
NC='\033[0m' # reset

err() {
    printf "${RED}error:${NC} %b\n" "$1" >&2
    exit 1
}

have() {
    command -v "$1" >/dev/null 2>&1
}

# banner prints the same block letters, split the same way, as
# cli.Banner: the top line muted, the bottom line in the accent.
banner() {
    printf "\n${MUTED}▀█▀ █▀█ █▄░█ █▀▀ █▀ ▀█▀ ▄▀█ █▀▀ █▄▀${NC}\n"
    printf "${ACCENT}░█░ █▄█ █░▀█ ██▄ ▄█ ░█░ █▀█ █▄▄ █░█${NC}\n\n"
    printf "${MUTED}describe a guitar or bass sound, get a Line 6 Helix preset${NC}\n\n"
}

detect_os() {
    case "$(uname -s)" in
        Darwin) os=darwin ;;
        Linux) os=linux ;;
        *) err "unsupported operating system: $(uname -s)\n  build from source: https://github.com/${REPO}#development" ;;
    esac
}

detect_arch() {
    case "$(uname -m)" in
        x86_64 | amd64) arch=amd64 ;;
        arm64 | aarch64) arch=arm64 ;;
        *) err "unsupported architecture: $(uname -m)" ;;
    esac
}

http_get() {
    if have curl; then
        curl -fsSL "$1" -o "$2"
    elif have wget; then
        wget -qO "$2" "$1"
    else
        err "neither curl nor wget is available"
    fi
}

resolve_version() {
    if [ -n "${TONESTACK_VERSION:-}" ]; then
        version="${TONESTACK_VERSION#v}"
        return
    fi

    tmp_tag=$(mktemp)
    http_get "https://api.github.com/repos/${REPO}/releases/latest" "$tmp_tag" \
        || err "could not reach GitHub to find the latest release"

    version=$(sed -n 's/.*"tag_name": *"v\{0,1\}\([^"]*\)".*/\1/p' "$tmp_tag" | head -1)
    rm -f "$tmp_tag"

    [ -n "$version" ] || err "could not determine the latest version"
}

path_contains() {
    case ":${PATH}:" in
        *":$1:"*) return 0 ;;
        *) return 1 ;;
    esac
}

resolve_install_dir() {
    if [ -n "${TONESTACK_INSTALL_DIR:-}" ]; then
        install_dir="$TONESTACK_INSTALL_DIR"
        return
    fi

    for candidate in "$HOME/.local/bin" "/usr/local/bin"; do
        if path_contains "$candidate" && [ -w "$candidate" ]; then
            install_dir="$candidate"
            return
        fi
    done

    install_dir="$HOME/.local/bin"
    mkdir -p "$install_dir"
}

setup_tmp() {
    tmp=$(mktemp -d)
    trap 'rm -rf "$tmp"' EXIT
}

download() {
    base="https://github.com/${REPO}/releases/download/v${version}"
    asset="${APP}_${version}_${os}_${arch}.tar.gz"

    printf "  ${MUTED}version${NC}  %s\n" "$version"
    printf "  ${MUTED}platform${NC} %s/%s\n" "$os" "$arch"

    http_get "$base/$asset" "$tmp/$asset" \
        || err "failed to download $base/$asset"
    http_get "$base/checksums.txt" "$tmp/checksums.txt" \
        || err "failed to download the checksums"
}

verify_checksum() {
    if have sha256sum; then
        actual=$(sha256sum "$tmp/$asset" | awk '{print $1}')
    elif have shasum; then
        actual=$(shasum -a 256 "$tmp/$asset" | awk '{print $1}')
    else
        printf "  ${MUTED}checksum${NC} skipped, no sha256 tool available\n"
        return
    fi

    expected=$(grep " ${asset}\$" "$tmp/checksums.txt" | awk '{print $1}')
    [ -n "$expected" ] || err "no checksum published for $asset"
    [ "$actual" = "$expected" ] || err "checksum mismatch for $asset — refusing to install"

    printf "  ${MUTED}checksum${NC} verified\n"
}

install_binary() {
    tar -xzf "$tmp/$asset" -C "$tmp" || err "could not unpack $asset"
    [ -f "$tmp/$APP" ] || err "$APP is not in the archive"

    # macOS quarantines anything downloaded; strip it so the binary runs.
    if [ "$os" = darwin ] && have xattr; then
        xattr -d com.apple.quarantine "$tmp/$APP" 2>/dev/null || true
    fi

    chmod +x "$tmp/$APP"
    mv "$tmp/$APP" "$install_dir/$APP" \
        || err "could not write to $install_dir — set TONESTACK_INSTALL_DIR to somewhere writable"

    printf "\n${MUTED}▀█▀ █▀█ █▄░█ █▀▀ █▀ ▀█▀ ▄▀█ █▀▀ █▄▀${NC}   ${MUTED}installed to${NC} ${ACCENT}%s${NC}\n" "$install_dir/$APP"
    printf "${ACCENT}░█░ █▄█ █░▀█ ██▄ ▄█ ░█░ █▀█ █▄▄ █░█${NC}   ${MUTED}version${NC} %s\n\n" "$version"
}

report_path() {
    if ! path_contains "$install_dir"; then
        printf "${MUTED}  %s is not on your PATH. Add it:${NC}\n\n" "$install_dir"
        printf "    export PATH=\"%s:\$PATH\"\n\n" "$install_dir"
    fi

    printf "${ACCENT}  %s${NC} ${MUTED}recipes list${NC}\n" "$APP"
    printf "${ACCENT}  %s${NC} ${MUTED}presets make --id mike-dirnt --out mike.hlx${NC}\n\n" "$APP"
}

main() {
    banner
    detect_os
    detect_arch
    resolve_version
    resolve_install_dir
    setup_tmp
    download
    verify_checksum
    install_binary
    report_path
}

main "$@"
