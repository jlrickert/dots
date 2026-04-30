# config.yaml Reference

`config.yaml` is the user configuration file for dots. It controls global behavior, registered taps, path aliases, and platform-specific overrides.

## Location

| Platform      | Path                         |
| ------------- | ---------------------------- |
| macOS / Linux | `~/.config/dots/config.yaml` |
| Windows       | `%APPDATA%\dots\config.yaml` |

Override the config path with the `--config` (`-c`) global flag:

```bash
dots --config /path/to/config.yaml status
```

If no config file exists, dots uses built-in defaults.

## Schema Modeline

Add this modeline as the first line for editor autocompletion and validation:

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/jlrickert/dots/main/schemas/dots-config.json
```

## `core` Section

Core behavior settings.

| Field               | Type    | Default     | Description                                                           |
| ------------------- | ------- | ----------- | --------------------------------------------------------------------- |
| `active_profile`    | string  | `""`        | Currently active [profile](profiles.md) name                          |
| `conflict_strategy` | string  | `"prompt"`  | How to handle file conflicts: `prompt`, `overwrite`, `skip`, `backup` |
| `backup`            | boolean | `true`      | Back up existing files before overwriting                             |
| `link_strategy`     | string  | `"symlink"` | Default link strategy: `symlink`, `copy`, `hardlink`                  |

See [Link Strategies - Backup Behavior](link-strategies.md#backup-behavior) for details on each `conflict_strategy`.

```yaml
core:
  active_profile: work
  conflict_strategy: backup
  backup: true
  link_strategy: symlink
```

## `git` Section

Git-related settings for tap management.

| Field            | Type   | Default  | Description                          |
| ---------------- | ------ | -------- | ------------------------------------ |
| `default_branch` | string | `"main"` | Default branch for new clones        |
| `protocol`       | string | `"ssh"`  | Preferred protocol: `ssh` or `https` |

```yaml
git:
  default_branch: main
  protocol: ssh
```

## `taps` Section

Registered taps keyed by alias name. Each tap entry describes a package source.

| Field        | Type   | Required | Description                                                 |
| ------------ | ------ | -------- | ----------------------------------------------------------- |
| `url`        | string | yes      | Git URL or local path                                       |
| `branch`     | string | no       | Git branch to track                                         |
| `provider`   | string | no       | Git hosting provider (e.g. `github`, `gitlab`, `bitbucket`) |
| `visibility` | string | no       | Repository visibility: `public` or `private`                |

```yaml
taps:
  personal:
    url: git@github.com:you/dotfiles.git
    branch: main
    provider: github
    visibility: private
  work:
    url: git@github.com:company/dotfiles.git
    branch: develop
```

Taps are typically managed via `dots tap add/remove` rather than editing this section directly.

## `work_mode` Section

Maps tap names to local directory paths for [work mode](work-mode.md). When a tap is in work mode, dots reads packages from the local path instead of its internal clone.

```yaml
work_mode:
  personal: /Users/you/code/dotfiles
```

This section is managed by `dots work on/off` commands.

## `aliases` Section

Custom [path aliases](path-aliases.md) that extend the built-in set. Values may reference other aliases (chaining), including any builtin from the default family (`@home`, `@config`, ...), the XDG family (`@xdg-*`), or the Apple family (`@apple-*`).

```yaml
aliases:
  "@nvim": "@config/nvim"
  "@scripts": "@home/scripts"
  "@dots": "@config/dots"
  "@launchd": "@apple-launchagents" # chains through an Apple-family builtin (darwin only)
```

## `platform` Section

Platform-specific overrides for `core` settings. Keys are OS or OS-arch identifiers (e.g. `darwin`, `linux`, `windows`, `darwin-arm64`).

Each platform block can override any field from the `core` section.

```yaml
platform:
  windows:
    link_strategy: copy
    conflict_strategy: backup
  darwin-arm64:
    link_strategy: symlink
```

Platform overrides follow the same cascade as [Dotfile.yaml](dotfile-yaml.md): **base core -> OS -> OS-arch**, with more specific values winning.

## Defaults

When no config file exists, dots uses these defaults:

```yaml
core:
  conflict_strategy: prompt
  backup: true
  link_strategy: symlink

git:
  default_branch: main
  protocol: ssh
```

## Complete Example

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/jlrickert/dots/main/schemas/dots-config.json
core:
  active_profile: work
  conflict_strategy: backup
  backup: true
  link_strategy: symlink

git:
  default_branch: main
  protocol: ssh

taps:
  personal:
    url: git@github.com:you/dotfiles.git
    branch: main
    provider: github
    visibility: private
  work:
    url: git@github.com:company/shared-dotfiles.git
    branch: develop

aliases:
  "@nvim": "@config/nvim"
  "@scripts": "@home/scripts"

platform:
  windows:
    link_strategy: copy
    conflict_strategy: backup
```
