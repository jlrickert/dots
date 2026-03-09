package dotsctl

import (
	"context"
	"fmt"

	"github.com/jlrickert/dots/pkg/dots"
)

// RemoveOptions configures the remove operation.
type RemoveOptions struct {
	// Package is the "<tap>/<package>" identifier to remove.
	Package string
}

// Remove uninstalls a package by removing its placed links and restoring backups.
func (d *Dots) Remove(ctx context.Context, opts RemoveOptions) error {
	tap, pkg, err := splitPackageRef(opts.Package)
	if err != nil {
		return err
	}

	lockfile, err := d.Repo.ReadLockfile(ctx)
	if err != nil {
		return fmt.Errorf("read lockfile: %w", err)
	}

	pkgRef := tap + "/" + pkg
	var installed *dots.InstalledPackage
	var idx int
	for i, p := range lockfile.Installed {
		if p.Package == pkgRef {
			installed = &lockfile.Installed[i]
			idx = i
			break
		}
	}
	if installed == nil {
		return &dots.PackageNotFoundError{Tap: tap, Package: pkg}
	}

	// Run pre_remove hook if manifest is available
	pkgDir := d.packageDir(tap, pkg)
	manifestData, _ := d.Repo.ReadManifest(ctx, tap, pkg)
	if manifestData != nil {
		manifest, err := dots.ParseManifest(manifestData)
		if err == nil {
			resolved := dots.ResolveManifest(manifest, d.PathService.Platform)
			hookRunner := d.hookRunner()
			if err := hookRunner.RunHook(ctx, resolved.Hooks.PreRemove, pkgDir); err != nil {
				return fmt.Errorf("pre_remove hook: %w", err)
			}
		}
	}

	// Remove placed files and restore backups
	cfg, _ := d.ConfigService.Config(true)
	shouldBackup := cfg.Core.Backup == nil || *cfg.Core.Backup

	for _, f := range installed.Files {
		_ = dots.RemoveLink(f.Dest)

		if shouldBackup {
			if data, err := d.Repo.RestoreFile(ctx, f.Dest); err == nil {
				_ = dots.RestoreFileFromBackup(f.Dest, data)
			}
		}
	}

	// Run post_remove hook
	if manifestData != nil {
		manifest, err := dots.ParseManifest(manifestData)
		if err == nil {
			resolved := dots.ResolveManifest(manifest, d.PathService.Platform)
			hookRunner := d.hookRunner()
			_ = hookRunner.RunHook(ctx, resolved.Hooks.PostRemove, pkgDir)
		}
	}

	// Remove from lockfile
	lockfile.Installed = append(lockfile.Installed[:idx], lockfile.Installed[idx+1:]...)
	return d.Repo.WriteLockfile(ctx, lockfile)
}
