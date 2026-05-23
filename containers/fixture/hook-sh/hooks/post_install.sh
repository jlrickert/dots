#!/bin/sh
# hook-sh post_install — writes a marker whose contents prove dash ran.
# $BASH_VERSION is a bash-only variable; dash leaves it unset, so the
# absence of that variable is the dispatch witness. The marker carries
# a stable `INTERPRETER=dash` tag for the verify.sh fixed-string grep,
# plus the raw $BASH_VERSION for human debugging if a failure shows up.
set -eu

if [ -n "${BASH_VERSION:-}" ]; then
    interp=bash
else
    interp=dash
fi

MARKER_DIR="${HOME}/markers"
mkdir -p "${MARKER_DIR}"
printf 'hook-sh post_install INTERPRETER=%s BASH_VERSION=%s\n' \
    "$interp" "${BASH_VERSION:-}" \
    > "${MARKER_DIR}/hook-sh.marker"
