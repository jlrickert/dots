# dots — System Design (v5)

A brew-style dotfile package manager with taps, profiles, overlays, and cross-platform support.

---

## Core Concepts

**Tap** — A named pointer to a Git repo (GitHub, Bitbucket, any Git remote). Public or private. One repo can hold many packages.

**Package** — A directory inside a tap repo. Contains a `Dotfile.yaml` manifest and the actual config files. Independently installable.

**Overlay** — A package that declares itself as a layer on top of another package. Merges its files into an already-installed base.

**Profile** — A named, ordered list of packages and overlays.

**Platform** — An OS-arch pair (e.g. `darwin-arm64`, `linux-amd64`, `windows-amd64`). Used to resolve platform-specific links, hooks, and behavior within a single manifest.

**Path Alias** — A named shorthand (e.g. `@config`, `@data`) that resolves to the correct platform-native path. Keeps manifests portable across Unix and Windows.

**Link Strategy** — How dots places files in their target location: `symlink`, `copy`, or `hardlink`. Configurable globally, per-platform, or per-package.

**Lockfile** — Records exactly what's installed, from which tap, at which commit.

**Work Mode** — A development mode where dots points at your local repo checkout instead of its internal clone.

---

## Path Resolution

### Unix (macOS, Linux, FreeBSD)

dots follows XDG. Every path resolves through environment variables with sensible defaults.

```
DOTS_CONFIG_DIR = $XDG_CONFIG_HOME/dots   (default: ~/.config/dots)
DOTS_STATE_DIR  = $XDG_STATE_HOME/dots    (default: ~/.local/state/dots)
```

### Windows (Native)

Windows uses its own standard locations, with environment variable overrides.

```
DOTS_CONFIG_DIR = %DOTS_CONFIG_DIR%       (default: %APPDATA%\dots)
DOTS_STATE_DIR  = %DOTS_STATE_DIR%        (default: %LOCALAPPDATA%\dots\state)
```

### WSL

WSL is treated as Linux. It uses XDG paths inside the WSL filesystem. If a WSL user wants to manage native Windows configs too, they run a separate dots instance on the Windows side.

### What Lives Where

**Config** (`$DOTS_CONFIG_DIR`) — things you author and edit:

```
# Unix
~/.config/dots/
├── config.yaml
└── profiles/

# Windows
%APPDATA%\dots\
├── config.yaml
└── profiles\
```

**State** (`$DOTS_STATE_DIR`) — things dots manages:

```
# Unix
~/.local/state/dots/
├── dots.lock.yaml
├── taps/
├── merged/
└── backups/

# Windows
%LOCALAPPDATA%\dots\state\
├── dots.lock.yaml
├── taps\
├── merged\
└── backups\
```

---

## Path Aliases

Path aliases solve the cross-platform path problem. A link target of `@config/nvim/init.lua` resolves to the right location on every platform without requiring platform sections.

### Built-in Aliases

| Alias     | macOS                        | Linux                        | Windows                    |
|-----------|------------------------------|------------------------------|----------------------------|
| `@home`   | `$HOME`                     | `$HOME`                     | `%USERPROFILE%`            |
| `@config` | `$XDG_CONFIG_HOME` (~/.config) | `$XDG_CONFIG_HOME` (~/.config) | `%APPDATA%`               |
| `@data`   | `$XDG_DATA_HOME` (~/.local/share) | `$XDG_DATA_HOME` (~/.local/share) | `%LOCALAPPDATA%`          |
| `@cache`  | `$XDG_CACHE_HOME` (~/.cache) | `$XDG_CACHE_HOME` (~/.cache) | `%LOCALAPPDATA%\cache`    |
| `@state`  | `$XDG_STATE_HOME` (~/.local/state) | `$XDG_STATE_HOME` (~/.local/state) | `%LOCALAPPDATA%\state`   |
| `@bin`    | `~/.local/bin`              | `~/.local/bin`              | `%LOCALAPPDATA%\bin`       |

### Usage in Manifests

```yaml
links:
  init.lua: @config/nvim/init.lua        # ~/.config/nvim/init.lua on Unix
                                          # %APPDATA%\nvim\init.lua on Windows
```

### Raw Paths Still Work

Aliases are optional. Raw paths are always relative to `$HOME` (Unix) or `%USERPROFILE%` (Windows).

