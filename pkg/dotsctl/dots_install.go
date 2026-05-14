package dotsctl

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jlrickert/dots/pkg/dots"
)

// InstallOptions configures the install operation.
type InstallOptions struct {
	// Package is the "<tap>/<package>" identifier to install.
	Package string
	// DryRun prints what would happen without writing.
	DryRun bool
	// Strategy overrides the link strategy for this install.
	Strategy dots.LinkStrategy
}

// InstallResult holds the result of an install operation.
type InstallResult struct {
	Package string
	Files   []dots.InstalledFile
	DryRun  bool
}

// Install installs a package by placing its links and recording state.
func (d *Dots) Install(ctx context.Context, opts InstallOptions) (*InstallResult, error) {
	tap, pkg, err := splitPackageRef(opts.Package)
	if err != nil {
		return nil, err
	}

	// Read and parse manifest
	manifestData, err := d.readManifest(ctx, tap, pkg)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}

	manifest, err := dots.ParseManifest(manifestData)
	if err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}

	platform := d.PathService.Platform
	if !manifest.SupportsCurrentPlatform(platform) {
		return nil, fmt.Errorf("package %s does not support platform %s", opts.Package, platform)
	}

	// Resolve manifest for current platform
	resolved := dots.ResolveManifest(manifest, platform)
	if err := resolved.ResolveSelfRefs(tap); err != nil {
		return nil, fmt.Errorf("resolve self refs: %w", err)
	}

	// Determine link strategy: flag > package manifest > work mode > global config > default
	strategy := d.resolveStrategy(tap, opts.Strategy, resolved.LinkStrategy)

	// Resolve link actions
	pkgDir := d.packageDir(tap, pkg)
	actions, err := dots.ResolveLinkActions(resolved, pkgDir, d.PathService.Resolver, strategy)
	if err != nil {
		return nil, fmt.Errorf("resolve links: %w", err)
	}

	if opts.DryRun {
		// Dry-run mirrors the real install flattening: directory entries
		// expand to per-leaf rows so the user sees exactly what would be
		// written. previewLinkLeaves is read-only — it stat-walks the
		// source but never opens or writes a destination file.
		files := previewLinkLeaves(actions)
		return &InstallResult{
			Package: opts.Package,
			Files:   files,
			DryRun:  true,
		}, nil
	}

	// Run pre_install hook
	hookRunner := d.hookRunner()
	if err := hookRunner.RunHook(ctx, resolved.Hooks.PreInstall, pkgDir); err != nil {
		return nil, fmt.Errorf("pre_install hook: %w", err)
	}

	// Place links, backing up existing files
	cfg, _ := d.ConfigService.Config(true)
	shouldBackup := cfg.Core.Backup == nil || *cfg.Core.Backup

	var installedFiles []dots.InstalledFile
	for _, action := range actions {
		if err := d.prepareDest(ctx, action.Dest, shouldBackup); err != nil {
			return nil, err
		}

		// PlaceLink now returns one or many results: a directory-copy
		// emits one InstalledFile per leaf so existing remove/diff/which
		// flows keep treating each as an independent unit.
		results, err := dots.PlaceLink(action)
		if err != nil {
			return nil, fmt.Errorf("place link: %w", err)
		}

		for _, result := range results {
			installedFiles = append(installedFiles, dots.InstalledFile{
				Src:      result.Src,
				Dest:     result.Dest,
				Method:   result.Method,
				Checksum: result.Checksum,
				Origin:   result.Origin,
			})
		}
	}

	// Run post_install hook
	if err := hookRunner.RunHook(ctx, resolved.Hooks.PostInstall, pkgDir); err != nil {
		return nil, fmt.Errorf("post_install hook: %w", err)
	}

	// Record in lockfile
	if err := d.recordInstall(ctx, tap, pkg, strategy, resolved, installedFiles); err != nil {
		return nil, fmt.Errorf("update lockfile: %w", err)
	}

	return &InstallResult{
		Package: opts.Package,
		Files:   installedFiles,
	}, nil
}

func (d *Dots) recordInstall(
	ctx context.Context,
	tap, pkg string,
	strategy dots.LinkStrategy,
	resolved *dots.ResolvedManifest,
	files []dots.InstalledFile,
) error {
	lockfile, err := d.Repo.ReadLockfile(ctx)
	if err != nil {
		if !isNotExist(err) {
			return err
		}
		lockfile = &dots.Lockfile{}
	}

	pkgRef := tap + "/" + pkg
	entry := dots.InstalledPackage{
		Package:      pkgRef,
		Tap:          tap,
		Version:      resolved.Package.Version,
		Type:         "base",
		LinkStrategy: strategy,
		Files:        files,
	}

	// Replace existing entry or append
	found := false
	for i, existing := range lockfile.Installed {
		if existing.Package == pkgRef {
			lockfile.Installed[i] = entry
			found = true
			break
		}
	}
	if !found {
		lockfile.Installed = append(lockfile.Installed, entry)
	}

	lockfile.State.LastApplied = time.Now()
	lockfile.State.Platform = d.PathService.Platform.String()
	lockfile.State.LinkStrategy = strategy

	return d.Repo.WriteLockfile(ctx, lockfile)
}

