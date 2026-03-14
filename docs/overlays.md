# Overlays

Overlays let a package layer content on top of a base package. This is useful for machine-specific or context-specific customizations that extend a shared base configuration without modifying it.

## Declaring an Overlay

In the overlay package's `Dotfile.yaml`:

```yaml
package:
  name: zsh-work
  description: Work-specific zsh additions

overlay:
  base: personal/zsh
  strategy: append
  priority: 50
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `base` | string | yes | Base package reference (`tap/package`) |
| `strategy` | string | no | Default merge strategy (default: `append`) |
| `priority` | integer | no | 0-99, higher priority wins for `replace` and ordering (default: 0) |

## Merge Strategies

### `append` (default)

Overlay content is added after the base content, wrapped in marker comments:

```
<base content>
# --- overlay: work/zsh-work ---
<overlay content>
# --- overlay: work/zsh-work --- end
```

### `prepend`

Overlay content is added before the base content:

```
# --- overlay: work/zsh-work ---
<overlay content>
# --- overlay: work/zsh-work --- end
<base content>
```

### `replace`

Overlay content completely replaces the base content. The base is discarded:

```
<overlay content>
```

When multiple overlays use `replace`, the one with the highest `priority` wins.

### `merge`

Line-level deduplication merge. Base lines come first, then unique overlay lines are appended:

```
<base lines>
<overlay lines not already in base>
```

Duplicate lines (exact string match) from the overlay are skipped. This is useful for files like `.gitignore` where you want to add entries without duplicating existing ones.

## Per-File Strategy Overrides

The `merge` block in a manifest provides per-file strategy overrides:

```yaml
overlay:
  base: personal/zsh
  strategy: append

merge:
  .zshenv: replace
  .zshrc: prepend
```

Files not listed in the `merge` block use the overlay's default `strategy`.

## Priority System

When multiple overlays target the same base package, `priority` determines the order of application:

- Layers are sorted by priority ascending (lowest first)
- Higher priority layers are applied last, giving them the final say
- For `replace` strategy, the highest-priority overlay wins
- Valid range: 0-99

```yaml
# Low priority: applied first
overlay:
  base: personal/zsh
  strategy: append
  priority: 10

# High priority: applied last, overrides lower layers
overlay:
  base: personal/zsh
  strategy: append
  priority: 90
```

## Merged File Storage

Merged output files are stored in dots' state directory under a `merged/` tree organized by tap and package name:

```
<state-dir>/merged/<tap>/<package>/<filename>
```

The merged files are what actually get linked to the target location, not the raw base or overlay files.

## Marker Comments

For `append` and `prepend` strategies, dots inserts marker comments to identify overlay boundaries:

```
# --- overlay: work/zsh-work ---
# work-specific aliases
alias vpn="sudo openvpn /etc/work.ovpn"
# --- overlay: work/zsh-work --- end
```

These markers help identify which overlay contributed which content when inspecting merged files.

## Platform-Specific Overlays

The `overlay` block can appear inside a `platform` section, allowing platform-specific overlay behavior:

```yaml
platform:
  darwin:
    overlay:
      base: personal/zsh
      strategy: append
```

## Example: Layered Shell Configuration

**Base package** (`personal/zsh`):

```yaml
package:
  name: zsh
  tags: [shell]

links:
  .zshrc: "@home/.zshrc"
  .zshenv: "@home/.zshenv"
```

**Work overlay** (`work/zsh-work`):

```yaml
package:
  name: zsh-work
  tags: [shell, work]

overlay:
  base: personal/zsh
  strategy: append
  priority: 50

merge:
  .zshenv: replace

links:
  work-aliases.zsh: "@config/zsh/work-aliases.zsh"
```

When both are installed:

- `.zshrc` gets the base content with work overlay appended
- `.zshenv` is completely replaced by the work overlay version
- `work-aliases.zsh` is linked independently (not merged)
