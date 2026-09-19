# Architecture

Internal structure of the dots codebase. Behavior is defined by the user docs in [docs/](../); this page describes how the code implements them. [dots-design.md](../../dots-design.md) is the original design record.

## Package Map

- **`cmd/dots/`** — Binary entrypoint: `toolkit.NewRuntime()` → `cli.Run()`
- **`pkg/cli/`** — Cobra command definitions bridging CLI flags to `pkg/dotsctl`. Each command in its own `cmd_*.go` file.
- **`pkg/dots/`** — Core library: platform detection, path aliases, cascade merge, config/manifest parsing, repository interface, error types, and the artifact subsystem (`Artifact`, `Fetcher`/`HTTPFetcher`/`MemoryFetcher`, `Extractor` with zip-slip / tar-slip hardening). The fetcher and extractor route all FS I/O through `*toolkit.Runtime`; `HTTPFetcher` additionally takes an `*http.Client` because cli-toolkit does not yet expose an HTTP seam.
- **`pkg/dotsctl/`** — Service layer: `Dots` struct composing `PathService`, `ConfigService`, and `Repository`. Each operation in its own `dots_*.go` file (file-per-operation pattern).

## Key Types and Flow

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

**AliasResolver** (`pkg/dots/platform.go`) expands alias paths with XDG support on Unix and `%APPDATA%`/`%LOCALAPPDATA%` on Windows. Supports custom user aliases with chaining. See [Path Aliases](../path-aliases.md).

**DeepMerge / ResolvePlatformCascade** (`pkg/dots/platform_cascade.go`) merges config sections in specificity order: base → OS → OS-arch. Maps deep-merge, scalars replace, lists concatenate-dedup. See [Platform System](../platform-system.md).

## File-Per-Operation Pattern

Service operations follow a consistent pattern in `pkg/dotsctl/`:

```
dots.go              # Dots struct, NewDots(), DotsOptions
dots_init.go         # InitOptions + (*Dots).Init()
dots_list.go         # ListOptions + (*Dots).List()
dots_status.go       # StatusResult + (*Dots).Status()
dots_doctor.go       # DoctorCheck + (*Dots).Doctor()
```

Each file contains an `Options` struct and a method on `*Dots`. New operations are added by creating a new `dots_<op>.go` file.

## Config Hierarchy

- User config: `~/.config/dots/config.yaml` (Unix) or `%APPDATA%\dots\config.yaml` (Windows)
- Defaults: `dots.DefaultConfig()` provides sensible fallbacks; a missing config file is not an error.
- Platform overrides: `config.yaml` `platform:` section applies per-OS/arch overrides

Config is loaded by `ConfigService` in `pkg/dotsctl/config_service.go`. See [config.yaml Reference](../config-yaml.md).

## Manifest Resolution

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

`dots.ResolveManifest(manifest, platform)` produces the effective manifest for the current platform. See [Dotfile.yaml Reference](../dotfile-yaml.md).

## Dependency: cli-toolkit

`github.com/jlrickert/cli-toolkit` provides `toolkit.Runtime` — the explicit dependency container carrying filesystem, env, clock, logger, hasher, stream, and process identity. All I/O flows through Runtime, enabling sandboxed test environments.
