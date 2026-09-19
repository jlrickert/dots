package dotsctl

import (
	"context"
	"path/filepath"

	"github.com/jlrickert/dots/pkg/dots"
)

// ProfileCreate creates a new empty profile.
func (d *Dots) ProfileCreate(ctx context.Context, name string) error {
	path := d.profilePath(name)
	if _, err := d.Runtime.Stat(path, false); err == nil {
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

	if err := d.Runtime.Mkdir(filepath.Dir(path), 0o755, true); err != nil {
		return err
	}
	return d.Runtime.WriteFile(path, data, 0o644)
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
