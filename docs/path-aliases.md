# Path Aliases

Path aliases provide cross-platform portability for link targets in [Dotfile.yaml](dotfile-yaml.md) manifests. Instead of hardcoding OS-specific paths, you write `@config/nvim/init.lua` and dots resolves it to the correct location on each platform.

## Built-in Aliases

| Alias | macOS / Linux | Windows |
|-------|---------------|---------|
| `@home` | `$HOME` | `%USERPROFILE%` |
| `@config` | `$XDG_CONFIG_HOME` (default: `~/.config`) | `%APPDATA%` (default: `~/AppData/Roaming`) |
| `@data` | `$XDG_DATA_HOME` (default: `~/.local/share`) | `%LOCALAPPDATA%` (default: `~/AppData/Local`) |
| `@cache` | `$XDG_CACHE_HOME` (default: `~/.cache`) | `%LOCALAPPDATA%/cache` |
| `@state` | `$XDG_STATE_HOME` (default: `~/.local/state`) | `%LOCALAPPDATA%/state` |
| `@bin` | `~/.local/bin` | `%LOCALAPPDATA%/bin` |

On Unix systems, dots respects XDG environment variables when set. When unset, it falls back to the standard XDG default directories.

On Windows, `@config` maps to `%APPDATA%` (Roaming) while `@data`, `@cache`, `@state`, and `@bin` map to subdirectories of `%LOCALAPPDATA%` (Local).

## Custom Aliases

Define custom aliases in your [config.yaml](config-yaml.md) under the `aliases` key:

```yaml
aliases:
  "@nvim": "@config/nvim"
  "@scripts": "@home/scripts"
  "@dots": "@config/dots"
```

### Chaining

Custom aliases can reference other aliases, including built-in ones. dots resolves them recursively:

```yaml
aliases:
  "@nvim": "@config/nvim"
  "@nvim-lua": "@nvim/lua"
```

`@nvim-lua/utils.lua` resolves to `~/.config/nvim/lua/utils.lua` on Unix.

Custom aliases are checked before built-in aliases, so you can use any `@name` that doesn't conflict with the six built-in names.

## Raw Paths

Paths without an `@` prefix are treated as relative to the home directory:

```yaml
links:
  .gitconfig: .gitconfig          # -> $HOME/.gitconfig
  .bashrc: .bashrc                # -> $HOME/.bashrc
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
@config/nvim/init.lua  -> /Users/you/.config/nvim/init.lua
@data/dots/state.json  -> /Users/you/.local/share/dots/state.json
@bin/my-script         -> /Users/you/.local/bin/my-script
@home/.bashrc          -> /Users/you/.bashrc
.gitconfig             -> /Users/you/.gitconfig
```

### Linux (custom XDG)

With `XDG_CONFIG_HOME=/opt/config`:

```
@config/nvim/init.lua  -> /opt/config/nvim/init.lua
@data/dots/state.json  -> /home/you/.local/share/dots/state.json
```

### Windows

```
@config/nvim/init.lua  -> C:\Users\you\AppData\Roaming\nvim\init.lua
@data/dots/state.json  -> C:\Users\you\AppData\Local\dots\state.json
@bin/my-script.exe     -> C:\Users\you\AppData\Local\bin\my-script.exe
@home/.bashrc          -> C:\Users\you\.bashrc
```
