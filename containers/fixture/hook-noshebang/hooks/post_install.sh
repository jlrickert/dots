# hook-noshebang post_install — intentionally has no shebang. dots'
# resolveInterpreter sees no #! line and falls back to resolveShell,
# which uses $SHELL (/bin/bash in this image). The marker carries
# `INTERPRETER=bash` when the fallback wired through to bash; the
# script is POSIX-only so it would still run under dash if $SHELL ever
# pointed there.
set -eu

if [ -n "${BASH_VERSION:-}" ]; then
    interp=bash
else
    interp=dash
fi

MARKER_DIR="${HOME}/markers"
mkdir -p "${MARKER_DIR}"
printf 'hook-noshebang post_install INTERPRETER=%s BASH_VERSION=%s\n' \
    "$interp" "${BASH_VERSION:-}" \
    > "${MARKER_DIR}/hook-noshebang.marker"
