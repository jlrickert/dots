# AGENTS.md

Guidance for coding agents working in this repository.

## Overview

**dots** is a Go CLI dotfile package manager with taps, profiles, overlays, and cross-platform support. It treats Git repos as package sources ("taps"), uses path aliases for portability, and supports symlink/copy/hardlink link strategies with platform-aware resolution.

Historical design record: [dots-design.md](dots-design.md)

## Docs First

- `docs/` is the source of truth for dots behavior. Code follows the docs.
- A behavior change starts in `docs/*.md`: update the doc, then change the code to match, then write tests that assert the documented behavior.
- When code and docs disagree, the code is wrong. Change a doc to match code only as a deliberate spec change, and say so in the commit.
- Documented behavior that is not implemented yet goes in the README status table as Partial/Planned — never delete it from the docs to match missing code. Update the row when the gap closes.
- `dots-design.md` is the original design record; where it conflicts with `docs/`, `docs/` wins.
- Full loop and drift checklist: [docs/development/workflow.md](docs/development/workflow.md)

## Workflow

```bash
go build ./cmd/dots                            # build
go test ./...                                  # all tests
go test ./pkg/dots/... -v                      # single package, verbose
go test ./pkg/dots -run TestAliasResolver      # single test by name
go test -race ./pkg/dots/...                   # with race detector
go vet ./...                                   # lint
```

- `task install` installs the CLI; `task podman:build` + `task podman:test` run the container smoke tests.
- Request flow: `pkg/cli` (Cobra) → `pkg/dotsctl.Dots` → `pkg/dots.Repository`.
- New service operation: add `pkg/dotsctl/dots_<op>.go` with an `<Op>Options` struct and a method on `*Dots`. New CLI command: add `pkg/cli/cmd_<name>.go`.
- Commits: conventional commits (`feat:`, `fix:`, `refactor:`, `docs:`), summary ≤72 chars, no co-author lines.

## Critical Rules

- All I/O goes through `*toolkit.Runtime` (cli-toolkit) so tests can sandbox it. `HTTPFetcher` additionally takes an `*http.Client` because cli-toolkit has no HTTP seam yet.
- New or changed fields on `Config`, `Manifest`, or `Profile` go doc → schema → struct: `docs/config-yaml.md` / `docs/dotfile-yaml.md` / `docs/profiles.md`, then `schemas/*.json`, then the Go struct.
- Path aliases always use forward slashes in manifests; Windows path tests also use forward slashes.
- Check errors with `errors.Is()` for sentinels and `errors.As()` for typed errors — never compare strings.
- Service tests use `dots.NewMemoryRepo()`; use FsRepo + `sandbox.NewSandbox` only when testing filesystem behavior. Assert with testify `require`.
- `FsRepo` is a skeleton — filesystem operations are not yet implemented.
- A missing config file is not an error; config falls back to `dots.DefaultConfig()`.

## Where to Look

| Topic                               | Doc                                                                  |
| ----------------------------------- | -------------------------------------------------------------------- |
| Docs-first workflow, drift handling | [docs/development/workflow.md](docs/development/workflow.md)         |
| Package map, key types, data flow   | [docs/development/architecture.md](docs/development/architecture.md) |
| Test harness, fixtures, smoke tests | [docs/development/testing.md](docs/development/testing.md)           |
| Sentinel and typed errors           | [docs/development/errors.md](docs/development/errors.md)             |
| JSON Schemas and modelines          | [docs/development/schemas.md](docs/development/schemas.md)           |
| `Dotfile.yaml` manifest format      | [docs/dotfile-yaml.md](docs/dotfile-yaml.md)                         |
| `config.yaml` user configuration    | [docs/config-yaml.md](docs/config-yaml.md)                           |
| Path aliases                        | [docs/path-aliases.md](docs/path-aliases.md)                         |
| Platform detection and cascade      | [docs/platform-system.md](docs/platform-system.md)                   |
| Link strategies                     | [docs/link-strategies.md](docs/link-strategies.md)                   |
| Profiles                            | [docs/profiles.md](docs/profiles.md)                                 |
| Overlays                            | [docs/overlays.md](docs/overlays.md)                                 |
| Work mode                           | [docs/work-mode.md](docs/work-mode.md)                               |
| CLI commands and flags              | [docs/cli-reference.md](docs/cli-reference.md)                       |
| Bootstrapping a machine             | [docs/bootstrapping.md](docs/bootstrapping.md)                       |
