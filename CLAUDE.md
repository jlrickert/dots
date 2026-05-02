# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

**dots** is a Go CLI dotfile package manager with taps, profiles, overlays, and cross-platform support. It treats Git repos as package sources ("taps"), uses path aliases for portability, and supports symlink/copy/hardlink link strategies with platform-aware resolution.

Design specification: `dots-design.md`

## Build & Development Commands

```bash
# Build
go build ./cmd/dots

# Test
go test ./...                                  # all tests
go test ./pkg/dots/... -v                      # single package, verbose
go test ./pkg/dots -run TestAliasResolver      # single test by name
go test -race ./pkg/dots/...                   # with race detector

# Lint
go vet ./...
```

## Architecture

### Package Map

- **`cmd/dots/`** — Binary entrypoint: `toolkit.NewRuntime()` → `cli.Run()`
- **`pkg/cli/`** — Cobra command definitions bridging CLI flags to `pkg/dotsctl`. Each command in its own `cmd_*.go` file.
- **`pkg/dots/`** — Core library: platform detection, path aliases, cascade merge, config/manifest parsing, repository interface, error types, and the artifact subsystem (`Artifact`, `Fetcher`/`HTTPFetcher`/`MemoryFetcher`, `Extractor` with zip-slip / tar-slip hardening). The fetcher and extractor route all FS I/O through `*toolkit.Runtime`; `HTTPFetcher` additionally takes an `*http.Client` because cli-toolkit does not yet expose an HTTP seam.
- **`pkg/dotsctl/`** — Service layer: `Dots` struct composing `PathService`, `ConfigService`, and `Repository`. Each operation in its own `dots_*.go` file (file-per-operation pattern).

### Key Types and Flow

```
CLI command → pkg/cli (Cobra) → pkg/dotsctl.Dots → pkg/dots.Repository
```

**Dots** (`pkg/dotsctl/dots.go`) is the root service struct composing:

- `PathService` — platform-native directory resolution
- `ConfigService` — cached config loading with defaults fallback
- `Repository` — storage backend (MemoryRepo for tests, FsRepo for production)

**Repository** (`pkg/dots/repository.go`) is the storage contract with:

- `MemoryRepo` (`repo_memory.go`) — in-memory, thread-safe, used in tests
- `FsRepo` (`repo_fs.go`) — filesystem-backed (skeleton, not yet implemented)

**Platform** (`pkg/dots/platform.go`) detects OS-arch at runtime and resolves path aliases (`@config`, `@data`, `@home`, `@cache`, `@state`, `@bin`) to platform-native paths.

**AliasResolver** (`pkg/dots/platform.go`) expands alias paths with XDG support on Unix and `%APPDATA%`/`%LOCALAPPDATA%` on Windows. Supports custom user aliases with chaining.

**DeepMerge / ResolvePlatformCascade** (`pkg/dots/platform_cascade.go`) merges config sections in specificity order: base → OS → OS-arch. Maps deep-merge, scalars replace, lists concatenate-dedup.

### File-Per-Operation Pattern

Service operations follow a consistent pattern in `pkg/dotsctl/`:

```
dots.go              # Dots struct, NewDots(), DotsOptions
dots_init.go         # InitOptions + (*Dots).Init()
dots_list.go         # ListOptions + (*Dots).List()
dots_status.go       # StatusResult + (*Dots).Status()
dots_doctor.go       # DoctorCheck + (*Dots).Doctor()
```

Each file contains an `Options` struct and a method on `*Dots`. New operations are added by creating a new `dots_<op>.go` file.

### Config Hierarchy

- User config: `~/.config/dots/config.yaml` (Unix) or `%APPDATA%\dots\config.yaml` (Windows)
- Defaults: `dots.DefaultConfig()` provides sensible fallbacks
- Platform overrides: `config.yaml` `platform:` section applies per-OS/arch overrides

Config is loaded by `ConfigService` in `pkg/dotsctl/config_service.go`.

### Manifest Resolution

`Dotfile.yaml` manifests support platform cascade resolution:

```yaml
links:
  init.lua: "@config/nvim/init.lua" # base
platform:
  darwin:
    links:
      clipboard.lua: "@config/nvim/lua/clipboard.lua" # OS override
  darwin-arm64:
    links:
      bin/nvim-arm: "@bin/nvim-silicon" # OS-arch override
```

`dots.ResolveManifest(manifest, platform)` produces the effective manifest for the current platform.

### Dependency: cli-toolkit

`github.com/jlrickert/cli-toolkit` provides `toolkit.Runtime` — the explicit dependency container carrying filesystem, env, clock, logger, hasher, stream, and process identity. All I/O flows through Runtime, enabling sandboxed test environments.

## Testing

- **Sandbox pattern**: CLI tests use `sandbox.NewSandbox(t, ...)` from cli-toolkit with a jailed temp directory and test runtime.
- **Fixtures**: `pkg/cli/data/` contains embedded test fixtures (config, tap manifests).
- **RunCommand harness**: `RunCommand(t, sb, args...)` in `testhelpers_test.go` runs dots CLI commands and returns stdout/stderr/exitCode.
- **MemoryRepo for speed**: Service layer tests use `dots.NewMemoryRepo()`. Use FsRepo + sandbox only when testing filesystem behavior.
- **Testify**: Uses `github.com/stretchr/testify/require` for assertions.

## Error Handling

- Sentinel errors in `pkg/dots/errors.go`: `ErrNotExist`, `ErrExist`, `ErrParse`, `ErrConflict`, `ErrChecksumMismatch`, `ErrUnsupportedExtract`.
- Typed errors: `TapNotFoundError`, `PackageNotFoundError`, `InvalidConfigError`.
- Check with `errors.Is()` for sentinels, `errors.As()` for typed errors.
- `TapNotFoundError` and `PackageNotFoundError` both satisfy `errors.Is(err, ErrNotExist)`.

## JSON Schemas

- `schemas/dotfile.json` — Draft 7 schema for `Dotfile.yaml` package manifests.
- `schemas/dots-config.json` — Draft 7 schema for `config.yaml` user configuration.
- `schemas/profile.json` — Draft 7 schema for profile definitions in `~/.config/dots/profiles/<name>.yaml`.
- Schema URL constants and modeline strings live in `pkg/dots/config.go`, `pkg/dots/manifest.go`, and `pkg/dots/profile.go`.
- `ConfigService.Save` and `dots init` prepend the `yaml-language-server` modeline to written config files. `MarshalProfile` does the same for profiles, so all profile writers (`ProfileCreate`, `ProfileExport`, internal `saveProfile`) inherit it.
- `DotfileSchemaModeline` is defined but not auto-injected — dots does not currently scaffold `Dotfile.yaml` files. Use it when adding such a command.
- When adding new fields to `Config`, `Manifest`, or `Profile` structs, update the corresponding JSON Schema file to keep them in sync.

## Gotchas

- Path aliases always use forward slashes in manifests. `filepath.ToSlash`/`filepath.FromSlash` handles normalization at resolution time.
- Windows path tests use forward slashes since `filepath.Join` on non-Windows hosts won't produce backslashes.
- The `FsRepo` is a skeleton — filesystem operations are not yet implemented.
- Config falls back to `DefaultConfig()` when no config file exists (not an error).
- Commit conventions: conventional commits (`feat:`, `fix:`, `refactor:`), summaries ≤72 chars. No co-author lines.
