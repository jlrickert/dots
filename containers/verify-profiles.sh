#!/usr/bin/env bash
# verify-profiles.sh — entrypoint for the dots-test:profiles image. Exercises
# the profile behavior specced in docs/profiles.md against the fixture tap
# under /opt/dotfiles-profiles.
#
# Layout under test (set by the Containerfile):
#   - tap "fixture" registered via `dots init --from file:///opt/dotfiles-profiles --path dots-config`
#   - dots-config links config.yaml plus profiles/{base,work,personal}.yaml
#   - core.always_installed = [fixture/dots-config]
#   - no profile applied yet
#   - link_strategy=copy, so installed files land as regular files
set -euo pipefail

HERE="$(cd "$(dirname "$0")" && pwd)"
# shellcheck source=lib.sh disable=SC1091
. "${HERE}/lib.sh"

cd "${HOME}"

CONFIG="${HOME}/.config"
SHELL_RC="${CONFIG}/shell-basic/rc"
TMUX_RC="${CONFIG}/tmux-ish/rc"
ZELLIJ_RC="${CONFIG}/zellij-ish/rc"

# --- Phase 0: bootstrap state ---
assert_cmd dots
assert_file "${CONFIG}/dots/config.yaml"
assert_file "${CONFIG}/dots/profiles/work.yaml"
assert_cmd_stdout_contains "dots list shows the bootstrap package" "fixture/dots-config" dots list

# --- Phase 1: resolution ---
# always_installed is part of every profile; work excludes the inherited
# tmux-ish and adds zellij-ish.
assert_cmd_stdout_contains "show work: dots-config comes from always_installed" \
    "fixture/dots-config  (always_installed)" dots profile show work
assert_cmd_stdout_contains "show work: shell-basic inherited from base" \
    "fixture/shell-basic  (base)" dots profile show work
assert_cmd_stdout_contains "show work: zellij-ish from work" \
    "fixture/zellij-ish   (work)" dots profile show work
# Scope the check to the Resolved section: tmux-ish legitimately appears
# above it on the "Exclude:" line.
if dots profile show work | sed -n '/^Resolved:/,$p' | grep -qF "fixture/tmux-ish"; then
    fail "show work: excluded tmux-ish should not appear in the resolved list"
else
    pass "show work: excluded tmux-ish is absent from the resolved list"
fi

assert_cmd_stdout_contains "show work: exclude is listed" "Exclude:  fixture/tmux-ish" dots profile show work

# --- Phase 2: apply work ---
assert_cmd_exits_zero "dots profile apply work" dots profile apply work
assert_file "${SHELL_RC}"
assert_file "${ZELLIJ_RC}"
assert_absent "${TMUX_RC}"
assert_file "${CONFIG}/dots/config.yaml"
assert_cmd_stdout_contains "apply work keeps the bootstrap package" "fixture/dots-config" dots list
assert_cmd_stdout_contains "status reports work as active" "work" dots status

# --- Phase 3: dry run changes nothing ---
assert_cmd_stdout_contains "apply personal --dry-run previews the removal" \
    "remove  fixture/zellij-ish" dots profile apply personal --dry-run
assert_file "${ZELLIJ_RC}"
assert_absent "${TMUX_RC}"
assert_cmd_stdout_contains "status still reports work after a dry run" "work" dots status

# --- Phase 4: apply personal reconciles back ---
assert_cmd_exits_zero "dots profile apply personal" dots profile apply personal
assert_file "${TMUX_RC}"
assert_absent "${ZELLIJ_RC}"
assert_file "${SHELL_RC}"
assert_file "${CONFIG}/dots/config.yaml"
assert_cmd_stdout_contains "apply personal keeps the bootstrap package" "fixture/dots-config" dots list
assert_cmd_stdout_contains "status reports personal as active" "personal" dots status

# --- Phase 5: re-applying is a no-op ---
assert_cmd_stdout_contains "re-applying personal changes nothing" \
    "0 removed, 0 installed" dots profile apply personal

summary
