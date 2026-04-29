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
		var files []dots.InstalledFile
		for _, a := range actions {
			files = append(files, dots.InstalledFile{
				Src:    a.Src,
				Dest:   a.Dest,
				Method: string(a.Strategy),
				Origin: a.Origin,
			})
		}
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

		result, err := dots.PlaceLink(action)
		if err != nil {
			return nil, fmt.Errorf("place link: %w", err)
		}

		installedFiles = append(installedFiles, dots.InstalledFile{
			Src:      result.Src,
			Dest:     result.Dest,
			Method:   result.Method,
			Checksum: result.Checksum,
			Origin:   result.Origin,
		})
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
// conflict and aborts before any destructive action. Paths are touched
// through d.Runtime so sandboxed callers stay inside their jail.
func (d *Dots) prepareDest(ctx context.Context, dest string, shouldBackup bool) error {
	info, err := d.Runtime.Stat(dest, false)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("stat %s: %w", dest, err)
	}

	if info.IsDir() {
		entries, err := d.Runtime.ReadDir(dest)
		if err != nil {
			return fmt.Errorf("read %s: %w", dest, err)
		}
		if len(entries) > 0 {
			return fmt.Errorf("destination %s is a non-empty directory; move or remove it to proceed", dest)
		}
	} else if shouldBackup {
		if data, err := d.Runtime.ReadFile(dest); err == nil {
			_ = d.Repo.BackupFile(ctx, dest, data)
		}
	}

	if err := d.Runtime.Remove(dest, true); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove existing %s: %w", dest, err)
	}
	return nil
}

// splitPackageRef splits "tap/package" into (tap, package).
func splitPackageRef(ref string) (string, string, error) {
	parts := strings.SplitN(ref, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid package reference %q: expected <tap>/<package>", ref)
	}
	return parts[0], parts[1], nil
}