// workModePath resolves the local checkout path for tap, preferring the new
// work state file. If the tap is not in state, falls back to the legacy
// config.WorkMode map for read-only compatibility (e.g. when migration has
// not yet been triggered by a write). Returns ("", false) when tap is not in
// work mode at all.
//
// A corrupt work state file is surfaced as a stderr warning rather than
// returned as an error (the install path keeps working from legacy/default);
// `dots doctor` reports the parse failure as an error-level check via
// checkWorkStateFile.
func (d *Dots) workModePath(tap string) (string, bool) {
	if d.WorkStateService != nil {
		state, err := d.WorkStateService.Load(true)
		if err != nil {
			fmt.Fprintf(d.Runtime.Stream().Err,
				"warning: work state file unreadable: %v (run `dots doctor` for details)\n", err)
		} else if state != nil {
			if path, ok := state.Taps[tap]; ok {
				return path, true
			}
		}
	}
	cfg, _ := d.ConfigService.Config(true)
	if cfg != nil {
		if path, ok := cfg.WorkMode[tap]; ok {
			return path, true
		}
	}
	return "", false
}

// resolveStrategy picks the link strategy for a tap install.
// Precedence: explicit override > package manifest > work mode > config > default.
// Work mode is treated as "live editing" — when the tap is in work mode, default
// to symlink so edits in the local checkout propagate without `dots sync`.
func (d *Dots) resolveStrategy(tap string, override, pkgStrategy dots.LinkStrategy) dots.LinkStrategy {
	if override != "" {
		return override
	}
	if pkgStrategy != "" {
		return pkgStrategy
	}
	if _, ok := d.workModePath(tap); ok {
		return dots.LinkSymlink
	}
	cfg, _ := d.ConfigService.Config(true)
	if cfg != nil {
		core := cfg.ResolveCorePlatform(d.PathService.Platform)
		if core.LinkStrategy != "" {
			return core.LinkStrategy
		}
	}
	return dots.LinkCopy
}

// readManifest reads a package manifest, checking work mode paths first.
// Work mode paths are real user-filesystem paths (from `dots work on <tap> <path>`)
// and must be read directly via os.ReadFile — the runtime's jail applies only to
// dots-managed state, not to arbitrary checkouts the user has declared.
func (d *Dots) readManifest(ctx context.Context, tap, pkg string) ([]byte, error) {
	if localPath, ok := d.workModePath(tap); ok {
		manifestPath := filepath.Join(localPath, pkg, "Dotfile.yaml")
		data, err := os.ReadFile(manifestPath)
		if err != nil {
			return nil, &dots.PackageNotFoundError{Tap: tap, Package: pkg}
		}
		return data, nil
	}
	return d.Repo.ReadManifest(ctx, tap, pkg)
}

// listPackages lists packages in a tap, checking work mode paths first.
func (d *Dots) listPackages(ctx context.Context, tap string) ([]dots.PackageInfo, error) {
	if localPath, ok := d.workModePath(tap); ok {
		return dots.ScanPackages(tap, localPath)
	}
	return d.Repo.ListPackages(ctx, tap)
}

func (d *Dots) packageDir(tap, pkg string) string {
	if localPath, ok := d.workModePath(tap); ok {
		return filepath.Join(localPath, pkg)
	}
	return filepath.Join(d.PathService.TapsDir(), tap, pkg)
}

func (d *Dots) hookRunner() *dots.HookRunner {
	streams := d.Runtime.Stream()
	return &dots.HookRunner{
		Stdout: streams.Out,
		Stderr: streams.Err,
	}
}

