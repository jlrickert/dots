package dotsctl

import (
	"context"
	"fmt"
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
	cfg, err := d.ConfigService.Config(false)
	if err != nil {
		return err
	}

	if cfg.WorkMode == nil {
		cfg.WorkMode = make(map[string]string)
	}
	cfg.WorkMode[opts.Tap] = opts.LocalPath

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
	cfg, err := d.ConfigService.Config(false)
	if err != nil {
		return err
	}

	if _, ok := cfg.WorkMode[tap]; !ok {
		return fmt.Errorf("tap %q is not in work mode", tap)
	}
	delete(cfg.WorkMode, tap)

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

// WorkStatusList returns work mode status for all taps.
func (d *Dots) WorkStatusList(ctx context.Context) ([]WorkStatus, error) {
	cfg, err := d.ConfigService.Config(true)
	if err != nil {
		return nil, err
	}

	var statuses []WorkStatus
	for tap, path := range cfg.WorkMode {
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
