# Work Mode

Work mode lets you develop dotfile packages by pointing dots at a local Git checkout instead of its internal clone. This way, edits to your local repo are immediately reflected in your installed dotfiles (when using symlink strategy).

## Typical Workflow

```bash
# Clone your dotfiles repo for editing
git clone git@github.com:you/dotfiles.git ~/code/dotfiles

# Enable work mode for the tap
dots work on personal ~/code/dotfiles

# Edit files in ~/code/dotfiles — changes appear immediately
vim ~/code/dotfiles/nvim/init.lua

# When done, commit and push your changes
cd ~/code/dotfiles
git add -A && git commit -m "update nvim config"
git push

# Disable work mode to switch back to the internal clone
dots work off personal
```

## Commands

### `dots work on`

Enable work mode for a tap, pointing it at a local directory.

```bash
dots work on <tap> <local-path>
```

This updates the `work_mode` section in your config:

```yaml
work_mode:
  personal: /Users/you/code/dotfiles
```

All installed packages from this tap will now resolve their source files from the local path.

### `dots work off`

Disable work mode for a tap, reverting to the internal clone.

```bash
dots work off <tap>
```

### `dots work status`

Show which taps are currently in work mode.

```bash
dots work status
```

Output: `TAP -> LOCAL_PATH` per active work mode, or "No taps in work mode".

### `dots work rebuild`

Re-link packages after making structural changes (adding/removing files or links in a Dotfile.yaml).

```bash
dots work rebuild                    # rebuild all work-mode packages
dots work rebuild personal/nvim      # rebuild a specific package
```

Rebuild is needed when you change the `links` section of a Dotfile.yaml in your local checkout. Simple content edits to already-linked files don't require a rebuild.

## Behavior by Link Strategy

| Strategy | Effect in work mode |
|----------|-------------------|
| `symlink` | Symlinks point to local checkout — edits propagate instantly |
| `hardlink` | Hard links to local files — edits propagate instantly |
| `copy` | Files were copied at install time — use `dots sync` to update |

For the best work mode experience, use `symlink` strategy so changes are reflected immediately without any sync step.

## Integration with `dots sync`

If you're using `copy` strategy, work mode alone doesn't make edits propagate. After editing files in your local checkout, run:

```bash
dots sync personal/nvim      # sync specific package
dots sync --all              # sync all copy-strategy packages
```

This re-copies the updated source files to their target locations.
