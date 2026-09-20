package dotsctl

import (
	"fmt"
	"slices"

	"github.com/jlrickert/dots/pkg/dots"
)

// AlwaysInstalledSource is the ResolvedPackage.Source for packages that come
// from core.always_installed rather than a profile.
const AlwaysInstalledSource = "always_installed"

// ResolvedPackage is one entry in a profile's resolved package list.
type ResolvedPackage struct {
	// Ref is the "<tap>/<package>" identifier.
	Ref string
	// Source is the profile that listed the package, or
	// AlwaysInstalledSource.
	Source string
}

// resolveProfile returns the resolved package list for a profile, following
// docs/profiles.md#resolution: parents in extends order (each already
// resolved with its own excludes), concatenated with first-occurrence dedupe,
// then this profile's excludes, then its own packages. stack holds the
// profiles currently being resolved and is used for cycle detection.
func (d *Dots) resolveProfile(name string, stack []string) ([]ResolvedPackage, error) {
	if i := slices.Index(stack, name); i >= 0 {
		chain := append(slices.Clone(stack[i:]), name)
		return nil, &dots.ProfileCycleError{Chain: chain}
	}

	profile, err := d.loadProfile(name)
	if err != nil {
		if len(stack) > 0 {
			return nil, fmt.Errorf("profile %q (extended by %q): %w", name, stack[len(stack)-1], err)
		}
		return nil, fmt.Errorf("profile %q: %w", name, err)
	}

	resolved, _, err := d.resolveLoadedProfile(profile, stack)
	return resolved, err
}

// resolveLoadedProfile resolves an already-loaded profile, which may not be
// saved yet. It also returns the excludes that matched no inherited package.
func (d *Dots) resolveLoadedProfile(profile *dots.Profile, stack []string) ([]ResolvedPackage, []string, error) {
	stack = append(slices.Clone(stack), profile.Name)

	var inherited []ResolvedPackage
	seen := make(map[string]struct{})
	for _, parent := range profile.Extends {
		pkgs, err := d.resolveProfile(parent, stack)
		if err != nil {
			return nil, nil, err
		}
		for _, p := range pkgs {
			if _, ok := seen[p.Ref]; ok {
				continue
			}
			seen[p.Ref] = struct{}{}
			inherited = append(inherited, p)
		}
	}

	var unmatched []string
	for _, ref := range profile.Exclude {
		if _, ok := seen[ref]; !ok {
			unmatched = append(unmatched, ref)
		}
	}

	resolved := make([]ResolvedPackage, 0, len(inherited)+len(profile.Packages))
	seen = make(map[string]struct{})
	for _, p := range inherited {
		if slices.Contains(profile.Exclude, p.Ref) {
			continue
		}
		seen[p.Ref] = struct{}{}
		resolved = append(resolved, p)
	}
	for _, ref := range profile.Packages {
		if _, ok := seen[ref]; ok {
			continue
		}
		seen[ref] = struct{}{}
		resolved = append(resolved, ResolvedPackage{Ref: ref, Source: profile.Name})
	}
	return resolved, unmatched, nil
}

// resolveProfileForApply resolves a profile and prepends the platform-resolved
// core.always_installed packages, which no exclude can remove.
func (d *Dots) resolveProfileForApply(name string) ([]ResolvedPackage, error) {
	pkgs, err := d.resolveProfile(name, nil)
	if err != nil {
		return nil, err
	}
	return d.withAlwaysInstalled(pkgs)
}

// withAlwaysInstalled prepends core.always_installed to pkgs, deduped.
func (d *Dots) withAlwaysInstalled(pkgs []ResolvedPackage) ([]ResolvedPackage, error) {
	cfg, err := d.ConfigService.Config(true)
	if err != nil {
		return nil, err
	}
	always := cfg.ResolveCorePlatform(d.PathService.Platform).AlwaysInstalled

	resolved := make([]ResolvedPackage, 0, len(always)+len(pkgs))
	seen := make(map[string]struct{})
	for _, ref := range always {
		seen[ref] = struct{}{}
		resolved = append(resolved, ResolvedPackage{Ref: ref, Source: AlwaysInstalledSource})
	}
	for _, p := range pkgs {
		if _, ok := seen[p.Ref]; ok {
			continue
		}
		seen[p.Ref] = struct{}{}
		resolved = append(resolved, p)
	}
	return resolved, nil
}
