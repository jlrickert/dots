# dots-test:dev

A self-contained, fresh-Containerfile smoke harness for the `dots` CLI.

The image:

1. Builds `dots` from the current source tree (no host binary needed).
2. Boots a clean Ubuntu 24.04 with a non-root `tester` user.
3. Initializes a synthetic fixture tap from `containers/fixture/` and installs
   a handful of fixture packages.
4. Runs `verify.sh` as the entrypoint, which asserts the install pipeline
   produced the expected on-disk shape (links, hooks, platform cascade).

## Usage

```bash
# Build the image (run from repo root — the build context is repo root).
podman build -t dots-test:dev .

# Run the smoke test (entrypoint = verify.sh).
podman run --rm dots-test:dev

# Drop into a shell inside the image to poke around. Override entrypoint.
podman run --rm -it --entrypoint bash dots-test:dev
```

Or via Taskfile:

```bash
task podman:build
task podman:test
task podman:shell
```

`task podman:test` and `task podman:shell` declare `podman:build` as a
dependency, so they will (re)build the image on demand. The `sources:` cache
on `podman:build` skips rebuilds when nothing relevant has changed.

## What gets exercised

The fixture under `containers/fixture/` ships five packages:

| Package          | Exercises                                                                 |
| ---------------- | ------------------------------------------------------------------------- |
| `dots-config`    | Bootstrap package; `link_strategy: copy` set here                         |
| `shell-basic`    | `links:` (file + auto-mode directory), `hooks: post_install`              |
| `platform-aware` | `platform:` cascade (base / linux / darwin)                               |
| `with-overlay`   | `overlay:` block with `@self/` reference                                  |
| `dir-links`      | Object-form `links:`, explicit `mode: symlink` and `mode: copy` + exclude |

`verify.sh` covers:

- `dots --version` exits 0
- `~/.config/dots/config.yaml` from the bootstrap package
- File link + directory cascade under `link_strategy=copy`
- `post_install` hook side effect (marker file under `~/.local/state/dots-test/`)
- Platform cascade: linux block applies, darwin block does not
- `dots list` / `dots status` / `dots which` resolve cleanly
- Directory-mode link entries: explicit `mode: symlink` overrides the
  copy strategy; `mode: copy` with `exclude: [__pycache__, *.pyc]` prunes
  both the segment-matched directory and the suffix-matched leaf
- Tab completion preloaded: `/etc/bash_completion.d/dots` and
  `/usr/share/zsh/vendor-completions/_dots` exist, and interactive bash
  registers a completion for `dots`

## Layout

```
Containerfile          # repo root — multi-stage build
.containerignore       # repo root
containers/
├── README.md          # this file
├── lib.sh             # pass/fail/assert helpers
├── verify.sh          # ENTRYPOINT
└── fixture/           # synthetic dotfiles repo, COPYed to /opt/dotfiles-src
    ├── dots-config/
    ├── shell-basic/
    ├── platform-aware/
    ├── with-overlay/
    └── dir-links/
```

The fixture is checked into the dots repo as plain files. Inside the image,
the Containerfile runs `git init && git add -A && git commit` against
`/opt/dotfiles-src` because `dots init --from file://...` clones from the
**committed** git tree, not the working directory.

## Notes

- The image runs `dots install` as `root` (mirroring the production reference
  image at `jlrickert/dotfiles/docker/ubuntu/Dockerfile`) so any post-install
  hook that wants to use apt or chown system paths works without sudo. The
  fixture's hook does not need root, but the pattern matches.
- After install, `chown -R tester:tester /home/tester` hands ownership back.
- The entrypoint runs as `tester`, not root.
- The `Containerfile` is read by Podman natively. Docker would also read it
  if invoked with `docker build -f Containerfile .`, but the harness is now
  authored against Podman.
- Tab completion for `dots` is preloaded for both bash and zsh. In `task
  podman:shell` (bash) typing `dots <TAB>` enumerates subcommands and `dots
  install <TAB>` enumerates tap/package names. zsh users get the same after
  the build-time `/etc/zsh/zshrc` runs `compinit`.
