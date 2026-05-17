#!/usr/bin/env bash
# install.sh — bootstrap installer for jlrickert/dots.
#
# Usage:
#   curl -fsSL https://github.com/jlrickert/dots/releases/latest/download/install.sh | bash
#   curl -fsSL .../install.sh | bash -s -- --version v0.7.0
#   curl -fsSL .../install.sh | bash -s -- --prefix /opt/dots/bin
#   curl -fsSL .../install.sh | bash -s -- --uninstall
#
# Env overrides (flags win):
#   DOTS_VERSION       Pin a version (e.g. v0.7.0). Default: latest.
#   DOTS_INSTALL_DIR   Install directory. Default: $HOME/.local/bin.

set -euo pipefail

REPO_OWNER="jlrickert"
REPO_NAME="dots"
REPO_SLUG="${REPO_OWNER}/${REPO_NAME}"
BIN_NAME="dots"
DEFAULT_PREFIX="${HOME}/.local/bin"

_is_tty() { [ -t 2 ]; }
if _is_tty; then
    C_RED=$'\033[31m'
    C_YEL=$'\033[33m'
    C_GRN=$'\033[32m'
    C_RST=$'\033[0m'
else
    C_RED=""
    C_YEL=""
    C_GRN=""
    C_RST=""
fi

info() { printf '%sinfo:%s %s\n'  "$C_GRN" "$C_RST" "$*" >&2; }
warn() { printf '%swarn:%s %s\n'  "$C_YEL" "$C_RST" "$*" >&2; }
err()  { printf '%serror:%s %s\n' "$C_RED" "$C_RST" "$*" >&2; }
die()  { err "$*"; exit 1; }

need_cmd() {
    command -v "$1" >/dev/null 2>&1 || die "required command not found: $1"
}

detect_os() {
    local os
    os="$(uname -s)"
    case "$os" in
        Linux)  echo linux ;;
        Darwin) echo darwin ;;
        MINGW*|MSYS*|CYGWIN*|Windows_NT)
            die "Windows is not supported by this installer. See https://github.com/${REPO_SLUG}#installation for alternatives." ;;
        *) die "unsupported OS: $os" ;;
    esac
}

detect_arch() {
    local arch
    arch="$(uname -m)"
    case "$arch" in
        x86_64|amd64)  echo amd64 ;;
        aarch64|arm64) echo arm64 ;;
        *) die "unsupported architecture: $arch (supported: amd64, arm64)" ;;
    esac
}

ACTION="install"
VERSION="${DOTS_VERSION:-}"
PREFIX="${DOTS_INSTALL_DIR:-${DEFAULT_PREFIX}}"

print_help() {
    cat <<EOF
install.sh — installer for ${REPO_SLUG}

USAGE:
  install.sh [--version vX.Y.Z] [--prefix DIR]
  install.sh --uninstall [--prefix DIR]

OPTIONS:
  --version V    Install version V (default: latest release)
  --prefix DIR   Install into DIR (default: \$HOME/.local/bin)
  --uninstall    Remove dots from --prefix
  -h, --help     Show this help

ENV:
  DOTS_VERSION       Same as --version
  DOTS_INSTALL_DIR   Same as --prefix
EOF
}

while [ $# -gt 0 ]; do
    case "$1" in
        --uninstall)  ACTION="uninstall"; shift ;;
        --version)    [ $# -ge 2 ] || die "--version requires an argument"; VERSION="$2"; shift 2 ;;
        --version=*)  VERSION="${1#*=}"; shift ;;
        --prefix)     [ $# -ge 2 ] || die "--prefix requires an argument";  PREFIX="$2"; shift 2 ;;
        --prefix=*)   PREFIX="${1#*=}"; shift ;;
        -h|--help)    print_help; exit 0 ;;
        *) die "unknown argument: $1 (try --help)" ;;
    esac
done

have_curl() { command -v curl >/dev/null 2>&1; }
have_wget() { command -v wget >/dev/null 2>&1; }

fetch_to() {
    local url="$1" out="$2"
    if have_curl; then
        curl -fsSL --proto '=https' --tlsv1.2 -o "$out" "$url"
    elif have_wget; then
        wget -q -O "$out" "$url"
    else
        die "neither curl nor wget found on PATH"
    fi
}

# Resolve the latest tag by following the redirect on /releases/latest. Avoids
# the GitHub API entirely so we don't hit unauthenticated rate limits.
resolve_latest() {
    local url effective tag
    url="https://github.com/${REPO_SLUG}/releases/latest"
    if have_curl; then
        effective="$(curl -fsSLI -o /dev/null -w '%{url_effective}' "$url")" \
            || die "failed to resolve latest release"
    elif have_wget; then
        effective="$(wget --max-redirect=0 -q -S -O /dev/null "$url" 2>&1 \
            | awk '/^  Location: /{print $2}' | tail -1)"
        [ -n "$effective" ] || die "failed to resolve latest release"
    else
        die "neither curl nor wget found on PATH"
    fi
    tag="${effective##*/tag/}"
    [ -n "$tag" ] && [ "$tag" != "$effective" ] \
        || die "could not parse latest tag from: $effective"
    printf '%s\n' "$tag"
}

