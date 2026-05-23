#!/usr/bin/env bash
# hook-env-bash post_install — writes a marker whose contents prove bash
# ran via the /usr/bin/env shim. Tag the marker with `INTERPRETER=bash`
# for the verify.sh fixed-string grep, and carry $BASH_VERSION for
# diagnostic value if a failure shows up.
set -euo pipefail

if [ -n "${BASH_VERSION:-}" ]; then
    interp=bash
else
    interp=dash
fi

MARKER_DIR="${HOME}/markers"
mkdir -p "${MARKER_DIR}"
printf 'hook-env-bash post_install INTERPRETER=%s BASH_VERSION=%s\n' \
    "$interp" "${BASH_VERSION:-}" \
    > "${MARKER_DIR}/hook-env-bash.marker"
