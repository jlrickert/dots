package dotsctl

import (
	"context"
	"fmt"

	"github.com/jlrickert/dots/pkg/dots"
)

// ProfileApplyOptions configures the profile apply operation.
type ProfileApplyOptions struct {
	// Name is the profile to apply.
	Name string
	// DryRun computes the plan without removing or installing anything.
	DryRun bool
}

// ProfileApplyResult describes what apply removed, installed, and kept.
type ProfileApplyResult struct {
	Profile string
	// Remove lists installed packages not in the resolved profile.
	Remove []string
	// Install lists resolved packages that were not installed.
	Install []string
	// Keep lists packages that were installed and remain in the profile.
	Keep   []string
	DryRun bool
}

// ProfileApply makes the installed packages match a profile exactly: it
// uninstalls every installed package not in the profile's resolved list
// (plus core.always_installed), installs the missing ones, and records the
// profile as active in the lockfile. Packages already installed and still in
// the list are left untouched.
func (d *Dots) ProfileApply(ctx context.Context, opts ProfileApplyOptions) (*ProfileApplyResult, error) {
	resolved, err := d.resolveProfileForApply(opts.Name)
	if err != nil {
		return nil, err
	}

	lockfile, err := d.Repo.ReadLockfile(ctx)
	if err != nil && !isNotExist(err) {
		return nil, fmt.Errorf("read lockfile: %w", err)
	}

	installed := make(map[string]struct{})
	if lockfile != nil {
		for _, p := range lockfile.Installed {
			installed[p.Package] = struct{}{}
		}
	}
	wanted := make(map[string]struct{}, len(resolved))
	for _, p := range resolved {
		wanted[p.Ref] = struct{}{}
	}

	result := &ProfileApplyResult{Profile: opts.Name, DryRun: opts.DryRun}
	if lockfile != nil {
		for _, p := range lockfile.Installed {
			if _, ok := wanted[p.Package]; !ok {
				result.Remove = append(result.Remove, p.Package)
			}
		}
	}
	for _, p := range resolved {
		if _, ok := installed[p.Ref]; ok {
			result.Keep = append(result.Keep, p.Ref)
		} else {
			result.Install = append(result.Install, p.Ref)
		}
	}

	if opts.DryRun {
		return result, nil
	}

	// Remove first so packages that conflict with the new profile are gone
	// before anything is installed in their place.
	for _, ref := range result.Remove {
		if err := d.Remove(ctx, RemoveOptions{Package: ref}); err != nil {
			return result, fmt.Errorf("remove %s: %w", ref, err)
		}
	}
	for _, ref := range result.Install {
		if _, err := d.Install(ctx, InstallOptions{Package: ref}); err != nil {
			return result, fmt.Errorf("install %s: %w", ref, err)
		}
	}

	// Remove and Install rewrite the lockfile, so re-read before recording
	// the active profile.
	lockfile, err = d.Repo.ReadLockfile(ctx)
	if err != nil && !isNotExist(err) {
		return result, fmt.Errorf("read lockfile: %w", err)
	}
	if lockfile == nil {
		lockfile = &dots.Lockfile{}
	}
	lockfile.State.ActiveProfile = opts.Name
	lockfile.State.LastApplied = d.Runtime.Clock().Now()
	if err := d.Repo.WriteLockfile(ctx, lockfile); err != nil {
		return result, fmt.Errorf("write lockfile: %w", err)
	}
	return result, nil
}
