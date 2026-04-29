package dotsctl

import (
	"context"
	"fmt"
)

// ImplodeOptions configures the implode operation.
type ImplodeOptions struct {
	// Yes confirms the destructive operation. Implode refuses to run without it.
	Yes bool
}

// ImplodeResult reports what happened during implode.
type ImplodeResult struct {
	// Uninstalled holds the package refs (tap/pkg) successfully uninstalled.
	Uninstalled []string
	// Failed holds package refs whose uninstall returned an error; the
	// corresponding errors are joined into the method's returned error.
	Failed []string
	// StateDirRemoved is true when the dots state directory existed and was removed.
	StateDirRemoved bool
	// ConfigDirRemoved is true when the dots config directory existed and was removed.
	ConfigDirRemoved bool
}

// Implode uninstalls every package recorded in the lockfile and removes the
// dots-managed config and state directories. The binary itself is left alone.
// Per-package uninstall failures do not abort the operation — directory
// removal still proceeds, and the returned error aggregates per-package
// failures.
func (d *Dots) Implode(ctx context.Context, opts ImplodeOptions) (*ImplodeResult, error) {
	if !opts.Yes {
		return nil, fmt.Errorf("implode is destructive; pass --yes to confirm")
	}

	result := &ImplodeResult{}

	lockfile, err := d.Repo.ReadLockfile(ctx)
	if err != nil && !isNotExist(err) {
		return nil, fmt.Errorf("read lockfile: %w", err)
	}
	if lockfile != nil {
		// Snapshot refs first — Remove rewrites the lockfile on each call.
		var refs []string
		for _, pkg := range lockfile.Installed {
			refs = append(refs, pkg.Package)
		}
		for _, ref := range refs {
			if err := d.Remove(ctx, RemoveOptions{Package: ref}); err != nil {
				result.Failed = append(result.Failed, ref)
				continue
			}
			result.Uninstalled = append(result.Uninstalled, ref)
		}
	}

	stateDir := d.PathService.StateDir()
	if _, err := d.Runtime.Stat(stateDir, false); err == nil {
		if err := d.Runtime.Remove(stateDir, true); err != nil {
			return result, fmt.Errorf("remove state dir: %w", err)
		}
		result.StateDirRemoved = true
	}

	configDir := d.PathService.ConfigDir()
	if _, err := d.Runtime.Stat(configDir, false); err == nil {
		if err := d.Runtime.Remove(configDir, true); err != nil {
			return result, fmt.Errorf("remove config dir: %w", err)
		}
		result.ConfigDirRemoved = true
	}

	d.ConfigService.InvalidateCache()

	if len(result.Failed) > 0 {
		return result, fmt.Errorf("imploded with %d uninstall error(s)", len(result.Failed))
	}
	return result, nil
}
