#!/usr/bin/env bash
# entry-live.sh — entrypoint for the interactive `task shell*` targets.
#
# The shell tasks mount the host's ~/.ssh read-only at /mnt/ssh. Under
# rootless podman those files appear owned by root inside the container while
# we run as `tester`, and ssh refuses to use a private key it does not own.
# Copying the mount into $HOME/.ssh fixes ownership and leaves known_hosts
# writable. Nothing is ever written back to the host.
set -euo pipefail

if [ -d /mnt/ssh ] && [ ! -d "${HOME}/.ssh" ]; then
    sudo cp -r /mnt/ssh "${HOME}/.ssh"
    sudo chown -R "$(id -u):$(id -g)" "${HOME}/.ssh"
    chmod 700 "${HOME}/.ssh"
    find "${HOME}/.ssh" -maxdepth 1 -type f ! -name '*.pub' -exec chmod 600 {} +
fi

if [ "$#" -eq 0 ]; then
    set -- bash -l
fi
exec "$@"