sha256_of() {
    local f="$1"
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum "$f" | awk '{print $1}'
    elif command -v shasum >/dev/null 2>&1; then
        shasum -a 256 "$f" | awk '{print $1}'
    else
        die "no sha256 tool found (need sha256sum or shasum)"
    fi
}

# Leave the tmpdir intact on checksum failure — the forensic value (MITM,
# stale CDN copy, re-uploaded release asset) outweighs the cleanup hygiene.
verify_checksum() {
    local archive_path="$1" archive_name="$2" sums_path="$3"
    local expected actual
    expected="$(awk -v n="$archive_name" '$2==n || $2=="*"n {print $1}' "$sums_path" | head -1)"
    [ -n "$expected" ] || die "no checksum entry for $archive_name in checksums.txt"
    actual="$(sha256_of "$archive_path")"
    if [ "$expected" != "$actual" ]; then
        err "checksum mismatch for $archive_name"
        err "  expected: $expected"
        err "  actual:   $actual"
        die "aborting install (tmpdir preserved at: $TMPDIR_KEEP)"
    fi
    info "checksum OK ($archive_name)"
}

path_hint() {
    case ":$PATH:" in
        *":${PREFIX}:"*) return 0 ;;
    esac
    local shell_name rcfile line
    shell_name="$(basename "${SHELL:-/bin/sh}")"
    line="export PATH=\"${PREFIX}:\$PATH\""
    # rcfile is display-only text shown to the user; the tilde is intentional.
    # shellcheck disable=SC2088
    case "$shell_name" in
        bash) rcfile='~/.bashrc' ;;
        zsh)  rcfile='~/.zshrc' ;;
        fish)
            rcfile='~/.config/fish/config.fish'
            line="fish_add_path ${PREFIX}"
            ;;
        *) rcfile="your shell rc file" ;;
    esac
    warn "${PREFIX} is not on \$PATH."
    warn "Add this to ${rcfile} to fix it:"
    printf '\n    %s\n\n' "$line" >&2
}

do_uninstall() {
    local target="${PREFIX}/${BIN_NAME}"
    if [ -e "$target" ] || [ -L "$target" ]; then
        rm -f -- "$target"
        info "removed $target"
    else
        info "nothing to remove at $target"
    fi
    exit 0
}

TMPDIR_KEEP=""
INSTALL_OK=0
cleanup() {
    if [ "$INSTALL_OK" = "1" ] && [ -n "$TMPDIR_KEEP" ] && [ -d "$TMPDIR_KEEP" ]; then
        rm -rf -- "$TMPDIR_KEEP"
    elif [ -n "$TMPDIR_KEEP" ] && [ -d "$TMPDIR_KEEP" ]; then
        warn "leaving tmpdir for inspection: $TMPDIR_KEEP"
    fi
}
trap cleanup EXIT

do_install() {
    need_cmd uname
    need_cmd tar
    need_cmd mkdir
    need_cmd mv
    need_cmd rm
    need_cmd awk
    have_curl || have_wget || die "neither curl nor wget found on PATH"

    local os arch version
    os="$(detect_os)"
    arch="$(detect_arch)"

    if [ -n "$VERSION" ]; then
        version="$VERSION"
        case "$version" in v*) ;; *) version="v${version}" ;; esac
    else
        info "resolving latest release..."
        version="$(resolve_latest)"
    fi
    info "installing ${BIN_NAME} ${version} (${os}/${arch}) -> ${PREFIX}"

    local version_no_v="${version#v}"
    local archive_name="dots_${version_no_v}_${os}_${arch}.tar.gz"
    local base_url="https://github.com/${REPO_SLUG}/releases/download/${version}"
    local archive_url="${base_url}/${archive_name}"
    local sums_url="${base_url}/checksums.txt"

    TMPDIR_KEEP="$(mktemp -d 2>/dev/null || mktemp -d -t dots-install)"
    local archive_path="${TMPDIR_KEEP}/${archive_name}"
    local sums_path="${TMPDIR_KEEP}/checksums.txt"

    info "downloading ${archive_url}"
    fetch_to "$archive_url" "$archive_path" \
        || die "download failed — does ${version} exist? URL: ${archive_url}"

    info "downloading ${sums_url}"
    fetch_to "$sums_url" "$sums_path" \
        || die "download failed for checksums.txt at ${sums_url}"

    verify_checksum "$archive_path" "$archive_name" "$sums_path"

    info "extracting..."
    tar -xzf "$archive_path" -C "$TMPDIR_KEEP"

    local extracted="${TMPDIR_KEEP}/${BIN_NAME}"
    [ -f "$extracted" ] || die "archive did not contain ${BIN_NAME} binary"

    mkdir -p "$PREFIX" || die "failed to create install dir: $PREFIX"
    chmod +x "$extracted"
    mv -f -- "$extracted" "${PREFIX}/${BIN_NAME}"

    info "installed: ${PREFIX}/${BIN_NAME}"
    path_hint
    info "run: ${BIN_NAME} --version"
    INSTALL_OK=1
}

case "$ACTION" in
    install)   do_install ;;
    uninstall) do_uninstall ;;
esac
