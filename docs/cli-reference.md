# CLI Reference

Complete reference for all `dots` commands, flags, and subcommands.

## Global Flags

These flags are available on every command:

| Flag          | Short | Default       | Description         |
| ------------- | ----- | ------------- | ------------------- |
| `--config`    | `-c`  | (auto-detect) | Path to config file |
| `--log-file`  |       | stderr        | Write logs to file  |
| `--log-level` |       | `info`        | Minimum log level   |
| `--log-json`  |       | `false`       | Output logs as JSON |
| `--version`   |       |               | Show version        |

## Initialization

### `dots init`

Initialize the dots environment. Creates a default config file if one doesn't exist.

```bash
dots init
dots init --from git@github.com:you/dotfiles.git --path dots
dots init --from git@github.com:you/dot-personal.git --name private --path dots-config
dots init --from git@github.com:you/dotfiles.git --path dots --apply work
```

| Flag     | Description                                                                                                                                                                               |
| -------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `--from` | Git URL to clone as initial tap                                                                                                                                                           |
| `--path` | Package path within the tap to install                                                                                                                                                    |
| `--name` | Tap name for the `--from` clone (default: `default`). Set this to match the tap name declared in the bootstrap package's `config.yaml` so the on-disk tap and the installed config agree. |
| `--apply` | Profile to apply after initialization, with the same behavior as [`dots profile apply`](#dots-profile-apply).                                                                             |

When `--from` is specified, dots clones the repo as a tap and installs the package at `--path` from it.

When `--apply` is specified, dots applies the named profile after everything else, so profiles installed by the bootstrap package are available. If the profile is missing, the command fails after the bootstrap has already completed. Config is reloaded after the bootstrap install, so `core.always_installed` from the bootstrap package's config applies. List the bootstrap package there, or `apply` uninstalls it. See [Bootstrapping](bootstrapping.md#one-command-bootstrap).

If the bootstrap package (the one at `--path`) ships a `config.yaml` that declares the same repo as a named tap (e.g. `taps.private.url` matching `--from`), pass `--name private` so dots registers the clone under `private` rather than the generic `default`. Otherwise dots ends up with two taps — `default` and `private` — pointing at the same URL.

### Cleaning up a duplicate tap

If a prior `dots init --from <URL> --path <pkg>` left a stale `default` tap next to the name declared in the installed config, remove the orphan:

```bash
dots tap remove default
```

## Tap Management

### `dots tap add`

Register a new tap (package source).

```bash
dots tap add personal git@github.com:you/dotfiles.git
dots tap add work git@github.com:company/dotfiles.git --branch develop
```

| Flag       | Default | Description         |
| ---------- | ------- | ------------------- |
| `--branch` | `main`  | Git branch to track |

### `dots tap remove`

Remove a registered tap. Aliases: `rm`.

Any packages installed from the tap are uninstalled first (links and lockfile entries removed). If the on-disk tap directory is already gone, lockfile entries are still cleaned up — use this to reconcile a tap whose state has drifted.

```bash
dots tap remove personal
```

### `dots tap list`

List all registered taps. Aliases: `ls`.

```bash
dots tap list
```

Output: `NAME\tURL` per line.

### `dots tap update`

Update tap(s) to latest from remote.

```bash
dots tap update              # update all taps
dots tap update personal     # update a specific tap
```

## Package Operations

### `dots install`

Install a dotfile package.

```bash
dots install personal/nvim
dots install personal/nvim --dry-run
dots install personal/nvim --strategy copy
```

| Flag         | Description                                           |
| ------------ | ----------------------------------------------------- |
| `--dry-run`  | Print what would happen without writing               |
| `--strategy` | Override link strategy: `symlink`, `copy`, `hardlink` |

### `dots remove`

Remove an installed package. Aliases: `rm`, `uninstall`.

```bash
dots remove personal/nvim
```

### `dots upgrade`

Upgrade installed packages.

```bash
dots upgrade personal/nvim   # upgrade one package
dots upgrade --all           # upgrade all installed packages
```

| Flag    | Description                    |
| ------- | ------------------------------ |
| `--all` | Upgrade all installed packages |

### `dots reinstall`

Remove then reinstall a package.

```bash
dots reinstall personal/nvim
```

### `dots sync`

Sync copy-strategy packages with their source files.

```bash
dots sync personal/nvim      # sync one package
dots sync --all              # sync all copy-strategy packages
```

| Flag    | Description                     |
| ------- | ------------------------------- |
| `--all` | Sync all copy-strategy packages |

## Discovery

### `dots list`

List packages. Aliases: `ls`.

```bash
dots list                    # list installed packages
dots list --available        # list available (uninstalled) packages
dots list --tap personal     # filter by tap
```

| Flag          | Description                             |
| ------------- | --------------------------------------- |
| `--available` | List available (not installed) packages |
| `--tap`       | Filter by tap name                      |

### `dots search`

Search packages across all taps.

```bash
dots search nvim
```

Output marks installed packages with `*`.

### `dots browse`

List all packages in a tap with details.

```bash
dots browse personal
```

Shows package name, version, description, tags, link count, and installation status.

### `dots info`

Show package or platform information.

```bash
dots info personal/nvim      # package details
dots info --platform         # current platform
```

| Flag         | Description                                |
| ------------ | ------------------------------------------ |
| `--platform` | Show platform info instead of package info |

## Inspection

### `dots status`

Show dots status overview.

```bash
dots status
```

Output includes platform, config path, state directory, link strategy, active profile, tap count, and package count.

### `dots doctor`

Run diagnostic checks on config, taps, and links.

```bash
dots doctor
```

Output: `[OK|WARN|ERROR] CheckName: detail` per check.

### `dots diff`

Show differences between source and installed files.

```bash
dots diff personal/nvim
```

### `dots which`

Identify which package placed a file.

```bash
dots which ~/.config/nvim/init.lua
```

## Profiles

### `dots profile create`

Create a new empty profile.

```bash
dots profile create work
dots profile create work --extends base --extends dev
```

| Flag        | Description                                    |
| ----------- | ---------------------------------------------- |
| `--extends` | Parent profile to inherit from. Repeatable.    |

### `dots profile delete`

Delete a profile.

```bash
dots profile delete work
```

### `dots profile list`

List all profiles. Aliases: `ls`.

```bash
dots profile list
```

### `dots profile show`

Show profile details: name, `extends` parents, `exclude` list, and the resolved package list with the source of each package. Flags excludes that match no inherited package.

```bash
dots profile show work
```

### `dots profile add`

Add packages to a profile.

```bash
dots profile add work personal/zsh personal/nvim personal/git
```

### `dots profile remove`

Remove a package from a profile.

```bash
dots profile remove work personal/git
```

### `dots profile extend`

Add parent profiles to a profile's `extends` list.

```bash
dots profile extend work base dev
```

### `dots profile unextend`

Remove parent profiles from a profile's `extends` list.

```bash
dots profile unextend work dev
```

### `dots profile exclude`

Exclude inherited packages from a profile. Excludes only affect packages inherited through `extends`; the profile's own packages are always installed. See [Excluding Inherited Packages](profiles.md#excluding-inherited-packages).

```bash
dots profile exclude work personal/tmux
```

### `dots profile unexclude`

Remove packages from a profile's `exclude` list.

```bash
dots profile unexclude work personal/tmux
```

### `dots profile apply`

Make the installed packages match a profile exactly: uninstall every installed package not in the profile's resolved list plus `core.always_installed` (including packages installed by hand), install the missing ones, and record the profile as active in the lockfile. This is also how you change profiles.

```bash
dots profile apply work
dots profile apply personal --dry-run
```

| Flag        | Description                                                        |
| ----------- | ------------------------------------------------------------------ |
| `--dry-run` | Print the packages that would be removed and installed, then exit. |

### `dots profile export`

Export a profile as YAML to stdout.

```bash
dots profile export work > work-profile.yaml
```

### `dots profile import`

Import a profile from a YAML file.

```bash
dots profile import work-profile.yaml
```

## Work Mode

### `dots work on`

Rewire links to a local checkout for development.

```bash
dots work on personal ~/code/dotfiles
```

### `dots work off`

Rewire links back to the internal clone.

```bash
dots work off personal
```

### `dots work status`

Show which taps are in work mode.

```bash
dots work status
```

### `dots work rebuild`

Re-link packages after local changes.

```bash
dots work rebuild                    # rebuild all
dots work rebuild personal/nvim      # rebuild one package
```