// prepareDest clears dest in preparation for placing a new link. Regular
// files and symlinks are backed up (when enabled) and removed; empty
// directories are removed. A non-empty directory is treated as a user-data
// conflict and aborts before any destructive action.
//
// Filesystem inspection here uses os.Lstat / os.RemoveAll directly (not
// d.Runtime.*) for two reasons:
//
//   - Symlink discrimination. os.Lstat does not follow symlinks; this is
//     load-bearing for the upgrade path. A previously installed symlink-dir
//     would otherwise be Stat-followed into the source package, causing
//     prepareDest to ReadDir into source or report a spurious "non-empty
//     directory at dest" error.
//   - Path consistency with the placer. The downstream PlaceLink path uses
//     raw os.Symlink / os.OpenFile against action.Dest, so prepareDest must
//     inspect and clear that same physical path. Routing through the runtime
//     would re-jail an already-jail-prefixed alias path in tests and silently
//     no-op against a different location than PlaceLink writes to.
//
// Backups are still routed through the runtime — they're purely virtual-
// state writes and the runtime is the right seam for them.
func (d *Dots) prepareDest(ctx context.Context, dest string, shouldBackup bool) error {
	info, err := os.Lstat(dest)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("stat %s: %w", dest, err)
	}

	// Symlink at dest: remove it without inspecting its target. The
	// short-circuit before the IsDir branch is required for dir-symlinks
	// — descending would walk into the source package (OsFS readdir
	// follows symlinks), so without this we'd either stomp source or
	// abort with a misleading "non-empty" error.
	if info.Mode()&os.ModeSymlink != 0 {
		if err := os.Remove(dest); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove existing %s: %w", dest, err)
		}
		return nil
	}

	if info.IsDir() {
		entries, err := os.ReadDir(dest)
		if err != nil {
			return fmt.Errorf("read %s: %w", dest, err)
		}
		if len(entries) > 0 {
			return fmt.Errorf("destination %s is a non-empty directory; move or remove it to proceed", dest)
		}
		if err := os.Remove(dest); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove existing %s: %w", dest, err)
		}
		return nil
	}

	if shouldBackup {
		if data, err := os.ReadFile(dest); err == nil {
			_ = d.Repo.BackupFile(ctx, dest, data)
		}
	}

	if err := os.Remove(dest); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove existing %s: %w", dest, err)
	}
	return nil
}

// previewLinkLeaves expands a slice of LinkActions into the flat
// InstalledFile rows that PlaceLink would emit, without writing anything.
// File actions become a single row keyed by Strategy. Directory actions
// are resolved through the same Mode/Strategy table as PlaceLink:
//
//   - mode=symlink (or auto + symlink/hardlink strategy) → one
//     "symlink-dir" row.
//   - mode=copy (or auto + copy strategy) → one "copy-dir-leaf" row per
//     regular file under the source, with Exclude applied.
//
// Best-effort: a missing or unreadable source surfaces as a single row
// describing the action so dry-run still says something useful instead of
// failing.
func previewLinkLeaves(actions []dots.LinkAction) []dots.InstalledFile {
	var files []dots.InstalledFile
	for _, a := range actions {
		if !a.IsDir {
			files = append(files, dots.InstalledFile{
				Src:    a.Src,
				Dest:   a.Dest,
				Method: string(a.Strategy),
				Origin: a.Origin,
			})
			continue
		}

		mode := a.Mode
		if mode == dots.LinkModeAuto {
			switch a.Strategy {
			case dots.LinkCopy:
				mode = dots.LinkModeCopy
			default:
				mode = dots.LinkModeSymlink
			}
		}

		if mode == dots.LinkModeSymlink {
			files = append(files, dots.InstalledFile{
				Src:    a.Src,
				Dest:   a.Dest,
				Method: "symlink-dir",
				Origin: a.Origin,
			})
			continue
		}

		// LinkModeCopy: enumerate leaves the way placeDirCopy will.
		leaves, err := previewCopyDirLeaves(a)
		if err != nil || len(leaves) == 0 {
			// Fall back to a single placeholder row so dry-run is never
			// silent. Users can still see the dest root.
			files = append(files, dots.InstalledFile{
				Src:    a.Src,
				Dest:   a.Dest,
				Method: "copy-dir-leaf",
				Origin: a.Origin,
			})
			continue
		}
		files = append(files, leaves...)
	}
	return files
}

// previewCopyDirLeaves walks the source directory and returns one
// InstalledFile per regular file leaf that placeDirCopy would copy. It
// applies the same exclude semantics so the dry-run row count matches the
// real install row count.
func previewCopyDirLeaves(a dots.LinkAction) ([]dots.InstalledFile, error) {
	var leaves []dots.InstalledFile
	err := filepath.WalkDir(a.Src, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(a.Src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}
		if d.IsDir() {
			if dots.MatchExclude(rel, a.Exclude) {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		if dots.MatchExclude(rel, a.Exclude) {
			return nil
		}
		leaves = append(leaves, dots.InstalledFile{
			Src:    path,
			Dest:   filepath.Join(a.Dest, rel),
			Method: "copy-dir-leaf",
			Origin: a.Origin,
		})
		return nil
	})
	return leaves, err
}

// splitPackageRef splits "tap/package" into (tap, package).
func splitPackageRef(ref string) (string, string, error) {
	parts := strings.SplitN(ref, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid package reference %q: expected <tap>/<package>", ref)
	}
	return parts[0], parts[1], nil
}
