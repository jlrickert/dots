# syntax=docker/dockerfile:1.7
#
# dots-test:dev — fresh-Containerfile smoke harness for the dots CLI.
#
# Stage layout:
#   1. builder       — pin go1.25-bookworm, CGO_ENABLED=0 build of ./cmd/dots.
#   2. runtime       — ubuntu:24.04 base, locale, apt tooling, dots binary,
#                      tester user, fixture COPY + snapshot commit, init/install,
#                      verify entrypoint.
#
# Build:  podman build -t dots-test:dev .
# Run:    podman run --rm dots-test:dev
# Debug:  podman run --rm -it --entrypoint bash dots-test:dev
#
# Build context = repo root. The .containerignore at repo root keeps the
# context tight. The `# syntax=` directive above is a BuildKit comment and is
# ignored by buildah/podman — leaving it in keeps the file portable across
# engines.

# ---------- Stage 1: build the dots binary ----------
FROM golang:1.25-bookworm AS builder

WORKDIR /src

# Cache go module downloads on go.sum changes.
COPY go.mod go.sum ./
RUN go mod download

# Bring in the rest of the source tree.
COPY . .

# CGO_ENABLED=0 produces a static binary that runs on the slim ubuntu:24.04
# runtime stage without needing libc symbols beyond what the base ships.
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/dots ./cmd/dots \
    && /out/dots --version

# ---------- Stage 2: runtime image ----------
FROM ubuntu:24.04

ENV DEBIAN_FRONTEND=noninteractive \
    TZ=America/Chicago

# Apt layer: shells, git, certs, sudo, locale data. No build-essential — the
# binary is built in the builder stage and copied in. tzdata reads $TZ during
# postinst under noninteractive frontend, so /etc/localtime + /etc/timezone
# get configured without dpkg-reconfigure.
RUN apt-get update && apt-get install -y --no-install-recommends \
        bash \
        bash-completion \
        ca-certificates \
        git \
        locales \
        sudo \
        tzdata \
        zsh \
    && rm -rf /var/lib/apt/lists/* \
    && locale-gen en_US.UTF-8

ENV LANG=en_US.UTF-8 \
    LC_ALL=en_US.UTF-8

# Pull in the binary built in stage 1.
COPY --from=builder /out/dots /usr/local/bin/dots
RUN dots --version

# Preload tab completion for both shells.
#
# bash: drop the Cobra-generated script in /etc/bash_completion.d/. Ubuntu
# 24.04 ships /etc/bash.bashrc with the bash-completion bootstrap block
# COMMENTED OUT — the package now relies on /etc/profile.d/bash_completion.sh,
# which only runs for *login* shells. Interactive non-login bash (`bash -i`,
# `podman run -it --entrypoint bash`) never sources it. We append an
# uncommented copy so interactive bash, login or not, picks up the
# completion engine and then autoloads /etc/bash_completion.d/dots.
#
# zsh: drop _dots into /usr/share/zsh/vendor-completions/ — this is on the
# default $fpath (`zsh -c 'echo $fpath'`) but Ubuntu's zsh-common does not
# pre-create it, so mkdir -p first. Ubuntu ships no default /etc/zsh/zshrc,
# so write a minimal one that runs compinit (with -u so insecure-directory
# warnings don't fail interactive sessions — fine for a smoke harness).
RUN mkdir -p /usr/share/zsh/vendor-completions \
    && dots completion bash > /etc/bash_completion.d/dots \
    && dots completion zsh > /usr/share/zsh/vendor-completions/_dots \
    && chmod 0644 /etc/bash_completion.d/dots /usr/share/zsh/vendor-completions/_dots \
    && printf '\n%s\n' \
        '# Enable bash-completion in interactive shells. Ubuntu 24.04 ships' \
        '# the equivalent block commented out and relies on a profile.d hook' \
        '# that only runs for login shells, so interactive non-login bash' \
        '# (the default for `podman run -it --entrypoint bash`) never loads' \
        '# completion. Added by dots-test:dev to fix that.' \
        'if ! shopt -oq posix; then' \
        '    if [ -f /usr/share/bash-completion/bash_completion ]; then' \
        '        . /usr/share/bash-completion/bash_completion' \
        '    fi' \
        'fi' \
        >> /etc/bash.bashrc \
    && printf '%s\n' \
        '# Activate completion for interactive zsh.' \
        'autoload -Uz compinit' \
        'compinit -u' \
        > /etc/zsh/zshrc

# Create the test user with passwordless sudo. The fixture is COPYed with
# matching ownership so the snapshot commit (run as tester) doesn't trip
# git's safe.directory check.
RUN useradd --create-home --shell /usr/bin/zsh tester \
    && echo 'tester ALL=(ALL) NOPASSWD:ALL' > /etc/sudoers.d/tester \
    && chmod 0440 /etc/sudoers.d/tester

# Ship the synthetic fixture under /opt/dotfiles-src. Owned by tester so the
# snapshot commit (next RUN, run as tester) doesn't trip git's safe.directory.
COPY --chown=tester:tester containers/fixture /opt/dotfiles-src

# `dots init --from file://` clones from the COMMITTED git tree, not the
# working directory. The fixture under containers/fixture/ is checked into the
# dots repo as plain files, so this stage initializes a one-commit repo
# inside the image to satisfy that contract. This is the dev/1091 workaround
# (snapshot-commit pattern).
USER tester
ENV SHELL=/bin/bash \
    HOME=/home/tester
RUN cd /opt/dotfiles-src \
    && git init -q -b main \
    && git config user.email "build@dots-test.local" \
    && git config user.name "dots-test build" \
    && git add -A \
    && git commit -q -m "fixture snapshot"

# Install as root so any post_install hooks that may want to touch system
# state (apt, etc.) work. None of the synthetic fixture's hooks do, but the
# pattern matches the production reference image. The safe.directory glob is
# required because the fixture worktree is owned by uid 1000 (tester) while
# we're now running as root.
USER root
RUN git config --system --add safe.directory '*' \
    && HOME=/home/tester dots init \
        --from "file:///opt/dotfiles-src" \
        --path dots-config \
        --name fixture \
    && HOME=/home/tester dots install fixture/shell-basic \
    && HOME=/home/tester dots install fixture/platform-aware \
    && HOME=/home/tester dots install fixture/with-overlay \
    && HOME=/home/tester dots install fixture/dir-links \
    && chown -R tester:tester /home/tester

# Verify harness lives outside the user's HOME so it isn't shadowed by any
# package that links into ~/.local/lib.
COPY containers/lib.sh containers/verify.sh /usr/local/lib/dots-test/
RUN chmod +x /usr/local/lib/dots-test/verify.sh /usr/local/lib/dots-test/lib.sh

USER tester
WORKDIR /home/tester
ENTRYPOINT ["/usr/local/lib/dots-test/verify.sh"]
