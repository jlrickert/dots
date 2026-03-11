package dotsctl

import (
	"context"
	"fmt"
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

	// Determine link strategy: flag > package manifest > global config > default
	strategy := d.resolveStrategy(opts.Strategy, resolved.LinkStrategy)

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
		// Backup existing file if it exists
		if shouldBackup {
			if data, err := dots.ReadFileForBackup(action.Dest); err == nil {
				_ = d.Repo.BackupFile(ctx, action.Dest, data)
			}
		}

		// Remove existing file/symlink if present
		_ = dots.RemoveLink(action.Dest)

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

func (d *Dots) resolveStrategy(override, pkgStrategy dots.LinkStrategy) dots.LinkStrategy {
	if override != "" {
		return override
	}
	if pkgStrategy != "" {
		return pkgStrategy
	}
	cfg, _ := d.ConfigService.Config(true)
	if cfg != nil {
		core := cfg.ResolveCorePlatform(d.PathService.Platform)
		if core.LinkStrategy != "" {
			return core.LinkStrategy
		}
	}
	return dots.LinkSymlink
}

// readManifest reads a package manifest, checking work mode paths first.
func (d *Dots) readManifest(ctx context.Context, tap, pkg string) ([]byte, error) {
	cfg, _ := d.ConfigService.Config(true)
	if cfg != nil {
		if localPath, ok := cfg.WorkMode[tap]; ok {
			manifestPath := localPath + "/" + pkg + "/Dotfile.yaml"
			data, err := d.Runtime.ReadFile(manifestPath)
			if err != nil {
				return nil, &dots.PackageNotFoundError{Tap: tap, Package: pkg}
			}
			return data, nil
		}
	}
	return d.Repo.ReadManifest(ctx, tap, pkg)
}

// listPackages lists packages in a tap, checking work mode paths first.
func (d *Dots) listPackages(ctx context.Context, tap string) ([]dots.PackageInfo, error) {
	cfg, _ := d.ConfigService.Config(true)
	if cfg != nil {
		if localPath, ok := cfg.WorkMode[tap]; ok {
			return dots.ScanPackages(tap, localPath)
		}
	}
	return d.Repo.ListPackages(ctx, tap)
}

func (d *Dots) packageDir(tap, pkg string) string {
	cfg, _ := d.ConfigService.Config(true)
	if cfg != nil {
		if localPath, ok := cfg.WorkMode[tap]; ok {
			return localPath + "/" + pkg
		}
	}
	return d.PathService.TapsDir() + "/" + tap + "/" + pkg
}

func (d *Dots) hookRunner() *dots.HookRunner {
	streams := d.Runtime.Stream()
	return &dots.HookRunner{
		Stdout: streams.Out,
		Stderr: streams.Err,
	}
}

// splitPackageRef splits "tap/package" into (tap, package).
func splitPackageRef(ref string) (string, string, error) {
	parts := strings.SplitN(ref, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid package reference %q: expected <tap>/<package>", ref)
	}
	return parts[0], parts[1], nil
}

