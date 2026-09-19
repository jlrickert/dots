package dotsctl

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"

	"github.com/jlrickert/dots/pkg/dots"
)

// ProfileCreateOptions configures profile creation.
type ProfileCreateOptions struct {
	// Name is the new profile's name.
	Name string
	// Extends lists parent profiles. Each must exist and must not create a
	// cycle.
	Extends []string
}

// ProfileCreate creates a new profile with no packages of its own.
func (d *Dots) ProfileCreate(ctx context.Context, opts ProfileCreateOptions) error {
	path := d.profilePath(opts.Name)
	if _, err := d.Runtime.Stat(path, false); err == nil {
		return dots.ErrExist
	}

	profile := &dots.Profile{
		Name:     opts.Name,
		Extends:  appendMissing(dots.StringList(nil), opts.Extends),
		Packages: []string{},
	}
	if _, _, err := d.resolveLoadedProfile(profile, nil); err != nil {
		return err
	}
	return d.saveProfile(profile)
}

// ProfileDelete removes a profile definition.
func (d *Dots) ProfileDelete(ctx context.Context, name string) error {
	path := d.profilePath(name)
	if _, err := d.Runtime.Stat(path, false); err != nil {
		if isNotExist(err) {
			return dots.ErrNotExist
		}
		return err
	}
	return d.Runtime.Remove(path, false)
}

// ProfileList returns all available profile names.
func (d *Dots) ProfileList(ctx context.Context) ([]string, error) {
	dir := d.PathService.ProfilesDir()
	entries, err := d.Runtime.ReadDir(dir)
	if err != nil {
		if isNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := filepath.Ext(e.Name())
		if ext == ".yaml" || ext == ".yml" {
			names = append(names, e.Name()[:len(e.Name())-len(ext)])
		}
	}
	return names, nil
}

// ProfileShowResult holds a profile definition and its resolution.
type ProfileShowResult struct {
	Profile *dots.Profile
	// Resolved is the list apply would install, including
	// core.always_installed, with the source of each package.
	Resolved []ResolvedPackage
	// UnmatchedExcludes lists excludes that match no inherited package.
	UnmatchedExcludes []string
}

// ProfileShow returns the profile definition and its resolved package list.
func (d *Dots) ProfileShow(ctx context.Context, name string) (*ProfileShowResult, error) {
	profile, err := d.loadProfile(name)
	if err != nil {
		return nil, fmt.Errorf("profile %q: %w", name, err)
	}
	pkgs, unmatched, err := d.resolveLoadedProfile(profile, nil)
	if err != nil {
		return nil, err
	}
	resolved, err := d.withAlwaysInstalled(pkgs)
	if err != nil {
		return nil, err
	}
	return &ProfileShowResult{
		Profile:           profile,
		Resolved:          resolved,
		UnmatchedExcludes: unmatched,
	}, nil
}

// ProfileAdd adds packages to a profile.
func (d *Dots) ProfileAdd(ctx context.Context, name string, packages []string) error {
	profile, err := d.loadProfile(name)
	if err != nil {
		return err
	}

	seen := make(map[string]struct{})
	for _, p := range profile.Packages {
		seen[p] = struct{}{}
	}
	for _, p := range packages {
		if _, ok := seen[p]; !ok {
			profile.Packages = append(profile.Packages, p)
			seen[p] = struct{}{}
		}
	}

	return d.saveProfile(profile)
}

// ProfileRemove removes a package from a profile.
func (d *Dots) ProfileRemove(ctx context.Context, name, pkg string) error {
	profile, err := d.loadProfile(name)
	if err != nil {
		return err
	}

	filtered := make([]string, 0, len(profile.Packages))
	for _, p := range profile.Packages {
		if p != pkg {
			filtered = append(filtered, p)
		}
	}
	profile.Packages = filtered

	return d.saveProfile(profile)
}

// ProfileExtend adds parent profiles to a profile's extends list. The change
// is rejected if a parent is missing or it would create a cycle.
func (d *Dots) ProfileExtend(ctx context.Context, name string, parents []string) error {
	return d.updateProfile(name, true, func(p *dots.Profile) {
		p.Extends = appendMissing(p.Extends, parents)
	})
}

// ProfileUnextend removes parent profiles from a profile's extends list.
func (d *Dots) ProfileUnextend(ctx context.Context, name string, parents []string) error {
	return d.updateProfile(name, false, func(p *dots.Profile) {
		p.Extends = removeAll(p.Extends, parents)
	})
}

// ProfileExclude adds packages to a profile's exclude list.
func (d *Dots) ProfileExclude(ctx context.Context, name string, packages []string) error {
	return d.updateProfile(name, false, func(p *dots.Profile) {
		p.Exclude = appendMissing(p.Exclude, packages)
	})
}

// ProfileUnexclude removes packages from a profile's exclude list.
func (d *Dots) ProfileUnexclude(ctx context.Context, name string, packages []string) error {
	return d.updateProfile(name, false, func(p *dots.Profile) {
		p.Exclude = removeAll(p.Exclude, packages)
	})
}

// updateProfile loads a profile, applies edit, optionally validates that the
// result still resolves, and saves it.
func (d *Dots) updateProfile(name string, validate bool, edit func(*dots.Profile)) error {
	profile, err := d.loadProfile(name)
	if err != nil {
		return fmt.Errorf("profile %q: %w", name, err)
	}
	edit(profile)
	if validate {
		if _, _, err := d.resolveLoadedProfile(profile, nil); err != nil {
			return err
		}
	}
	return d.saveProfile(profile)
}

// appendMissing appends each item not already in list, preserving order.
func appendMissing[S ~[]string](list S, items []string) S {
	for _, item := range items {
		if !slices.Contains(list, item) {
			list = append(list, item)
		}
	}
	return list
}

// removeAll returns list without any of items.
func removeAll[S ~[]string](list S, items []string) S {
	return slices.DeleteFunc(list, func(s string) bool {
		return slices.Contains(items, s)
	})
}

// ProfileExport returns the profile as YAML bytes.
func (d *Dots) ProfileExport(ctx context.Context, name string) ([]byte, error) {
	profile, err := d.loadProfile(name)
	if err != nil {
		return nil, err
	}
	return dots.MarshalProfile(profile)
}

// ProfileImport creates a profile from YAML data.
func (d *Dots) ProfileImport(ctx context.Context, data []byte) error {
	profile, err := dots.ParseProfile(data)
	if err != nil {
		return err
	}
	return d.saveProfile(profile)
}

func (d *Dots) profilePath(name string) string {
	return filepath.Join(d.PathService.ProfilesDir(), name+".yaml")
}

func (d *Dots) loadProfile(name string) (*dots.Profile, error) {
	data, err := d.Runtime.ReadFile(d.profilePath(name))
	if err != nil {
		if isNotExist(err) {
			return nil, dots.ErrNotExist
		}
		return nil, err
	}
	return dots.ParseProfile(data)
}

func (d *Dots) saveProfile(profile *dots.Profile) error {
	data, err := dots.MarshalProfile(profile)
	if err != nil {
		return err
	}
	path := d.profilePath(profile.Name)
	if err := d.Runtime.Mkdir(filepath.Dir(path), 0o755, true); err != nil {
		return err
	}
	return d.Runtime.WriteFile(path, data, 0o644)
}
