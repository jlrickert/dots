package dotsctl

import (
	"context"
	"fmt"
)

// ReinstallOptions configures the reinstall operation.
type ReinstallOptions struct {
	// Package is the "<tap>/<package>" identifier to reinstall.
	Package string
}

// Reinstall removes and then installs a package.
func (d *Dots) Reinstall(ctx context.Context, opts ReinstallOptions) (*InstallResult, error) {
	// Remove existing installation (ignore not-found)
	_ = d.Remove(ctx, RemoveOptions{Package: opts.Package})

	result, err := d.Install(ctx, InstallOptions{Package: opts.Package})
	if err != nil {
		return nil, fmt.Errorf("reinstall %s: %w", opts.Package, err)
	}
	return result, nil
}