```yaml
links:
  .gitconfig: .gitconfig                  # $HOME/.gitconfig on all platforms
```

For files that land in different locations per platform, use platform sections with raw paths or aliases — whichever is clearer:

```yaml
links:
  config: @config/ghostty/config

platform:
  windows-amd64:
    links:
      config: @config/ghostty/config      # %APPDATA%\ghostty\config
      # or equivalently with a raw Windows path:
      # config: AppData/Roaming/ghostty/config
```

### Custom Aliases

Users can define custom aliases in their config:

```yaml
# in config.yaml
aliases:
  @dots: @config/dots
  @nvim: @config/nvim
  @scripts: @home/scripts
```

### Path Separator Normalization

Manifests always use forward slashes. dots normalizes to the platform-native separator at resolution time. You write `@config/nvim/init.lua` and dots resolves it to `%APPDATA%\nvim\init.lua` on Windows.

---

## Link Strategy

### The Problem

Symlinks are the ideal link mechanism — edit in place, changes propagate instantly. But they're not universally available. Windows requires Developer Mode or admin privileges for symlinks. Some filesystems don't support them. Some deployment scenarios (USB drives, shared folders) need copies.

### Configuration

Link strategy is configurable at three levels. More specific wins.

**Global** (in `config.yaml`):
```yaml
core:
  link_strategy: symlink     # symlink | copy | hardlink
```

**Per-platform** (in `config.yaml`):
```yaml
core:
  link_strategy: symlink

platform:
  windows:
    link_strategy: copy      # Windows defaults to copy
```

**Per-package** (in `Dotfile.yaml`):
```yaml
package:
  name: ssh
  link_strategy: copy        # always copy — ssh doesn't follow symlinks
```

### Resolution Order

1. Package-level `link_strategy` (highest priority)
2. Platform-level in `config.yaml`
3. Global `link_strategy` in `config.yaml`
4. Default: `symlink`

### Strategy Behaviors

| Strategy   | Behavior | Work Mode | Edits Propagate |
|------------|----------|-----------|-----------------|
| `symlink`  | Create symlink from target → source | Yes, instant | Instantly — same file |
| `copy`     | Copy file from source to target | Yes, but requires `dots sync` | No — must run `dots sync` |
| `hardlink` | Create hardlink (files only, same filesystem) | Yes, instant | Instantly — same inode |

### The `dots sync` Command

When using `copy` strategy, edits to source files don't automatically appear at the target. `dots sync` bridges this gap:

```
dots sync [<tap>/<package>]         # re-copy changed files
dots sync --all                      # sync everything using copy strategy
dots sync --watch                    # watch for changes and auto-sync
```

`dots sync` compares checksums and only copies files that changed. It's a no-op for packages using symlink or hardlink strategy.

### Windows Auto-Detection

On Windows, dots checks for symlink capability at startup:

1. Try to create a test symlink in `$DOTS_STATE_DIR`
2. If it succeeds → symlinks are available, use configured strategy
3. If it fails → warn the user, fall back to `copy` regardless of config

```
⚠ Symlinks unavailable (enable Developer Mode or run as admin).
  Falling back to copy strategy. Run `dots sync` after editing source files.
```

### Work Mode + Copy Strategy

Work mode with copy strategy is slightly different from symlink mode:

- Symlink mode: `~/.config/nvim/init.lua` → `~/code/dotfiles/nvim/init.lua` (same file)
- Copy mode: `~/.config/nvim/init.lua` is a copy of `~/code/dotfiles/nvim/init.lua`

With copies, you edit in `~/code/dotfiles/` and run `dots sync` to push changes to the targets. Or use `dots sync --watch` for automatic propagation.

---

## Platform System

### Identifiers

Platforms are OS-arch pairs using Go/uname conventions:

| Identifier        | OS      | Arch    |
|--------------------|---------|---------|
| `darwin-arm64`     | macOS   | Apple Silicon |
| `darwin-amd64`     | macOS   | Intel |
| `linux-amd64`     | Linux   | x86_64 |
| `linux-arm64`     | Linux   | aarch64 |
| `windows-amd64`   | Windows | x86_64 |
| `windows-arm64`   | Windows | ARM64 |
| `freebsd-amd64`   | FreeBSD | x86_64 |

