# Path Aliases

Path aliases provide cross-platform portability for link targets in [Dotfile.yaml](dotfile-yaml.md) manifests. Instead of hardcoding OS-specific paths, you write `@config/nvim/init.lua` and dots resolves it to the correct location on each platform.

## Built-in Aliases

dots ships three families of built-in aliases:

- **Default family** — sensible cross-platform defaults: XDG on Unix, native on Windows.
- **XDG family** (`@xdg-*`) — forces XDG paths everywhere, including Windows.
- **Apple family** (`@apple-*`) — Apple HIG locations under `~/Library`. Darwin only.

### Default family

| Alias     | macOS / Linux                                 | Windows                                       |
| --------- | --------------------------------------------- | --------------------------------------------- |
| `@home`   | `$HOME`                                       | `%USERPROFILE%`                               |
| `@config` | `$XDG_CONFIG_HOME` (default: `~/.config`)     | `%APPDATA%` (default: `~/AppData/Roaming`)    |
| `@data`   | `$XDG_DATA_HOME` (default: `~/.local/share`)  | `%LOCALAPPDATA%` (default: `~/AppData/Local`) |
| `@cache`  | `$XDG_CACHE_HOME` (default: `~/.cache`)       | `%LOCALAPPDATA%/cache`                        |
| `@state`  | `$XDG_STATE_HOME` (default: `~/.local/state`) | `%LOCALAPPDATA%/state`                        |
| `@bin`    | `~/.local/bin`                                | `%LOCALAPPDATA%/bin`                          |

On Unix systems, dots respects XDG environment variables when set. When unset, it falls back to the standard XDG default directories.

On Windows, `@config` maps to `%APPDATA%` (Roaming) while `@data`, `@cache`, `@state`, and `@bin` map to subdirectories of `%LOCALAPPDATA%` (Local).

### XDG family (`@xdg-*`)

The XDG family always resolves to XDG paths regardless of OS. This is the right choice for tools like Neovim that follow the XDG Base Directory spec on every platform — including Windows.

| Alias         | macOS                                         | Linux                                         | Windows                                       |
| ------------- | --------------------------------------------- | --------------------------------------------- | --------------------------------------------- |
| `@xdg-config` | `$XDG_CONFIG_HOME` (default: `~/.config`)     | `$XDG_CONFIG_HOME` (default: `~/.config`)     | `$XDG_CONFIG_HOME` (default: `~/.config`)     |
| `@xdg-data`   | `$XDG_DATA_HOME` (default: `~/.local/share`)  | `$XDG_DATA_HOME` (default: `~/.local/share`)  | `$XDG_DATA_HOME` (default: `~/.local/share`)  |
| `@xdg-cache`  | `$XDG_CACHE_HOME` (default: `~/.cache`)       | `$XDG_CACHE_HOME` (default: `~/.cache`)       | `$XDG_CACHE_HOME` (default: `~/.cache`)       |
| `@xdg-state`  | `$XDG_STATE_HOME` (default: `~/.local/state`) | `$XDG_STATE_HOME` (default: `~/.local/state`) | `$XDG_STATE_HOME` (default: `~/.local/state`) |

There is no `@xdg-bin`: the XDG Base Directory specification does not define a binary directory.

The Windows defaults explicitly use `~/.config`, `~/.local/share`, etc. — they do **not** fall back to `%APPDATA%` or `%LOCALAPPDATA%`. If you want native Windows directories, use the default family (`@config`, `@data`).

### Apple family (`@apple-*`, darwin only)

The Apple family resolves to subdirectories of `~/Library`, following Apple's Human Interface Guidelines. These aliases are useful for Mac-native locations like `LaunchAgents` that have no cross-platform equivalent.

| Alias                 | macOS                           | Linux / Windows |
| --------------------- | ------------------------------- | --------------- |
| `@apple-config`       | `~/Library/Application Support` | error           |
| `@apple-data`         | `~/Library/Application Support` | error           |
| `@apple-cache`        | `~/Library/Caches`              | error           |
| `@apple-logs`         | `~/Library/Logs`                | error           |
| `@apple-launchagents` | `~/Library/LaunchAgents`        | error           |

> **Apple HIG conflates config and data.** Both `@apple-config` and `@apple-data` resolve to the same directory (`~/Library/Application Support`). Apple's guidelines do not separate user configuration from application data; we surface both names so manifests can document intent.

