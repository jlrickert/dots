# Dotfile.yaml Reference

`Dotfile.yaml` is the package manifest file that declares how dots should install, link, and manage a package's configuration files. Every package directory must contain one.

## Location

A `Dotfile.yaml` lives at the root of a package directory inside a tap:

```
taps/
  personal/
    git/
      Dotfile.yaml
      .gitconfig
      .gitignore_global
    nvim/
      Dotfile.yaml
      init.lua
      lua/
      scripts/install-plugins.sh
```

## Schema Modeline

Add this modeline as the first line of your `Dotfile.yaml` to enable autocompletion and validation in editors that support `yaml-language-server`:

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/jlrickert/dots/main/schemas/dotfile.json
```

## `package` Block

The `package` block contains metadata about the package. Only `name` is required.

| Field           | Type     | Required | Description                                                                                                                                                          |
| --------------- | -------- | -------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `name`          | string   | yes      | Package name, used for identification and references                                                                                                                 |
| `description`   | string   | no       | Human-readable description                                                                                                                                           |
| `version`       | string   | no       | Semantic version string (e.g. `1.0.0`)                                                                                                                               |
| `requires`      | string[] | no       | Other packages this package depends on. Each entry is `<tap>/<package>`, or `@self/<package>` to resolve to the same tap as this package (e.g. `@self/common-shell`) |
| `tags`          | string[] | no       | Searchable tags for discovery (e.g. `[editor, neovim]`)                                                                                                              |
| `platforms`     | string[] | no       | Restrict installation to these platforms (e.g. `[darwin, linux-arm64]`)                                                                                              |
| `link_strategy` | string   | no       | Override link strategy for this package: `symlink`, `copy`, or `hardlink`                                                                                            |

If `platforms` is set, the package will only install on a matching OS or OS-arch. An empty list (or omitted) means all platforms.

```yaml
package:
  name: nvim
  description: Neovim configuration
  version: 1.0.0
  requires: [personal/zsh]
  tags: [editor, neovim]
  platforms: [darwin, linux]
  link_strategy: symlink
```

### Same-tap references with `@self/`

Intra-tap dependencies should use the `@self/` pseudo-prefix so the reference is portable across consumer-chosen tap aliases:

```yaml
package:
  name: bash
  requires:
    - "@self/common-shell" # resolves to '<this-tap>/common-shell'
```

At resolution time dots rewrites `@self/<package>` to `<current-tap>/<package>`. The current tap is the tap the manifest was loaded from. Cross-tap dependencies keep the explicit `<tap>/<package>` form.

## `links` Block

The `links` block maps source files (relative to the package directory) to target paths where they should be placed.

```yaml
links:
  init.lua: "@config/nvim/init.lua"
  lua/: "@config/nvim/lua/"
  .gitconfig: .gitconfig
```

**Source paths** (keys) are relative to the package directory inside the tap.

**Target paths** (values) support two forms:

- **Alias paths** start with `@` and resolve to platform-native directories. See [Path Aliases](path-aliases.md) for the full list.
- **Raw paths** (without `@`) are relative to `$HOME` (Unix) or `%USERPROFILE%` (Windows).

Manifests always use forward slashes (`/`). dots normalizes to platform-native separators at resolution time.

### Directory Linking

Trailing slashes indicate directory linking. The entire directory tree is linked as a unit:

```yaml
links:
  lua/: "@config/nvim/lua/"
```

## `hooks` Block

Hooks are lifecycle scripts that run at specific points during package operations. Each hook is either a path to a script file (relative to the package directory) or an inline shell command.

| Hook           | When it runs                      |
| -------------- | --------------------------------- |
| `pre_install`  | Before files are linked           |
| `post_install` | After files are linked            |
| `pre_remove`   | Before files are unlinked         |
| `post_remove`  | After files are unlinked          |
| `pre_upgrade`  | Before upgrade (after tap update) |
| `post_upgrade` | After upgrade completes           |

```yaml
hooks:
  post_install: scripts/install-plugins.sh
  pre_remove: "echo 'Removing nvim config'"
```

### Script Files vs Inline Commands

dots determines which form you're using by checking whether the value resolves to a file:

- **Script file**: If `<package-dir>/<hook-value>` exists as a file, it is executed as a script.
- **Inline command**: Otherwise, the value is passed to the shell via `$SHELL -c` (Unix) or `cmd.exe /C` (Windows).

### Shell Resolution

For script files, the shell is chosen by file extension:

| Extension      | Shell                                          | Platform |
| -------------- | ---------------------------------------------- | -------- |
| `.ps1`         | `powershell.exe -ExecutionPolicy Bypass -File` | Windows  |
| `.cmd`, `.bat` | `cmd.exe /C`                                   | Windows  |
| All others     | `$SHELL` (fallback: `/bin/sh`)                 | Unix     |

For inline commands, `$SHELL -c` is used on Unix and `cmd.exe` on Windows.

### Environment Variables

All hooks receive:

| Variable           | Value                                  |
| ------------------ | -------------------------------------- |
| `DOTS_PACKAGE_DIR` | Absolute path to the package directory |

The working directory is set to the package directory.

## `overlay` Block

The overlay system lets one package layer content on top of another. This is useful for machine-specific customizations that extend a base config.

```yaml
overlay:
  base: personal/zsh
  strategy: append
  priority: 50
