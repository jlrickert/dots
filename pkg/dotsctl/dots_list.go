package dotsctl

import (
	"context"

	"github.com/jlrickert/dots/pkg/dots"
)

// ListOptions configures the list operation.
type ListOptions struct {
	// Tap restricts listing to a specific tap. Empty means all taps.
	Tap string
	// Available lists available (not yet installed) packages.
	Available bool
}

// ListResult holds the result of a list operation.
type ListResult struct {
	Installed []dots.InstalledPackage
	Available []dots.PackageInfo
}

// List returns installed packages or available packages depending on options.
func (d *Dots) List(ctx context.Context, opts ListOptions) (*ListResult, error) {
	if opts.Available {
		return d.listAvailable(ctx, opts)
	}
	return d.listInstalled(ctx, opts)
}

func (d *Dots) listInstalled(ctx context.Context, opts ListOptions) (*ListResult, error) {
	lockfile, err := d.Repo.ReadLockfile(ctx)
	if err != nil {
		// No lockfile means nothing installed.
		if isNotExist(err) {
			return &ListResult{}, nil
		}
		return nil, err
	}

	installed := lockfile.Installed
	if opts.Tap != "" {
		filtered := make([]dots.InstalledPackage, 0)
		for _, pkg := range installed {
			if pkg.Tap == opts.Tap {
				filtered = append(filtered, pkg)
			}
		}
		installed = filtered
	}

	return &ListResult{Installed: installed}, nil
}

func (d *Dots) listAvailable(ctx context.Context, opts ListOptions) (*ListResult, error) {
	if opts.Tap != "" {
		pkgs, err := d.listPackages(ctx, opts.Tap)
		if err != nil {
			return nil, err
		}
		return &ListResult{Available: pkgs}, nil
	}

	taps, err := d.Repo.ListTaps(ctx)
	if err != nil {
		return nil, err
	}

	var all []dots.PackageInfo
	for _, tap := range taps {
		pkgs, err := d.listPackages(ctx, tap.Name)
		if err != nil {
			return nil, err
		}
		all = append(all, pkgs...)
	}
	return &ListResult{Available: all}, nil
}
