# Link Strategies

dots supports three strategies for placing configuration files from a tap package to their target locations. The strategy controls how the source file in the tap relates to the installed file on disk.

## Overview

| Strategy         | Mechanism                            | Edits propagate?     | Cross-filesystem?         |
| ---------------- | ------------------------------------ | -------------------- | ------------------------- |
| `copy` (default) | Independent copy of source to target | No (use `dots sync`) | Yes                       |
| `symlink`        | Symbolic link from target to source  | Instantly            | Yes                       |
| `hardlink`       | Hard link (same inode)               | Instantly            | No (same filesystem only) |

## Configuration Hierarchy

The effective link strategy for a package is determined by the first match in this priority order:

1. **Install-time override** (`dots install --strategy copy`)
2. **Per-package** (`link_strategy` in the `package` block of Dotfile.yaml)
3. **Work mode** — if the package's tap is in work mode (`dots work on <tap> <path>`), the strategy defaults to `symlink` so edits in the local checkout propagate without `dots sync`.
4. **Platform config** (`platform` section of config.yaml)
5. **Global config** (`core.link_strategy` in config.yaml)
6. **Default** (`copy`)

## Symlink

```yaml
core:
  link_strategy: symlink
```

Creates a symbolic link at the target path pointing to the source file in the tap.

```
~/.config/nvim/init.lua -> ~/.local/share/dots/taps/personal/nvim/init.lua
```

**Characteristics:**

- Edits to the target file immediately affect the source (they're the same file)
- Works across filesystems and partitions
- Visible as a symlink in `ls -la`
- Best for development workflows where you want changes reflected immediately
- Does not work on Windows without developer mode or elevated privileges

## Copy

```yaml
core:
  link_strategy: copy
```

Creates an independent copy of the source file at the target location. dots records a SHA-256 checksum at install time for change detection.

```
~/.config/nvim/init.lua  (copy of source, checksum: sha256:abc123...)
```

**Characteristics:**

- Target is a standalone file, not linked to the source
- Edits to the target do **not** propagate back to the source
- Use `dots sync` to push source changes to installed copies
- Works everywhere, including restricted Windows environments
- Checksum tracking detects drift between source and installed copies

### Syncing Copy-Strategy Packages

```bash
dots sync personal/nvim    # sync one package
dots sync --all            # sync all copy-strategy packages
```

`dots sync` re-copies source files to targets when the source has changed. It uses checksums to detect which files need updating.

## Hardlink

```yaml
core:
  link_strategy: hardlink
```

Creates a hard link (shared inode) between the source and target.

```
~/.config/nvim/init.lua  (same inode as source)
```

**Characteristics:**

- Edits propagate instantly (same underlying file data)
- Not visible as a link in `ls -la` (looks like a regular file)
- Both files must be on the **same filesystem** — fails if source and target are on different partitions
- Cannot link directories (only files)
- Survives deletion of the original path (data persists while any link exists)

## Windows Auto-Detection

On Windows, dots checks whether symbolic links are available (requires developer mode or administrator privileges). If symlinks are unavailable, dots automatically falls back to `copy` strategy regardless of configuration.

You can explicitly configure this in the platform section:

```yaml
platform:
  windows:
    link_strategy: copy
```

## Backup Behavior

When `core.backup` is `true` (the default) and a file already exists at the target path, dots backs up the existing file before replacing it. This applies to all three strategies.

The `core.conflict_strategy` setting controls what happens when a conflict is detected:

| Strategy    | Behavior                                |
| ----------- | --------------------------------------- |
| `prompt`    | Ask the user what to do                 |
| `overwrite` | Replace the existing file               |
| `skip`      | Leave the existing file in place        |
| `backup`    | Back up the existing file, then replace |

## Choosing a Strategy

| Use case                         | Recommended strategy |
| -------------------------------- | -------------------- |
| Development (frequent edits)     | `symlink`            |
| Shared/restricted machines       | `copy`               |
| Same-filesystem, invisible links | `hardlink`           |
| Windows without developer mode   | `copy`               |
| CI/deployment                    | `copy`               |