dots detects the current platform at runtime:
- Unix: `uname -s` + `uname -m`, normalized to lowercase
- Windows: OS detection + `%PROCESSOR_ARCHITECTURE%`
- WSL: detected as `linux` (the kernel is Linux)

### Resolution Cascade

```
base  →  OS-only  →  OS-arch
```

Deep merge maps, replace scalars. More specific wins.

### Manifest with Platform Sections

```yaml
package:
  name: nvim
  description: Neovim configuration
  version: 1.2.0
  tags: [editor, neovim]

links:
  init.lua: @config/nvim/init.lua
  lua/: @config/nvim/lua/
  after/: @config/nvim/after/

hooks:
  post_install: scripts/install-plugins.sh

platform:
  darwin:
    links:
      helpers/mac-clipboard.lua: @config/nvim/lua/clipboard.lua
    hooks:
      post_install: scripts/install-plugins-mac.sh

  darwin-arm64:
    links:
      bin/nvim-silicon-arm: @bin/nvim-silicon

  linux:
    links:
      helpers/xclip.lua: @config/nvim/lua/clipboard.lua

  windows:
    links:
      helpers/win-clipboard.lua: @config/nvim/lua/clipboard.lua
    hooks:
      post_install: scripts/install-plugins.ps1
```

### Resolved Example — `windows-amd64`

```yaml
# Effective manifest (computed, never written to disk)
links:
  init.lua: %APPDATA%\nvim\init.lua                        # @config → %APPDATA%
  lua/: %APPDATA%\nvim\lua\
  after/: %APPDATA%\nvim\after\
  helpers/win-clipboard.lua: %APPDATA%\nvim\lua\clipboard.lua  # from windows

hooks:
  post_install: scripts/install-plugins.ps1                 # windows replaced base
```

### Platform-Only Packages

```yaml
package:
  name: aerospace
  description: macOS tiling window manager config
  platforms: [darwin-arm64, darwin-amd64]

links:
  config.toml: @config/aerospace/config.toml
```

### Hooks and Platform

Hooks can be platform-specific scripts. The convention:

```
scripts/
├── install-plugins.sh          # Unix default
├── install-plugins-mac.sh      # macOS override
└── install-plugins.ps1         # Windows override
```

The platform section points to the right script. dots does not interpret or translate scripts — it runs whatever the manifest specifies using the platform's native shell:
- Unix: `$SHELL` or `/bin/sh`
- Windows: `powershell.exe` for `.ps1`, `cmd.exe` for `.bat`/`.cmd`

---

## Self-Bootstrapping

### The Bootstrap Command

```bash
# Unix
dots init --from git@github.com:me/dotfiles.git --path dots

# Windows (PowerShell)
dots init --from git@github.com:me/dotfiles.git --path dots
```

Same command, same behavior. dots resolves platform-specific paths automatically.

### The dots Package

```yaml
package:
  name: dots
  description: dots package manager configuration
  version: 1.0.0
  tags: [meta, dots]

links:
  config.yaml: @config/dots/config.yaml
  profiles/: @config/dots/profiles/
```

Using `@config` here means the dots config lands in the right place on every platform without platform sections.

### Recovery

- `dots init --from <url>` always works — bypasses local config
- `dots --config <path>` flag overrides config location
- `dots doctor` validates config, taps, links, and link strategy capability
- Lockfile in state — dots knows what it installed even if config breaks

---

## Data Model

### Package Manifest (`Dotfile.yaml`) — Full Schema

```yaml
package:
  name: string                     # required
  description: string
  version: string                  # semver
  requires: [tap/package, ...]     # dependencies
  tags: [string, ...]
  platforms: [os-arch, ...]        # restrict to platforms (default: all)
  link_strategy: symlink | copy | hardlink   # override global strategy

links:                              # required — source: target
  source-path: target-path          # target can use @aliases or raw paths

hooks:
  pre_install: script-path
  post_install: script-path
  pre_remove: script-path
  post_remove: script-path
  pre_upgrade: script-path
  post_upgrade: script-path

overlay:                            # omit for base packages
  base: tap/package
  strategy: append | prepend | replace | merge
  priority: 0-99

merge:                              # per-file strategy overrides
  filename: append | prepend | replace | merge

platform:
  <os>:                             # darwin, linux, windows, freebsd
    links: {}
    hooks: {}
    requires: []
    tags: []
    overlay: {}
    merge: {}
    link_strategy: symlink | copy | hardlink
  <os-arch>:                        # darwin-arm64, windows-amd64, etc.
    # same structure
```

