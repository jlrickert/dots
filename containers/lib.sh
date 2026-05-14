#!/usr/bin/env bash
# Shared assertion helpers for the dots-test:dev verify entrypoint.
# Adapted from jlrickert/dotfiles tests/e2e/lib.sh.

PASS_COUNT=0
FAIL_COUNT=0

pass() {
    PASS_COUNT=$((PASS_COUNT + 1))
    printf 'PASS: %s\n' "$1"
}

fail() {
    FAIL_COUNT=$((FAIL_COUNT + 1))
    printf 'FAIL: %s\n' "$1" >&2
}

assert_link() {
    local path="$1"
    if [ ! -L "$path" ]; then
        fail "$path is not a symlink"
        return
    fi
    if [ ! -e "$path" ]; then
        fail "$path -> $(readlink "$path") (target missing)"
        return
    fi
    pass "$path -> $(readlink "$path")"
}

assert_file() {
    local path="$1"
    if [ ! -f "$path" ]; then
        fail "$path is not a regular file"
        return
    fi
    pass "$path is a regular file"
}

assert_dir() {
    local path="$1"
    if [ ! -d "$path" ]; then
        fail "$path is not a directory"
        return
    fi
    pass "$path is a directory"
}

assert_absent() {
    local path="$1"
    if [ -e "$path" ] || [ -L "$path" ]; then
        fail "$path exists but should not"
        return
    fi
    pass "$path absent"
}

assert_grep() {
    local path="$1" needle="$2"
    if [ ! -f "$path" ]; then
        fail "$path missing (looking for '$needle')"
        return
    fi
    if grep -qF "$needle" "$path"; then
        pass "$path contains '$needle'"
    else
        fail "$path missing '$needle'"
    fi
}

assert_cmd() {
    local cmd="$1"
    if command -v "$cmd" >/dev/null 2>&1; then
        pass "$cmd available at $(command -v "$cmd")"
    else
        fail "$cmd not on PATH"
    fi
}

assert_cmd_exits_zero() {
    local label="$1"
    shift
    if "$@" >/dev/null 2>&1; then
        pass "$label"
    else
        fail "$label (command failed: $*)"
    fi
}

# Run a command and assert its stdout contains a substring.
assert_cmd_stdout_contains() {
    local label="$1" needle="$2"
    shift 2
    local out
    if ! out="$("$@" 2>/dev/null)"; then
        fail "$label (command failed: $*)"
        return
    fi
    if printf '%s\n' "$out" | grep -qF "$needle"; then
        pass "$label"
    else
        fail "$label (stdout missing '$needle')"
    fi
}

summary() {
    printf '\n%d passed, %d failed\n' "$PASS_COUNT" "$FAIL_COUNT"
    [ "$FAIL_COUNT" -eq 0 ]
}
