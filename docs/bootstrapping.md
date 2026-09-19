# Bootstrapping

Bootstrapping is the process of setting up dots on a new machine. dots supports both fresh initialization and cloning from an existing dotfiles repository.

## Fresh Start

If you don't have an existing dotfiles repo:

```bash
dots init
```

This creates a default config file at:

- **Unix**: `~/.config/dots/config.yaml`
- **Windows**: `%APPDATA%\dots\config.yaml`

From there, add taps and install packages:

```bash
dots tap add personal git@github.com:you/dotfiles.git
dots install personal/nvim
```

## One-Command Bootstrap

If you already have a dotfiles repo, bootstrap everything in one command:

```bash
dots init --from git@github.com:you/dotfiles.git --path dots --apply work
```

This:

1. Clones the repo as a tap
2. Installs the package at the specified `--path` within the tap
3. The installed package typically contains dots' own config and profiles, completing the bootstrap
4. Applies the `work` profile (see [Applying a Profile](profiles.md#applying-a-profile))

`--apply` runs only after the bootstrap package is installed, so profiles shipped by that package are available to it. If the profile can't be found, `init` reports the error after the bootstrap has already succeeded; fix the profile and run `dots profile apply work`.

`--apply` also works without `--from`, on a machine where the profile already exists:

```bash
dots init --apply work
```

`apply` uninstalls every package that isn't in the profile. To keep the bootstrap package installed, list it in `core.always_installed` in the config it ships (see [The `dots` Package Pattern](#the-dots-package-pattern)). `init` reloads config after installing the bootstrap package, so `--apply` sees that setting.

You can also skip `--apply` and apply a profile later:

```bash
dots profile apply work
```

## The `dots` Package Pattern

A common pattern is to include a `dots` package in your tap that manages dots' own configuration:

```
taps/personal/
  dots/
    Dotfile.yaml
    config.yaml
    profiles/
      base.yaml
      work.yaml
  nvim/
    Dotfile.yaml
    init.lua
  ...
```

The `dots/Dotfile.yaml`:

```yaml
package:
  name: dots
  description: dots self-configuration
  tags: [meta]

links:
  config.yaml: "@config/dots/config.yaml"
  profiles/base.yaml: "@config/dots/profiles/base.yaml"
  profiles/work.yaml: "@config/dots/profiles/work.yaml"
```

The `config.yaml` inside the dots package contains your tap registrations, aliases, and preferences. The profile files define which packages each kind of machine gets. When `dots init --from <url> --path dots` installs this package, it links both into place, giving dots full knowledge of your taps, settings, and profiles, which is what lets `--apply` work in the same command.

Link each profile file individually; directory links (a trailing `/`) are not supported yet.

The package's `config.yaml` should list the package itself in `core.always_installed`, so applying any profile keeps it installed:

```yaml
# dots/config.yaml
core:
  always_installed:
    - personal/dots

taps:
  personal:
    url: git@github.com:you/dotfiles.git
```

Without this, every profile would have to list `personal/dots`, and applying one that didn't would uninstall the package that defines your config and profiles. See [Always-Installed Packages](profiles.md#always-installed-packages).

## Recovery

If dots is installed but your config is missing or corrupted:

```bash
# Re-initialize with defaults
dots init

# Or re-bootstrap from your repo and re-apply your profile
dots init --from git@github.com:you/dotfiles.git --path dots --apply work
```

If you have a profile exported:

```bash
dots profile import work-profile.yaml
dots profile apply work
```

## Platform-Specific Bootstrap

The `dots` package can include platform-specific configuration:

```yaml
package:
  name: dots
  description: dots self-configuration

links:
  config.yaml: "@config/dots/config.yaml"

platform:
  darwin:
    hooks:
      post_install: "xcode-select --install 2>/dev/null || true"
  linux:
    hooks:
      post_install: scripts/install-deps.sh
```

This lets the bootstrap process install platform-specific prerequisites automatically.

## Recommended Bootstrap Script

For fully automated setup on a new machine, create a small bootstrap script:

```bash
#!/bin/bash
# bootstrap.sh - Set up a new machine

# Install dots (via Homebrew or from source)
brew install jlrickert/formulae/dots

# Bootstrap from your dotfiles repo and apply your profile
dots init --from git@github.com:you/dotfiles.git --path dots --apply "${1:-work}"
```

Save this somewhere accessible (e.g., a GitHub gist) so you can run it on any new machine.
