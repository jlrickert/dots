#!/usr/bin/env bash
# shell-basic post_install hook — writes a marker file so verify.sh can
# confirm the hook actually ran. The marker lives under ~/.local/state so it
# stays out of the user's config tree.
set -euo pipefail

MARKER_DIR="${HOME}/.local/state/dots-test"
mkdir -p "${MARKER_DIR}"
printf 'shell-basic post_install ran at %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    > "${MARKER_DIR}/post_install.marker"
