package dotsctl

import (
	"context"
	"fmt"

	"github.com/jlrickert/dots/pkg/dots"
)

// SyncOptions configures the sync operation.
type SyncOptions struct {
	// Package syncs a specific package. Empty with All=true syncs everything.
	Package string
	// All syncs all copy-strategy packages.
	All bool
}

// SyncResult holds the result of a sync operation.
type SyncResult struct {
	Updated []string
	Skipped []string
}

// Sync re-copies changed files for packages using the copy strategy.
func (d *Dots) Sync(ctx context.Context, opts SyncOptions) (*SyncResult, error) {
	lockfile, err := d.Repo.ReadLockfile(ctx)
	if err != nil {
		if isNotExist(err) {
			return &SyncResult{}, nil
		}
		return nil, err
	}

	result := &SyncResult{}

	for _, pkg := range lockfile.Installed {
		if pkg.LinkStrategy != dots.LinkCopy {
			continue
		}
		if !opts.All && opts.Package != "" && pkg.Package != opts.Package {
			continue
		}

		for _, f := range pkg.Files {
			if f.Method != "copy" {
				continue
			}

			srcChecksum, err := dots.FileChecksum(f.Src)
			if err != nil {
				result.Skipped = append(result.Skipped, f.Dest)
				continue
			}

			if srcChecksum == f.Checksum {
				result.Skipped = append(result.Skipped, f.Dest)
				continue
			}

			// Re-copy the file
			_, err = dots.PlaceLink(dots.LinkAction{
				Src:      f.Src,
				Dest:     f.Dest,
				Strategy: dots.LinkCopy,
			})
			if err != nil {
				return nil, fmt.Errorf("sync %s: %w", f.Dest, err)
			}
			result.Updated = append(result.Updated, f.Dest)
		}
	}

	return result, nil
}