#### Non-darwin behavior

On Linux, Windows, FreeBSD, or any non-darwin platform, every `@apple-*` alias returns an `*AliasUnavailableError` that wraps the sentinel `ErrAliasUnavailable`. The error message names the alias and the current OS. Callers can detect the condition with `errors.Is(err, dots.ErrAliasUnavailable)`.

This means a manifest that unconditionally references `@apple-launchagents` will fail on Linux. To avoid that, scope Apple-family aliases inside a `platform.darwin` block, or pair them with `package.platforms: [darwin]`. See [Platform System](platform-system.md).

## When to use which

- **Default family** (`@config`, `@data`, ...) — for portable configs that should "do the right thing" on each OS: XDG on Unix, native paths on Windows. This is the everyday choice.
- **XDG family** (`@xdg-*`) — when a tool insists on XDG everywhere. Neovim on Windows is the canonical example: it reads `~/.config/nvim` regardless of `%APPDATA%`.
- **Apple family** (`@apple-*`) — when you need a Mac-native location with no cross-platform equivalent (LaunchAgents, Application Support, Logs). Always scope these to `platform.darwin` or a darwin-only package.

## Custom Aliases

Define custom aliases in your [config.yaml](config-yaml.md) under the `aliases` key:

```yaml
aliases:
  "@nvim": "@config/nvim"
  "@scripts": "@home/scripts"
  "@dots": "@config/dots"
  "@launchd": "@apple-launchagents"
```

### Chaining

Custom aliases can reference other aliases, including any built-in from any family. dots resolves them recursively:

```yaml
aliases:
  "@nvim": "@config/nvim"
  "@nvim-lua": "@nvim/lua"
  "@launchd": "@apple-launchagents"
```

`@nvim-lua/utils.lua` resolves to `~/.config/nvim/lua/utils.lua` on Unix.
`@launchd/com.user.foo.plist` resolves to `~/Library/LaunchAgents/com.user.foo.plist` on darwin and errors on other platforms.

Custom aliases are checked before built-in aliases, so you can use any `@name` that doesn't conflict with the built-in names.

## Raw Paths

Paths without an `@` prefix are treated as relative to the home directory:

```yaml
links:
  .gitconfig: .gitconfig # -> $HOME/.gitconfig
  .bashrc: .bashrc # -> $HOME/.bashrc
```

This is equivalent to using `@home/.gitconfig`.

## Path Separator Normalization

Manifests always use forward slashes (`/`), regardless of the target platform. dots normalizes to platform-native separators (`\` on Windows) at resolution time using `filepath.ToSlash` / `filepath.FromSlash`.

```yaml
# Always write this (forward slashes):
links:
  init.lua: "@config/nvim/init.lua"

# Never write this (backslashes):
links:
  init.lua: "@config\\nvim\\init.lua"
```

## Resolution Examples

### macOS (default XDG)

```
@config/nvim/init.lua            -> /Users/you/.config/nvim/init.lua
@data/dots/state.json            -> /Users/you/.local/share/dots/state.json
@bin/my-script                   -> /Users/you/.local/bin/my-script
@home/.bashrc                    -> /Users/you/.bashrc
@apple-launchagents/foo.plist    -> /Users/you/Library/LaunchAgents/foo.plist
@apple-config/MyApp              -> /Users/you/Library/Application Support/MyApp
.gitconfig                       -> /Users/you/.gitconfig
```

### Linux (custom XDG)

With `XDG_CONFIG_HOME=/opt/config`:

```
@config/nvim/init.lua            -> /opt/config/nvim/init.lua
@xdg-config/nvim/init.lua        -> /opt/config/nvim/init.lua
@data/dots/state.json            -> /home/you/.local/share/dots/state.json
@apple-launchagents/foo.plist    -> error: alias @apple-launchagents is not available on linux
```

### Windows

```
@config/nvim/init.lua            -> C:\Users\you\AppData\Roaming\nvim\init.lua
@xdg-config/nvim/init.lua        -> C:\Users\you\.config\nvim\init.lua
@data/dots/state.json            -> C:\Users\you\AppData\Local\dots\state.json
@bin/my-script.exe               -> C:\Users\you\AppData\Local\bin\my-script.exe
@home/.bashrc                    -> C:\Users\you\.bashrc
@apple-launchagents/foo.plist    -> error: alias @apple-launchagents is not available on windows
```
