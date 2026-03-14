# Profiles

Profiles are named collections of packages that can be applied together. They let you maintain different sets of dotfiles for different contexts (e.g., work vs personal machines) and switch between them.

## Creating a Profile

```bash
dots profile create work
```

This creates an empty profile. Add packages to it:

```bash
dots profile add work personal/zsh personal/nvim personal/git personal/tmux
```

## Applying a Profile

Install all packages in a profile:

```bash
dots profile apply work
```

This installs every package listed in the profile that isn't already installed.

## Switching Profiles

Switch the active profile:

```bash
dots profile switch personal
```

This sets `core.active_profile` in your config. It does not automatically install or remove packages — use `dots profile apply` after switching to install the new profile's packages.

## Managing Packages in Profiles

```bash
dots profile add work personal/docker       # add a package
dots profile remove work personal/docker    # remove a package
```

## Viewing Profiles

```bash
dots profile list          # list all profile names
dots profile show work     # show details of a profile
```

`show` displays the profile name, any `extends` parent, and the full package list.

## Inheritance with `extends`

A profile can inherit packages from a parent profile:

```yaml
name: work
extends: base
packages:
  - personal/slack
  - personal/docker
```

When `work` extends `base`, applying `work` installs all packages from `base` plus the packages listed in `work`. The inheritance chain is resolved with deduplication — a package listed in both parent and child only installs once.

## Export and Import

Export a profile to share it or back it up:

```bash
dots profile export work > work-profile.yaml
```

Import a profile from a YAML file:

```bash
dots profile import work-profile.yaml
```

This is useful for:
- Sharing profiles between machines
- Backing up profile definitions
- Version-controlling profiles separately from taps

## Storage

Profiles are stored as YAML files in the dots config directory under `profiles/`:

| Platform | Path |
|----------|------|
| macOS / Linux | `~/.config/dots/profiles/<name>.yaml` |
| Windows | `%APPDATA%\dots\profiles\<name>.yaml` |

They are managed entirely through the `dots profile` commands — you don't need to edit profile files directly.

## Example Workflow

```bash
# Set up profiles for different machines
dots profile create base
dots profile add base personal/zsh personal/git personal/nvim

dots profile create work
# work inherits from base, so you only add work-specific packages
dots profile add work personal/docker personal/slack

dots profile create personal
dots profile add personal personal/gaming personal/media

# On a work machine
dots profile apply work       # installs base + work packages
dots profile switch work      # marks work as active

# On a personal machine
dots profile apply personal   # installs base + personal packages
dots profile switch personal
```
