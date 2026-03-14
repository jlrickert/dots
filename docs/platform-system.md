# Platform System

dots detects the current OS and architecture at runtime and uses this information to apply platform-specific configuration. This enables a single manifest or config file to work across macOS, Linux, Windows, and multiple CPU architectures.

## Platform Identifiers

A platform is an OS-architecture pair in `os-arch` format:

| Identifier | OS | Architecture |
|------------|------|--------------|
| `darwin-arm64` | macOS | Apple Silicon |
| `darwin-amd64` | macOS | Intel |
| `linux-amd64` | Linux | x86_64 |
| `linux-arm64` | Linux | ARM64 |
| `windows-amd64` | Windows | x86_64 |
| `windows-arm64` | Windows | ARM64 |
| `freebsd-amd64` | FreeBSD | x86_64 |

You can use either the OS alone (`darwin`) or the full OS-arch pair (`darwin-arm64`) as keys in platform sections. The OS-only key matches all architectures for that OS.

WSL is detected as `linux`.

Check your current platform:

```bash
dots info --platform
```

## Detection

dots uses Go's `runtime.GOOS` and `runtime.GOARCH` to detect the platform. No external tools or environment variables are needed.

## Cascade Resolution

When a manifest or config file contains platform-specific sections, dots merges them in order of increasing specificity:

```
base  ->  OS  ->  OS-arch
```

For example, on `darwin-arm64`:

1. Start with the **base** (top-level) configuration
2. Merge the **`darwin`** platform section (if present)
3. Merge the **`darwin-arm64`** platform section (if present)

Each step builds on the previous result. More specific sections override less specific ones.

## Merge Rules

How values combine depends on the data type:

| Type | Behavior | Example |
|------|----------|---------|
| Maps (links, merge) | Deep merge: new keys are added, existing keys are replaced | Base links + darwin links |
| Scalars (link_strategy, hooks) | Replace: more specific value wins | `copy` overrides `symlink` |
| Lists (tags, requires) | Concatenate with deduplication | `[editor]` + `[macos]` = `[editor, macos]` |

### Map Merge Example

```yaml
# Base links
links:
  init.lua: "@config/nvim/init.lua"
  lua/: "@config/nvim/lua/"

platform:
  darwin:
    links:
      clipboard.lua: "@config/nvim/lua/clipboard.lua"
```

On macOS, the resolved links contain all three entries. The `init.lua` and `lua/` entries come from the base, and `clipboard.lua` is added from the darwin section.

### Scalar Replace Example

```yaml
hooks:
  post_install: scripts/install.sh

platform:
  darwin:
    hooks:
      post_install: scripts/install-mac.sh
```

On macOS, `post_install` is `scripts/install-mac.sh`. On other platforms, it remains `scripts/install.sh`.

### List Concatenation Example

```yaml
package:
  tags: [editor]
  requires: [personal/zsh]

platform:
  darwin:
    tags: [macos]
    requires: [personal/homebrew]
```

On macOS, tags resolve to `[editor, macos]` and requires to `[personal/zsh, personal/homebrew]`. Duplicates are removed.

## Where Platform Sections Appear

### Dotfile.yaml Manifests

The `platform` block in a manifest can override `links`, `hooks`, `requires`, `tags`, `overlay`, `merge`, and `link_strategy`. See [Dotfile.yaml Reference](dotfile-yaml.md#platform-block).

### config.yaml

The `platform` block in the config can override `core` settings (`link_strategy`, `conflict_strategy`, `backup`). See [config.yaml Reference](config-yaml.md#platform-section).

## Resolved Manifest Walkthrough

Given this manifest on `darwin-arm64`:

```yaml
package:
  name: nvim
  tags: [editor]

links:
  init.lua: "@config/nvim/init.lua"

hooks:
  post_install: scripts/install.sh

platform:
  darwin:
    links:
      mac-clip.lua: "@config/nvim/lua/clipboard.lua"
    hooks:
      post_install: scripts/install-mac.sh
    tags: [macos]
  darwin-arm64:
    links:
      bin/nvim: "@bin/nvim"
```

The resolved result:

| Field | Value |
|-------|-------|
| links | `init.lua`, `mac-clip.lua`, `bin/nvim` (3 entries) |
| hooks.post_install | `scripts/install-mac.sh` (darwin overrode base) |
| tags | `[editor, macos]` (concatenated, deduped) |

Note that `darwin-arm64` did not override `post_install` again, so the darwin value persists.

## Platform-Only Packages

Use the `platforms` field in the `package` block to restrict a package to specific platforms:

```yaml
package:
  name: aerospace
  platforms: [darwin]
```

This package is skipped during installation on non-macOS systems. Both OS (`darwin`) and OS-arch (`darwin-arm64`) identifiers are accepted. An empty or omitted `platforms` list means the package works on all platforms.
