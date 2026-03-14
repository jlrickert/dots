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
dots init --from git@github.com:you/dotfiles.git --path dots
```

This:

1. Clones the repo as a tap
2. Installs the package at the specified `--path` within the tap
3. The installed package typically contains dots' own config, completing the bootstrap

After the initial bootstrap, apply a profile to install the rest:

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
```

The `config.yaml` inside the dots package contains your tap registrations, aliases, and preferences. When `dots init --from <url> --path dots` installs this package, it links your config.yaml into place, giving dots full knowledge of your taps and settings.

## Recovery

If dots is installed but your config is missing or corrupted:

```bash
# Re-initialize with defaults
dots init

# Or re-bootstrap from your repo
dots init --from git@github.com:you/dotfiles.git --path dots
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

# Bootstrap from your dotfiles repo
dots init --from git@github.com:you/dotfiles.git --path dots

# Apply your profile
dots profile apply work
```

Save this somewhere accessible (e.g., a GitHub gist) so you can run it on any new machine.