```

| Field      | Type    | Required | Description                                                             |
| ---------- | ------- | -------- | ----------------------------------------------------------------------- |
| `base`     | string  | yes      | Base package reference (`tap/package`, or `@self/package` for same-tap) |
| `strategy` | string  | no       | Merge strategy: `append` (default), `prepend`, `replace`, `merge`       |
| `priority` | integer | no       | Priority 0-99, higher wins when multiple overlays target the same base  |

See [Overlays](overlays.md) for a detailed guide on merge strategies.

## `merge` Block

The `merge` block provides per-file strategy overrides for overlay merges. Keys are filenames, values are strategy names.

```yaml
overlay:
  base: personal/zsh
  strategy: append

merge:
  .zshenv: replace
  .zshrc: prepend
```

This overrides the overlay-level `strategy` for specific files. Any file not listed falls back to the overlay's default strategy.

## `platform` Block

The `platform` block contains OS-specific and OS-arch-specific overrides. Keys are platform identifiers.

### Platform Identifiers

| Key            | Matches                    |
| -------------- | -------------------------- |
| `darwin`       | macOS (any architecture)   |
| `linux`        | Linux (any architecture)   |
| `windows`      | Windows (any architecture) |
| `freebsd`      | FreeBSD (any architecture) |
| `darwin-arm64` | macOS on Apple Silicon     |
| `darwin-amd64` | macOS on Intel             |
| `linux-amd64`  | Linux on x86_64            |
| `linux-arm64`  | Linux on ARM64             |

### What Can Be Overridden

Each platform block can contain:

| Field           | Merge behavior                                      |
| --------------- | --------------------------------------------------- |
| `links`         | Maps merge (new keys added, existing keys replaced) |
| `hooks`         | Non-empty hooks replace base hooks                  |
| `requires`      | Lists concatenate with deduplication                |
| `tags`          | Lists concatenate with deduplication                |
| `overlay`       | Replaces entirely                                   |
| `merge`         | Maps merge (per-file overrides added/replaced)      |
| `link_strategy` | Replaces                                            |

### Cascade Resolution

Platform overrides are applied in specificity order:

1. **Base** (top-level `links`, `hooks`, etc.)
2. **OS** (e.g. `platform.darwin`)
3. **OS-arch** (e.g. `platform.darwin-arm64`)

More specific sections win. See [Platform System](platform-system.md) for the full cascade rules.

```yaml
package:
  name: nvim
  tags: [editor]

links:
  init.lua: "@config/nvim/init.lua"

platform:
  darwin:
    links:
      helpers/mac-clipboard.lua: "@config/nvim/lua/clipboard.lua"
    hooks:
      post_install: scripts/install-plugins-mac.sh
  darwin-arm64:
    links:
      bin/nvim-arm: "@bin/nvim"
```

On `darwin-arm64`, the resolved manifest would contain all three link entries and the macOS-specific `post_install` hook.

## Complete Examples

### Simple Package (git)

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/jlrickert/dots/main/schemas/dotfile.json
package:
  name: git
  description: Git configuration
  version: 1.0.0
  tags: [git, vcs]

links:
  .gitconfig: .gitconfig
  .gitignore_global: .gitignore_global
```

### Cross-Platform Package (nvim)

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/jlrickert/dots/main/schemas/dotfile.json
package:
  name: nvim
  description: Neovim configuration
  version: 1.0.0
  tags: [editor, neovim]

links:
  init.lua: "@config/nvim/init.lua"
  lua/: "@config/nvim/lua/"

hooks:
  post_install: scripts/install-plugins.sh

platform:
  darwin:
    links:
      helpers/mac-clipboard.lua: "@config/nvim/lua/clipboard.lua"
    hooks:
      post_install: scripts/install-plugins-mac.sh
  windows:
    links:
      helpers/win-clipboard.lua: "@config/nvim/lua/clipboard.lua"
    hooks:
      post_install: scripts/install-plugins.ps1
```

### Platform-Restricted Package (aerospace)

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/jlrickert/dots/main/schemas/dotfile.json
package:
  name: aerospace
  description: AeroSpace tiling window manager config
  version: 1.0.0
  tags: [wm, macos]
  platforms: [darwin]

links:
  .aerospace.toml: "@config/aerospace/aerospace.toml"

hooks:
  post_install: "brew install --cask nikitabobko/tap/aerospace"
```

### Overlay Package

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/jlrickert/dots/main/schemas/dotfile.json
package:
  name: zsh-work
  description: Work-specific zsh additions
  tags: [shell, work]

overlay:
  base: personal/zsh
  strategy: append
  priority: 50

merge:
  .zshenv: replace

links:
  work-aliases.zsh: "@config/zsh/work-aliases.zsh"
```
