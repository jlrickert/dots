package dotsctl

import (
	"context"
	"fmt"

	"github.com/jlrickert/cli-toolkit/toolkit"
)

// WorkOnOptions configures the work on operation.
type WorkOnOptions struct {
	Tap       string
	LocalPath string
}

// WorkStatus holds work mode status for a tap.
type WorkStatus struct {
	Tap       string
	LocalPath string
}

// WorkOn rewires links for a tap to point at a local checkout.
func (d *Dots) WorkOn(ctx context.Context, opts WorkOnOptions) error {
	localPath, err := toolkit.ExpandPath(d.Runtime, opts.LocalPath)
	if err != nil {
		return fmt.Errorf("expand local path: %w", err)
	}

	if err := d.migrateWorkModeIfNeeded(); err != nil {
		return fmt.Errorf("migrate legacy work_mode: %w", err)
	}

	if err := d.WorkStateService.Set(opts.Tap, localPath); err != nil {
		return fmt.Errorf("save work state: %w", err)
	}

	// Re-link all installed packages from this tap using local path
	lockfile, err := d.Repo.ReadLockfile(ctx)
	if err == nil {
		for _, pkg := range lockfile.Installed {
			if pkg.Tap == opts.Tap {
				_, _ = d.Reinstall(ctx, ReinstallOptions{Package: pkg.Package})
			}
		}
	}

	return nil
}

// WorkOff rewires links back to the internal clone.
func (d *Dots) WorkOff(ctx context.Context, tap string) error {
	if err := d.migrateWorkModeIfNeeded(); err != nil {
		return fmt.Errorf("migrate legacy work_mode: %w", err)
	}

	if _, ok := d.WorkStateService.Get(tap); !ok {
		return fmt.Errorf("tap %q is not in work mode", tap)
	}

	if err := d.WorkStateService.Delete(tap); err != nil {
		return fmt.Errorf("save work state: %w", err)
	}

	// Re-link packages from internal clone
	lockfile, err := d.Repo.ReadLockfile(ctx)
	if err == nil {
		for _, pkg := range lockfile.Installed {
			if pkg.Tap == tap {
				_, _ = d.Reinstall(ctx, ReinstallOptions{Package: pkg.Package})
			}
		}
	}

	return nil
}

// WorkStatusList returns work mode status for all taps. Reads from both the
// new state file and any legacy config.yaml work_mode entries, merging in
// memory for display. Does not trigger migration — migration happens only on
// the WorkOn/WorkOff write path so that read queries never mutate persistent
// state.
func (d *Dots) WorkStatusList(ctx context.Context) ([]WorkStatus, error) {
	taps, err := d.WorkStateService.All()
	if err != nil {
		return nil, fmt.Errorf("load work state: %w", err)
	}

	cfg, err := d.ConfigService.Config(true)
	if err == nil && cfg != nil {
		for tap, path := range cfg.WorkMode {
			if _, exists := taps[tap]; exists {
				continue
			}
			taps[tap] = path
		}
	}

	var statuses []WorkStatus
	for tap, path := range taps {
		statuses = append(statuses, WorkStatus{
			Tap:       tap,
			LocalPath: path,
		})
	}
	return statuses, nil
}

// Rebuild re-links all files for a package or all packages in a tap.
func (d *Dots) Rebuild(ctx context.Context, pkgRef string) error {
	if pkgRef == "" {
		// Rebuild all
		lockfile, err := d.Repo.ReadLockfile(ctx)
		if err != nil {
			if isNotExist(err) {
				return nil
			}
			return err
		}
		for _, pkg := range lockfile.Installed {
			if _, err := d.Reinstall(ctx, ReinstallOptions{Package: pkg.Package}); err != nil {
				return fmt.Errorf("rebuild %s: %w", pkg.Package, err)
			}
		}
		return nil
	}

	_, err := d.Reinstall(ctx, ReinstallOptions{Package: pkgRef})
	return err
}
