package dotsctl

import (
	"context"
	"fmt"

	"github.com/jlrickert/dots/pkg/dots"
)

// UpgradeOptions configures the upgrade operation.
type UpgradeOptions struct {
	// Package is the "<tap>/<package>" identifier to upgrade.
	// Empty with All=true means upgrade everything.
	Package string
	// All upgrades all installed packages.
	All bool
}

// Upgrade upgrades an installed package by updating the tap and re-linking.
func (d *Dots) Upgrade(ctx context.Context, opts UpgradeOptions) error {
	if opts.All {
		return d.upgradeAll(ctx)
	}

	if opts.Package == "" {
		return fmt.Errorf("specify a package or use --all")
	}

	tap, pkg, err := splitPackageRef(opts.Package)
	if err != nil {
		return err
	}

	// Update the tap
	if err := d.Repo.UpdateTap(ctx, tap); err != nil {
		return fmt.Errorf("update tap %s: %w", tap, err)
	}

	// Run pre_upgrade hook
	pkgDir := d.packageDir(tap, pkg)
	manifestData, err := d.readManifest(ctx, tap, pkg)
	if err != nil {
		return fmt.Errorf("read manifest: %w", err)
	}

	manifest, err := dots.ParseManifest(manifestData)
	if err != nil {
		return fmt.Errorf("parse manifest: %w", err)
	}

	resolved := dots.ResolveManifest(manifest, d.PathService.Platform)
	if err := resolved.ResolveSelfRefs(tap); err != nil {
		return fmt.Errorf("resolve self refs: %w", err)
	}
	hookRunner := d.hookRunner()
	if err := hookRunner.RunHook(ctx, resolved.Hooks.PreUpgrade, pkgDir); err != nil {
		return fmt.Errorf("pre_upgrade hook: %w", err)
	}

	// Remove old links then install new
	_ = d.Remove(ctx, RemoveOptions{Package: opts.Package})

	result, err := d.Install(ctx, InstallOptions{Package: opts.Package})
	if err != nil {
		return fmt.Errorf("reinstall after upgrade: %w", err)
	}
	_ = result

	// Run post_upgrade hook
	if err := hookRunner.RunHook(ctx, resolved.Hooks.PostUpgrade, pkgDir); err != nil {
		return fmt.Errorf("post_upgrade hook: %w", err)
	}

	return nil
}

func (d *Dots) upgradeAll(ctx context.Context) error {
	lockfile, err := d.Repo.ReadLockfile(ctx)
	if err != nil {
		if isNotExist(err) {
			return nil // nothing installed
		}
		return err
	}

	// Update all taps first
	taps, err := d.Repo.ListTaps(ctx)
	if err != nil {
		return err
	}
	for _, tap := range taps {
		if err := d.Repo.UpdateTap(ctx, tap.Name); err != nil {
			return fmt.Errorf("update tap %s: %w", tap.Name, err)
		}
	}

	// Upgrade each installed package
	for _, pkg := range lockfile.Installed {
		if err := d.Upgrade(ctx, UpgradeOptions{Package: pkg.Package}); err != nil {
			return fmt.Errorf("upgrade %s: %w", pkg.Package, err)
		}
	}

	return nil
}
