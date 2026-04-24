package dotsctl

import (
	"context"
	"errors"
	"fmt"

	"github.com/jlrickert/dots/pkg/dots"
)

// TapAddOptions configures the tap add operation.
type TapAddOptions struct {
	Name   string
	URL    string
	Branch string
}

// TapAdd registers a new tap.
func (d *Dots) TapAdd(ctx context.Context, opts TapAddOptions) error {
	if opts.Name == "" {
		return fmt.Errorf("tap name is required")
	}
	if opts.URL == "" {
		return fmt.Errorf("tap URL is required")
	}

	tap := dots.TapInfo{
		Name:   opts.Name,
		URL:    opts.URL,
		Branch: opts.Branch,
	}

	return d.Repo.AddTap(ctx, tap)
}

// TapRemoveResult reports what happened during a tap removal.
type TapRemoveResult struct {
	// Uninstalled holds the package refs (tap/pkg) that were successfully
	// uninstalled during the cascade.
	Uninstalled []string
	// Failed holds package refs whose uninstall returned an error; the
	// corresponding errors are joined into the method's returned error.
	Failed []string
	// TapExisted is true when the on-disk tap directory existed and was
	// removed. False when the lockfile held orphan entries for a tap that
	// was already gone.
	TapExisted bool
}

// TapRemove removes a registered tap after uninstalling every package installed
// from it. Lockfile entries for the tap are cleaned up even when the on-disk
// tap directory is already gone (orphan case). Per-package uninstall failures
// do not abort the operation — the tap is still removed, and the returned
// error aggregates the per-package failures.
func (d *Dots) TapRemove(ctx context.Context, name string) (*TapRemoveResult, error) {
	if name == "" {
		return nil, fmt.Errorf("tap name is required")
	}

	result := &TapRemoveResult{}

	lockfile, err := d.Repo.ReadLockfile(ctx)
	if err != nil && !isNotExist(err) {
		return nil, fmt.Errorf("read lockfile: %w", err)
	}
	if lockfile != nil {
		// Snapshot refs first — Remove rewrites the lockfile on each call.
		var refs []string
		for _, pkg := range lockfile.Installed {
			if pkg.Tap == name {
				refs = append(refs, pkg.Package)
			}
		}
		for _, ref := range refs {
			if err := d.Remove(ctx, RemoveOptions{Package: ref}); err != nil {
				result.Failed = append(result.Failed, ref)
				continue
			}
			result.Uninstalled = append(result.Uninstalled, ref)
		}
	}

	err = d.Repo.RemoveTap(ctx, name)
	switch {
	case err == nil:
		result.TapExisted = true
	case errors.Is(err, dots.ErrNotExist):
		// Orphan lockfile case — packages were pinned to a tap that's
		// already gone on disk. Uninstalls above already cleaned the
		// lockfile, so this is success from the caller's perspective.
	default:
		return result, fmt.Errorf("remove tap %s: %w", name, err)
	}

	if len(result.Failed) > 0 {
		return result, fmt.Errorf("tap %s removed with %d uninstall error(s)", name, len(result.Failed))
	}
	return result, nil
}

// TapList returns all registered taps.
func (d *Dots) TapList(ctx context.Context) ([]dots.TapInfo, error) {
	return d.Repo.ListTaps(ctx)
}

// TapUpdate fetches the latest state for a tap.
func (d *Dots) TapUpdate(ctx context.Context, name string) error {
	if name == "" {
		// Update all taps
		taps, err := d.Repo.ListTaps(ctx)
		if err != nil {
			return err
		}
		for _, tap := range taps {
			if err := d.Repo.UpdateTap(ctx, tap.Name); err != nil {
				return fmt.Errorf("update tap %s: %w", tap.Name, err)
			}
		}
		return nil
	}
	return d.Repo.UpdateTap(ctx, name)
}
