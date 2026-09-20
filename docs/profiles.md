# Profiles

Profiles are named collections of packages that can be applied together. They let you maintain different sets of dotfiles for different contexts (e.g., work vs personal machines) and move between them with a single command.

## Creating a Profile

```bash
dots profile create work
```

This creates an empty profile. Add packages to it:

```bash
dots profile add work personal/zsh personal/nvim personal/git personal/tmux
```

A profile can inherit from one or more other profiles at creation time:

```bash
dots profile create work --extends base --extends dev
```

See [Inheritance with `extends`](#inheritance-with-extends).

## Applying a Profile

```bash
dots profile apply work
```

`apply` makes the installed packages match the profile exactly:

1. Resolves the profile's full package list (following `extends` and `exclude`, plus [always-installed packages](#always-installed-packages)).
2. Uninstalls every installed package that is **not** in that list.
3. Installs every package in the list that isn't already installed.
4. Records the profile as the active profile in the lockfile.

Packages already installed and still in the list are left alone. Running `apply` again with the same profile is a no-op when nothing has changed; running it with a different profile is how you change profiles.

Preview the changes without touching anything:

```bash
dots profile apply work --dry-run
```

The dry run prints the packages that would be removed and installed.

> **Warning:** `apply` removes _every_ installed package that isn't in the profile, including packages installed by hand with `dots install`. Add anything you want to keep to the profile (or a profile it extends) before applying.

> **Tip:** Packages every profile needs, especially the `dots` package that manages your `config.yaml` and profiles, belong in [`core.always_installed`](#always-installed-packages) instead. Then no profile has to remember them, and `apply` never removes them.

## Managing Packages in Profiles

```bash
dots profile add work personal/docker          # add packages
dots profile remove work personal/docker       # remove a package

dots profile extend work dev                   # add parent profiles
dots profile unextend work dev                 # remove parent profiles

dots profile exclude work personal/tmux        # exclude inherited packages
dots profile unexclude work personal/tmux      # stop excluding them
```

Editing a profile does not change what's installed. Run `dots profile apply` to bring the machine in line with the updated profile.

## Viewing Profiles

```bash
dots profile list          # list all profile names
dots profile show work     # show details of a profile
```

`show` displays the profile name, its `extends` parents, its `exclude` list, and the fully resolved package list. Each resolved package is annotated with where it came from (the profile itself, the parent that contributed it, or `always_installed`). Excludes that don't match any inherited package are flagged so stale entries are easy to spot.

## Inheritance with `extends`

A profile can inherit packages from one or more parent profiles. `extends` accepts a single name or a list:

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/jlrickert/dots/main/schemas/profile.json
name: work
extends: [base, dev]
packages:
  - personal/slack
  - personal/docker
```

```yaml
name: laptop
extends: base # a single parent is also fine
```

### Resolution

A profile's resolved package list is built as follows:

1. Resolve each parent in `extends`, in the order listed. Parents are resolved recursively, and each parent's own `exclude` list has already been applied to its result.
2. Concatenate the parents' lists, keeping the first occurrence of each package.
3. Remove every package in this profile's `exclude` list.
4. Append this profile's own `packages`, skipping any already present.
5. When the profile is applied, prepend `core.always_installed` from config (see [Always-Installed Packages](#always-installed-packages)), skipping any already present. These can't be excluded.

Because deduplication keeps the first occurrence, a package shared by several parents (for example, two profiles that both extend `base`) installs once.

### Errors

- **Missing parent** — applying or showing a profile fails if any profile in its `extends` chain doesn't exist.
- **Cycles** — a profile that directly or indirectly extends itself is rejected, and the error names the cycle (e.g. `work → dev → work`).

### Example

```yaml
# base.yaml
name: base
packages: [personal/zsh, personal/git, personal/tmux]

# dev.yaml
name: dev
extends: base
packages: [personal/nvim, personal/go]

# work.yaml
name: work
extends: [base, dev]
packages: [work/zellij]
exclude: [personal/tmux]
```

Resolving `work`:

| Step                   | Packages                                                                   |
| ---------------------- | -------------------------------------------------------------------------- |
| `base`                 | zsh, git, tmux                                                             |
| `dev` (extends `base`) | zsh, git, tmux, nvim, go                                                   |
| Concatenate + dedupe   | zsh, git, tmux, nvim, go                                                   |
| Apply `exclude`        | zsh, git, nvim, go                                                         |
| Append own packages    | zsh, git, nvim, go, **zellij**                                             |
| Prepend always-installed (`personal/dots`) | **dots**, zsh, git, nvim, go, zellij                   |

## Excluding Inherited Packages

`exclude` removes packages that a profile would otherwise inherit from its parents. Use it to drop a package that conflicts with something in the child profile, without editing the shared parent:

```yaml
name: work
extends: base
packages:
  - work/zellij
exclude:
  - personal/tmux # base ships tmux; work uses zellij instead
```

Rules:

- `exclude` only affects **inherited** packages. A package listed in the profile's own `packages` is always installed, even if it also appears in `exclude`.
- Excludes are inherited like everything else: if `work` excludes `personal/tmux`, a profile that extends `work` won't receive `personal/tmux` from `work`. It can still get it from another parent, or list it in its own `packages`.
- Excluding a package that no parent provides is allowed and has no effect; `dots profile show` flags it.
- `exclude` never removes an [always-installed package](#always-installed-packages).

## Always-Installed Packages

Some packages belong on every machine no matter which profile is applied. The most important is the `dots` package that links your `config.yaml` and profile files (see [Bootstrapping](bootstrapping.md#the-dots-package-pattern)). If a profile left it out, applying that profile would uninstall the package that defines the profile.

List these in `core.always_installed` in [config.yaml](config-yaml.md#core-section):

```yaml
core:
  always_installed:
    - personal/dots
    - personal/zsh
```

Rules:

- Every `dots profile apply` includes these packages, whichever profile is applied. They are prepended to the resolved list in the order given.
- They override `exclude`: no profile can exclude an always-installed package.
- `apply` never uninstalls them. Removing one with `dots remove` works, but the next `apply` reinstalls it. To stop installing it, remove it from `core.always_installed`.
- Platform overrides add to the list, so a `platform.darwin.always_installed` entry is included only on macOS (see [config.yaml platform section](config-yaml.md#platform-section)).
- `dots profile show` and `apply --dry-run` label these packages as coming from `always_installed`.

Because the `dots` package usually ships `config.yaml` itself, its config can name the package, and every machine bootstrapped from it gets the protection automatically.

## Active Profile

The active profile is the one most recently applied. It is recorded in the lockfile under `state.active_profile`, not in `config.yaml`, so it stays local to each machine and never ends up in a tap:

| Platform      | Lockfile path                                                                      |
| ------------- | ---------------------------------------------------------------------------------- |
| macOS / Linux | `$XDG_STATE_HOME/dots/dots.lock.yaml` (default `~/.local/state/dots/dots.lock.yaml`) |
| Windows       | `%LOCALAPPDATA%\state\dots\dots.lock.yaml`                                         |

`dots status` reports the active profile.

## Export and Import

Export a profile to share it or back it up:

```bash
dots profile export work > work-profile.yaml
```

Import a profile from a YAML file:

```bash
dots profile import work-profile.yaml
```

Export writes the profile as defined (its `extends`, `exclude`, and own `packages`), not the resolved list. Importing a profile that extends others requires those parents to exist before it can be applied.

This is useful for:

- Sharing profiles between machines
- Backing up profile definitions
- Version-controlling profiles separately from taps

## Storage

Profiles are stored as YAML files in the dots config directory under `profiles/`:

| Platform      | Path                                  |
| ------------- | ------------------------------------- |
| macOS / Linux | `~/.config/dots/profiles/<name>.yaml` |
| Windows       | `%APPDATA%\dots\profiles\<name>.yaml` |

They are managed through the `dots profile` commands, but they are plain YAML and can also be shipped from a tap (see [Bootstrapping](bootstrapping.md#the-dots-package-pattern)).

When dots writes a profile, it prepends a `yaml-language-server` modeline pointing at `schemas/profile.json` so editors can offer schema validation and completion.

## Example Workflow

```bash
# personal/dots is listed in core.always_installed in config.yaml,
# so no profile needs to include it.

# Shared packages every machine gets
dots profile create base
dots profile add base personal/zsh personal/git personal/nvim personal/tmux

# Work machines: everything in base, plus work tools, minus tmux
dots profile create work --extends base
dots profile add work personal/docker work/zellij
dots profile exclude work personal/tmux

# Personal machines: everything in base, plus personal extras
dots profile create personal --extends base
dots profile add personal personal/gaming personal/media

# On a work machine
dots profile apply work --dry-run   # preview
dots profile apply work             # installs dots + base + work, minus tmux

# Later, repurpose the machine
dots profile apply personal         # removes docker/zellij, installs tmux/gaming/media
```

## Not Yet Specified

- **Install origin tracking.** dots does not yet record whether a package was installed by a profile or by hand, so `apply` treats them the same and removes anything outside the profile. A future option (e.g. `dots install --keep`) may mark packages that `apply` should leave in place.
