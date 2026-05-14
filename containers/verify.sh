#!/usr/bin/env bash
# verify.sh — entrypoint for the dots-test:dev image. Exercises a focused
# slice of the dots install pipeline against the synthetic fixture under
# /opt/dotfiles-src.
#
# Layout under test (set by the Containerfile):
#   - tap "fixture" registered via `dots init --from file:///opt/dotfiles-src --path dots-config`
#   - packages installed (in order): fixture/shell-basic, fixture/platform-aware, fixture/with-overlay
#   - link_strategy=copy (set in dots-config/config.yaml), so files land as
#     regular files (not symlinks).
set -euo pipefail

HERE="$(cd "$(dirname "$0")" && pwd)"
# shellcheck source=lib.sh disable=SC1091
. "${HERE}/lib.sh"

cd "${HOME}"

# --- Phase 0: dots binary itself ---
assert_cmd dots
assert_cmd_exits_zero "dots --version exits 0" dots --version

# --- Phase 1: dots-config bootstrap ---
# dots init --path dots-config --name fixture should have copied
# dots-config/config.yaml to ~/.config/dots/config.yaml.
assert_file "${HOME}/.config/dots/config.yaml"
assert_grep "${HOME}/.config/dots/config.yaml" "link_strategy: copy"

# --- Phase 2: shell-basic — file link, directory cascade, copy strategy ---
# bashrc is a regular file (link_strategy=copy → real bytes, not a symlink).
assert_file "${HOME}/.bashrc"
if [ -L "${HOME}/.bashrc" ]; then
    fail "${HOME}/.bashrc is a symlink (expected regular file under copy strategy)"
else
    pass "${HOME}/.bashrc is not a symlink (copy strategy)"
fi
assert_grep "${HOME}/.bashrc" "DOTS_TEST_SHELL_BASIC=1"

# Directory link with copy strategy resolves to per-leaf copies under the dest.
# shell-basic/profile.d/10-greeting.sh → ~/.config/profile.d/10-greeting.sh
assert_dir "${HOME}/.config/profile.d"
assert_file "${HOME}/.config/profile.d/10-greeting.sh"
assert_grep "${HOME}/.config/profile.d/10-greeting.sh" "DOTS_TEST_GREETING"

# --- Phase 3: post_install hook side effect ---
# shell-basic/hooks/post_install.sh writes a marker; presence proves the
# hook ran.
assert_file "${HOME}/.local/state/dots-test/post_install.marker"
assert_grep "${HOME}/.local/state/dots-test/post_install.marker" "shell-basic post_install ran"

# --- Phase 4: platform-aware cascade (linux block applies, darwin does not) ---
assert_file "${HOME}/.config/dots-test/platform-aware/base.txt"
assert_file "${HOME}/.config/dots-test/platform-aware/linux.txt"
assert_absent "${HOME}/.config/dots-test/platform-aware/darwin.txt"

# --- Phase 5: with-overlay parsed and installed ---
assert_file "${HOME}/.config/dots-test/with-overlay/marker.txt"

# --- Phase 5b: dir-links — explicit directory modes + exclude globs ---
# lua/ is declared with object-form `mode: symlink`. Under dots-config's
# copy strategy auto-mode would emit per-leaf copies; the explicit mode
# must win, so the dest is a single symlink whose target resolves to the
# source under the tap.
assert_link "${HOME}/.config/dir-links/lua"
assert_file "${HOME}/.config/dir-links/lua/init.lua"
assert_grep "${HOME}/.config/dir-links/lua/init.lua" "DOTS_TEST_DIR_LINKS_LUA_INIT"
assert_file "${HOME}/.config/dir-links/lua/util.lua"

# factorizers/ is declared with `mode: copy` and `exclude: [__pycache__,
# *.pyc]`. The dest dir is a real directory (not a symlink), regular files
# land as copies, and the two exclude globs prune their targets.
assert_dir "${HOME}/.config/dir-links/factorizers"
if [ -L "${HOME}/.config/dir-links/factorizers" ]; then
    fail "${HOME}/.config/dir-links/factorizers is a symlink (expected real dir under mode: copy)"
else
    pass "${HOME}/.config/dir-links/factorizers is not a symlink (mode: copy)"
fi
assert_file "${HOME}/.config/dir-links/factorizers/factor.py"
assert_grep "${HOME}/.config/dir-links/factorizers/factor.py" "DOTS_TEST_DIR_LINKS_FACTOR"
assert_file "${HOME}/.config/dir-links/factorizers/helper.py"
# Segment-match: __pycache__ is one path segment in the source tree; the
# exclude rule should prune the entire directory.
assert_absent "${HOME}/.config/dir-links/factorizers/__pycache__"
# Suffix-match: orphan.pyc lives at the top level of factorizers/; the
# *.pyc glob should match its leaf name.
assert_absent "${HOME}/.config/dir-links/factorizers/orphan.pyc"

# --- Phase 6: dots list / dots status / dots which ---
assert_cmd_exits_zero "dots list exits 0" dots list
assert_cmd_exits_zero "dots status exits 0" dots status

assert_cmd_stdout_contains "dots list mentions fixture/shell-basic" \
    "fixture/shell-basic" dots list
assert_cmd_stdout_contains "dots list mentions fixture/platform-aware" \
    "fixture/platform-aware" dots list
assert_cmd_stdout_contains "dots list mentions fixture/with-overlay" \
    "fixture/with-overlay" dots list
assert_cmd_stdout_contains "dots list mentions fixture/dir-links" \
    "fixture/dir-links" dots list

# `dots which` takes a destination path and reports the package that placed
# it. The bashrc landed at ~/.bashrc via shell-basic, so this should resolve.
assert_cmd_stdout_contains "dots which ~/.bashrc resolves to fixture/shell-basic" \
    "fixture/shell-basic" dots which "${HOME}/.bashrc"

# --- Phase 7: shell completion preload ---
# The Containerfile drops Cobra-generated completion into the standard
# system locations. The first three assertions confirm shape; the last
# spawns an interactive bash (-i) so /etc/bash.bashrc sources the
# bash-completion engine, which in turn autoloads /etc/bash_completion.d/dots.
# If any link in that chain breaks (apt package missing, file mode wrong,
# Ubuntu's bashrc not auto-sourcing completions), `complete -p dots` exits
# non-zero and this assertion fails loudly.
assert_file /etc/bash_completion.d/dots
assert_grep /etc/bash_completion.d/dots "__start_dots"
assert_file /usr/share/zsh/vendor-completions/_dots
assert_cmd_exits_zero "interactive bash auto-loads dots completion" \
    bash -ic "complete -p dots >/dev/null"
# Symmetric zsh check: prove /etc/zsh/zshrc runs compinit and that _dots is
# discoverable on $fpath. `${(k)_comps[dots]}` prints "dots" when the
# completion is registered; empty otherwise.
assert_cmd_stdout_contains "interactive zsh registers dots completion" \
    "dots" zsh -ic "print -l \${(k)_comps[dots]}"

summary
