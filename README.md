# dots

A brew-style dotfile package manager with taps, profiles, overlays, and cross-platform support. dots treats Git repos as package sources ("taps"), uses path aliases for portability, and supports symlink/copy/hardlink link strategies with platform-aware resolution.

## Installation

### From source

```bash
go install github.com/jlrickert/dots/cmd/dots@latest
```

### Build from repo

```bash
git clone https://github.com/jlrickert/dots.git
cd dots
go build ./cmd/dots
```

## Quick Start

### Bootstrapping a fresh machine

Initialize dots and pull in your dotfiles repo in one command:

```bash
dots init --from git@github.com:you/dotfiles.git --path dots
```

This clones the repo as a tap and installs the `dots` package from it, which sets up dots' own configuration. From there you can apply a profile or install individual packages:

```bash
dots profile apply work
```

### Bootstrapping without a repo

If you're starting fresh without an existing dotfiles repo:

```bash
dots init
```

This creates a default config at `~/.config/dots/config.yaml` (Unix) or `%APPDATA%\dots\config.yaml` (Windows).

## Adding Taps

Taps are Git repos that contain your dotfile packages. Register one with:

```bash
dots tap add personal git@github.com:you/dotfiles.git
dots tap add personal git@github.com:you/dotfiles.git --branch develop
```

Manage taps with:

```bash
dots tap list                # list registered taps
dots tap update              # update all taps
dots tap update personal     # update a specific tap
dots tap remove personal     # remove a tap
```

## Installing Packages

A package is a directory inside a tap containing a `Dotfile.yaml` manifest and the config files it manages.

```bash
dots install personal/nvim
dots install personal/nvim --dry-run        # preview without changes
dots install personal/nvim --strategy copy  # override link strategy
```

Other package operations:

```bash
dots remove personal/nvim       # uninstall
dots upgrade personal/nvim      # upgrade to latest
dots upgrade --all              # upgrade everything
dots reinstall personal/nvim    # remove then reinstall
```

## Discovering Packages

```bash
dots list                    # list installed packages
dots list --available        # list available (uninstalled) packages
dots search nvim             # search across all taps
dots browse personal         # list all packages in a tap
dots info personal/nvim      # show package details
```

## Profiles

Profiles are named lists of packages that can be applied together:

```bash
dots profile create work
dots profile add work personal/zsh personal/nvim personal/git
dots profile apply work
dots profile switch personal   # switch to a different profile
```

Profiles support inheritance via `extends` and can be exported/imported:

```bash
dots profile export work > work-profile.yaml
dots profile import work-profile.yaml
```

## Work Mode

Work mode points dots at your local repo checkout instead of its internal clone, so edits propagate immediately (with symlinks):

```bash
dots work on personal ~/code/dotfiles
dots work off personal
dots work status
```

## Writing a Package Manifest

Each package needs a `Dotfile.yaml` in its directory:

```yaml
package:
  name: nvim
  description: Neovim configuration
  version: 1.0.0
  tags: [editor, neovim]

links:
  init.lua: @config/nvim/init.lua
  lua/: @config/nvim/lua/

hooks:
  post_install: scripts/install-plugins.sh

platform:
  darwin:
    links:
      helpers/mac-clipboard.lua: @config/nvim/lua/clipboard.lua
  windows:
    links:
      helpers/win-clipboard.lua: @config/nvim/lua/clipboard.lua
    hooks:
      post_install: scripts/install-plugins.ps1
```

Link targets use **path aliases** for cross-platform portability:

| Alias | macOS / Linux | Windows |
|-------|---------------|---------|
| `@home` | `$HOME` | `%USERPROFILE%` |
| `@config` | `$XDG_CONFIG_HOME` (~/.config) | `%APPDATA%` |
| `@data` | `$XDG_DATA_HOME` (~/.local/share) | `%LOCALAPPDATA%` |
| `@cache` | `$XDG_CACHE_HOME` (~/.cache) | `%LOCALAPPDATA%\cache` |
| `@state` | `$XDG_STATE_HOME` (~/.local/state) | `%LOCALAPPDATA%\state` |
| `@bin` | `~/.local/bin` | `%LOCALAPPDATA%\bin` |

Raw paths (without aliases) are relative to `$HOME` / `%USERPROFILE%`. Manifests always use forward slashes; dots normalizes to platform-native separators at resolution time.

Platform sections cascade in specificity order: **base -> OS -> OS-arch**. Maps deep-merge, scalars replace.

## Link Strategy

dots supports three strategies, configurable globally, per-platform, or per-package:

| Strategy | Behavior | Edits propagate? |
|----------|----------|-------------------|
| `symlink` (default) | Symlink from target to source | Instantly |
| `copy` | Copy source to target | No, run `dots sync` |
| `hardlink` | Hardlink (same filesystem) | Instantly |

When using `copy` strategy, use `dots sync` to push changes:

```bash
dots sync personal/nvim    # sync one package
dots sync --all            # sync all copy-strategy packages
```

On Windows, dots auto-detects symlink capability and falls back to `copy` if unavailable.

## Diagnostics

```bash
dots status    # overview: platform, config path, active profile, installed counts
dots doctor    # run diagnostic checks on config, taps, and links
dots diff personal/nvim   # show differences between source and installed files
dots which ~/.config/nvim/init.lua   # identify which package owns a file
```

## Configuration

Config lives at `~/.config/dots/config.yaml` (Unix) or `%APPDATA%\dots\config.yaml` (Windows):

```yaml
core:
  link_strategy: symlink
  conflict_strategy: prompt
  backup: true

git:
  default_branch: main
  protocol: ssh

taps:
  personal:
    url: git@github.com:you/dotfiles.git
    branch: main

aliases:
  @nvim: @config/nvim
  @scripts: @home/scripts

platform:
  windows:
    link_strategy: copy
```

For the full design specification, see [dots-design-v5.md](dots-design-v5.md).
