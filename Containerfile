# syntax=docker/dockerfile:1.7
#
# dots-test:dev — fresh-Containerfile smoke harness for the dots CLI.
#
# Stage layout (selected by --build-arg BUILD_FROM=source|deb, default source):
#   builder         — pin go1.25-bookworm, CGO_ENABLED=0 build of ./cmd/dots.
#                     Only consumed by dots-from-source.
#   dots-base       — ubuntu:24.04, apt tooling + locale. Shared by both
#                     variants.
#   dots-from-source — drop the freshly-built binary at /usr/local/bin and
#                     emit completions via `dots completion`.
#   dots-from-deb   — `apt install /tmp/dist/dots_*_${ARCH}.deb`. Exercises
#                     the packaging pipeline produced by `task package:deb`.
#   runtime         — picks dots-from-source or dots-from-deb based on
#                     BUILD_FROM, then layers on the bash/zsh completion
#                     bootstrap, the tester user, the fixture, and runs
#                     dots init/install. Entrypoint = verify.sh.
#
# Build (default, source variant):
#   podman build -t dots-test:dev .
# Build (deb variant — requires `task package:deb` first):
#   podman build --build-arg BUILD_FROM=deb -t dots-test:deb .
# Run:    podman run --rm dots-test:dev
# Debug:  podman run --rm -it --entrypoint bash dots-test:dev
#
# Build context = repo root. The .containerignore at repo root keeps the
# context tight; it makes a narrow exception for /dist/*.deb so the deb
# variant can find its package.

ARG BUILD_FROM=source

# ---------- Stage: build the dots binary (source variant only) ----------
FROM golang:1.25-bookworm AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/dots ./cmd/dots \
    && /out/dots --version

# ---------- Stage: shared runtime base ----------
FROM ubuntu:24.04 AS dots-base

ENV DEBIAN_FRONTEND=noninteractive \
    TZ=America/Chicago

# Apt layer: shells, git, certs, sudo, locale data. tzdata reads $TZ during
# postinst under noninteractive frontend, so /etc/localtime + /etc/timezone
# get configured without dpkg-reconfigure.
RUN apt-get update && apt-get install -y --no-install-recommends \
        bash \
        bash-completion \
        ca-certificates \
        git \
        locales \
        python3 \
        sudo \
        tzdata \
        zsh \
    && rm -rf /var/lib/apt/lists/* \
    && locale-gen en_US.UTF-8

ENV LANG=en_US.UTF-8 \
    LC_ALL=en_US.UTF-8

# ---------- Stage: source-built dots ----------
# Drops the binary from the builder stage at /usr/local/bin/dots and emits
# completions by invoking `dots completion <shell>`. Mirrors what the
# Homebrew formula does.
FROM dots-base AS dots-from-source

COPY --from=builder /out/dots /usr/local/bin/dots
RUN dots --version \
    && mkdir -p /usr/share/zsh/vendor-completions /usr/share/bash-completion/completions \
    && dots completion bash > /usr/share/bash-completion/completions/dots \
    && dots completion zsh > /usr/share/zsh/vendor-completions/_dots \
    && chmod 0644 /usr/share/bash-completion/completions/dots /usr/share/zsh/vendor-completions/_dots

# ---------- Stage: .deb-installed dots ----------
# Installs the .deb produced by `task package:deb`. The .deb itself drops the
# binary at /usr/bin/dots and ships completions + man pages — no manual
# completion emission needed.
FROM dots-base AS dots-from-deb

# .containerignore excludes /dist but negates /dist/*.deb so this COPY only
# brings in the .deb artifacts, not the rest of the goreleaser output.
COPY dist/ /tmp/dist/
RUN ARCH="$(dpkg --print-architecture)" \
    && apt-get update \
    && apt-get install -y --no-install-recommends /tmp/dist/dots_*_${ARCH}.deb \
    && rm -rf /var/lib/apt/lists/* /tmp/dist \
    && dots --version

# ---------- Stage: runtime ----------
# Final image. BUILD_FROM picks which intermediate provides the dots binary
# and completion scripts; everything below this point is identical across
# variants.
FROM dots-from-${BUILD_FROM} AS runtime

# bash: append an uncommented bash-completion bootstrap. Ubuntu 24.04 ships
# /etc/bash.bashrc with the equivalent block commented out — the package
# relies on /etc/profile.d/bash_completion.sh, which only runs for *login*
# shells. Interactive non-login bash (`bash -i`, `podman run -it`) never
# sources it.
#
# zsh: write a minimal /etc/zsh/zshrc that runs compinit so the completion
# scripts on $fpath are picked up. -u suppresses insecure-directory warnings
# for the smoke harness.
RUN printf '\n%s\n' \
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
    && HOME=/home/tester dots install fixture/hook-sh \
    && HOME=/home/tester dots install fixture/hook-env-bash \
    && HOME=/home/tester dots install fixture/hook-env-python3 \
    && HOME=/home/tester dots install fixture/hook-noshebang \
    && chown -R tester:tester /home/tester

# Verify harness lives outside the user's HOME so it isn't shadowed by any
# package that links into ~/.local/lib.
COPY containers/lib.sh containers/verify.sh /usr/local/lib/dots-test/
RUN chmod +x /usr/local/lib/dots-test/verify.sh /usr/local/lib/dots-test/lib.sh

USER tester
WORKDIR /home/tester
ENTRYPOINT ["/usr/local/lib/dots-test/verify.sh"]
