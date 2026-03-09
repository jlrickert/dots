package dotsctl

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jlrickert/dots/pkg/dots"
)

// ProfileCreate creates a new empty profile.
func (d *Dots) ProfileCreate(ctx context.Context, name string) error {
	path := d.profilePath(name)
	if _, err := os.Stat(path); err == nil {
		return dots.ErrExist
	}

	profile := &dots.Profile{
		Name:     name,
		Packages: []string{},
	}

	data, err := dots.MarshalProfile(profile)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// ProfileDelete removes a profile definition.
func (d *Dots) ProfileDelete(ctx context.Context, name string) error {
	path := d.profilePath(name)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return dots.ErrNotExist
	}
	return os.Remove(path)
}

// ProfileList returns all available profile names.
func (d *Dots) ProfileList(ctx context.Context) ([]string, error) {
	dir := d.PathService.ProfilesDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
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

// ProfileShow returns the profile definition.
func (d *Dots) ProfileShow(ctx context.Context, name string) (*dots.Profile, error) {
	return d.loadProfile(name)
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

// ProfileApply installs all packages in a profile.
func (d *Dots) ProfileApply(ctx context.Context, name string) error {
	profile, err := d.resolveProfileChain(name)
	if err != nil {
		return err
	}

	for _, pkg := range profile.Packages {
		_, err := d.Install(ctx, InstallOptions{Package: pkg})
		if err != nil {
			return fmt.Errorf("install %s: %w", pkg, err)
		}
	}

	// Update active profile in lockfile
	lockfile, err := d.Repo.ReadLockfile(ctx)
	if err != nil && !isNotExist(err) {
		return err
	}
	if lockfile == nil {
		lockfile = &dots.Lockfile{}
	}
	lockfile.State.ActiveProfile = name
	return d.Repo.WriteLockfile(ctx, lockfile)
}

// ProfileSwitch removes current profile packages and applies the new one.
func (d *Dots) ProfileSwitch(ctx context.Context, name string) error {
	// Remove currently installed packages
	lockfile, err := d.Repo.ReadLockfile(ctx)
	if err == nil {
		for _, pkg := range lockfile.Installed {
			_ = d.Remove(ctx, RemoveOptions{Package: pkg.Package})
		}
	}

	return d.ProfileApply(ctx, name)
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
	data, err := os.ReadFile(d.profilePath(name))
	if err != nil {
		if os.IsNotExist(err) {
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
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// resolveProfileChain resolves a profile and its extends chain.
func (d *Dots) resolveProfileChain(name string) (*dots.Profile, error) {
	profile, err := d.loadProfile(name)
	if err != nil {
		return nil, err
	}

	if profile.Extends == "" {
		return profile, nil
	}

	parent, err := d.resolveProfileChain(profile.Extends)
	if err != nil {
		return nil, fmt.Errorf("resolve parent profile %s: %w", profile.Extends, err)
	}

	// Merge: parent packages first, then child's (dedup)
	seen := make(map[string]struct{})
	var merged []string
	for _, p := range parent.Packages {
		if _, ok := seen[p]; !ok {
			merged = append(merged, p)
			seen[p] = struct{}{}
		}
	}
	for _, p := range profile.Packages {
		if _, ok := seen[p]; !ok {
			merged = append(merged, p)
			seen[p] = struct{}{}
		}
	}

	profile.Packages = merged
	return profile, nil
}