### Config (`~/.config/dots/config.yaml`) — Full Schema

```yaml
core:
  active_profile: string
  conflict_strategy: overlay | prompt | skip
  backup: true | false
  link_strategy: symlink | copy | hardlink

git:
  default_branch: main
  protocol: ssh | https

taps:
  <name>:
    url: string
    branch: string
    provider: github | bitbucket | generic
    visibility: public | private

work_mode:
  <tap-name>: /local/path

aliases:
  @custom: @config/custom/path

platform:
  windows:
    link_strategy: copy
  # any platform can override core settings
```

### Profile

```yaml
profile:
  name: work
  description: Full work environment
  extends: personal

packages:
  - personal/zsh
  - personal/nvim
  - personal/tmux
  - personal/git
  - work/git
  - work/nvim
  - work/ssh
  - work/aws
```

### Lockfile

```yaml
state:
  active_profile: work
  last_applied: 2025-03-15T10:30:00Z
  platform: darwin-arm64
  link_strategy: symlink

installed:
  - package: personal/nvim
    tap: personal
    commit: a1b2c3d
    version: 1.2.0
    type: base
    link_strategy: symlink
    platform_resolved: [base, darwin, darwin-arm64]
    files:
      - src: init.lua
        dest: ~/.config/nvim/init.lua
        origin: base
        method: symlink
        checksum: sha256:abc123
      - src: helpers/mac-clipboard.lua
        dest: ~/.config/nvim/lua/clipboard.lua
        origin: darwin
        method: symlink
        checksum: sha256:def456
```

---

## Work Mode

### Interface

```
dots work on <tap> <local-path>
dots work off <tap>
dots work status
dots rebuild [<tap>/<package>]
```

### Behavior by Link Strategy

| Strategy | Work mode behavior | Edit propagation |
|----------|-------------------|------------------|
| `symlink` | Rewire symlinks to local path | Instant |
| `hardlink` | Recreate hardlinks to local path | Instant |
| `copy` | Set local path as source, require `dots sync` | Manual via `dots sync` |

### The `dots sync` Command

```
dots sync [<tap>/<package>]
dots sync --all
dots sync --watch                   # file watcher, auto-sync on change
```

Only relevant for `copy` strategy. No-op for symlink/hardlink. Compares checksums, copies only changed files.

---

## Merge System

### Strategies

| Strategy  | Behavior |
|-----------|----------|
| `append`  | Overlay content after base, with marker comment |
| `prepend` | Overlay content before base |
| `replace` | Overlay file wins entirely |
| `merge`   | Deep merge for structured formats (v2) |

### Marker Comments

```bash
# ── base: personal/zsh ──────────────────────────
export PATH="$HOME/.local/bin:$PATH"

# ── overlay: work/zsh (priority: 50) ────────────
export HTTP_PROXY="http://proxy.corp:8080"
```

### Merged File Storage

```
$DOTS_STATE_DIR/merged/
└── work/
    └── nvim/
        └── init.lua
```

---

## CLI Interface

### Init & Config
```
dots init
dots init --from <url> --path <dir>
dots config get <key>
dots config set <key> <value>
dots doctor
```

### Taps
```
dots tap add <n> <url> [--branch main]
dots tap remove <n>
dots tap list
dots tap update [<n>]
```

### Install & Remove
```
dots install <tap>/<package>
dots install <tap>/<package> --dry-run
dots install <tap>/<package> --strategy copy
dots remove <tap>/<package>
dots reinstall <tap>/<package>
dots upgrade [<tap>/<package>]
dots upgrade --all
```

### Work Mode
```
dots work on <tap> <local-path>
dots work off <tap>
dots work status
dots rebuild [<tap>/<package>]
```

### Sync (Copy Strategy)
```
dots sync [<tap>/<package>]
dots sync --all
dots sync --watch
```

### Status & Inspection
```
dots status
dots list
dots list --available
dots list --available --all-platforms
dots info <tap>/<package>
dots info --platform
dots diff <tap>/<package>
dots which <file>
```

