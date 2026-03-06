package dotsctl

import (
	"context"

	"github.com/jlrickert/dots/pkg/dots"
)

// StatusResult holds the result of a status operation.
type StatusResult struct {
	Platform     dots.Platform
	ConfigPath   string
	ConfigDir    string
	StateDir     string
	LinkStrategy dots.LinkStrategy
	Profile      string
	TapCount     int
	PackageCount int
}

// Status returns an overview of the current dots state.
func (d *Dots) Status(ctx context.Context) (*StatusResult, error) {
	cfg, err := d.ConfigService.Config(true)
	if err != nil {
		return nil, err
	}

	platform := d.PathService.Platform
	core := cfg.ResolveCorePlatform(platform)

	result := &StatusResult{
		Platform:     platform,
		ConfigPath:   d.ConfigService.ConfigPath,
		ConfigDir:    d.PathService.ConfigDir(),
		StateDir:     d.PathService.StateDir(),
		LinkStrategy: core.LinkStrategy,
		Profile:      core.ActiveProfile,
	}

	taps, err := d.Repo.ListTaps(ctx)
	if err == nil {
		result.TapCount = len(taps)
	}

	lockfile, err := d.Repo.ReadLockfile(ctx)
	if err == nil {
		result.PackageCount = len(lockfile.Installed)
		if lockfile.State.ActiveProfile != "" {
			result.Profile = lockfile.State.ActiveProfile
		}
	}

	return result, nil
}