### Profiles
```
dots profile create <n>
dots profile delete <n>
dots profile list
dots profile show <n>
dots profile add <n> <tap>/<package>...
dots profile remove <n> <tap>/<package>
dots profile apply <n>
dots profile switch <n>
dots profile export <n>
dots profile import <file>
```

### Discovery
```
dots search <query>
dots browse <tap>
dots browse <tap>/<package>
```

---

## Lifecycle Scenarios

### Fresh machine — macOS
```bash
brew install dots
dots init --from git@github.com:me/dotfiles.git --path dots
dots profile apply work
dots work on personal ~/code/dotfiles
```

### Fresh machine — Windows
```powershell
scoop install dots    # or winget, choco
dots init --from git@github.com:me/dotfiles.git --path dots
# dots auto-detects symlink capability
# if unavailable, falls back to copy with a warning
dots profile apply work
dots work on personal C:\Users\me\code\dotfiles
```

### Fresh machine — WSL
```bash
# WSL is Linux — same as any Linux setup
sudo apt install dots   # or brew, cargo
dots init --from git@github.com:me/dotfiles.git --path dots
dots profile apply work
```

### Adding a cross-platform package
```yaml
# ~/code/dotfiles/git/Dotfile.yaml
package:
  name: git
  description: Git configuration
  version: 1.0.0
  tags: [git, vcs]

links:
  .gitconfig: .gitconfig
  .gitignore_global: .gitignore_global

platform:
  windows:
    hooks:
      post_install: scripts/setup-credential-manager.ps1

  darwin:
    hooks:
      post_install: scripts/setup-keychain.sh

  linux:
    hooks:
      post_install: scripts/setup-credential-helper.sh
```

The `.gitconfig` link is the same everywhere — `$HOME/.gitconfig` on Unix, `%USERPROFILE%\.gitconfig` on Windows. Only the credential setup hook differs.

### Package that uses path aliases for portability
```yaml
# ~/code/dotfiles/alacritty/Dotfile.yaml
package:
  name: alacritty
  description: Alacritty terminal config
  version: 1.0.0
  tags: [terminal]

links:
  alacritty.toml: @config/alacritty/alacritty.toml
  themes/: @config/alacritty/themes/

platform:
  darwin:
    links:
      fonts.toml: @config/alacritty/fonts-mac.toml

  linux:
    links:
      fonts.toml: @config/alacritty/fonts-linux.toml

  windows:
    links:
      fonts.toml: @config/alacritty/fonts-windows.toml
```

`@config/alacritty/alacritty.toml` resolves to:
- macOS: `~/.config/alacritty/alacritty.toml`
- Linux: `~/.config/alacritty/alacritty.toml`
- Windows: `%APPDATA%\alacritty\alacritty.toml`

One manifest. No per-platform path juggling.

---

## Design Principles

1. **XDG-native on Unix, platform-native on Windows.** Respect `XDG_CONFIG_HOME`/`XDG_STATE_HOME` on Unix. Use `%APPDATA%`/`%LOCALAPPDATA%` on Windows. WSL is Linux.

2. **Self-bootstrapping.** dots manages its own config as a package. One command from bare machine to fully configured.

3. **Taps are just Git repos.** No registry, no special hosting.

4. **Packages are just directories.** `Dotfile.yaml` is the only special file. Config files are real config files, not templates.

5. **No template language.** Cross-platform uses platform sections with cascading resolution. Path aliases handle location differences. Your `.zshrc` is always a valid `.zshrc`.

6. **Path aliases for portability.** `@config`, `@data`, `@home` resolve to platform-native paths. Raw paths still work when you don't need portability.

7. **Configurable link strategy.** Symlink by default, copy or hardlink when needed. Per-platform and per-package overrides. Windows auto-detects capability.

8. **Work mode is the natural state** for your own dotfiles. `dots sync` bridges the gap when using copy strategy.

9. **Overlays are explicit.** A package declares what it overlays.

10. **Profiles are local** but version-controllable via self-bootstrapping.

11. **Everything is reversible.** Backups, lockfile, clean remove.

12. **No magic.** Symlinks (or copies), readable YAML, visible merge markers. The lockfile records every file's origin layer, link method, and checksum.

13. **YAML everywhere.** One format for all configuration.
