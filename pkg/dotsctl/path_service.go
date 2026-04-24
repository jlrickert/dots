package dotsctl

import (
	"path/filepath"

	"github.com/jlrickert/cli-toolkit/toolkit"
	"github.com/jlrickert/dots/pkg/dots"
)

// PathService resolves platform-native paths for dots directories and files.
type PathService struct {
	Platform platform
	Resolver *dots.AliasResolver
	runtime  *toolkit.Runtime
}

type platform = dots.Platform

// NewPathService creates a PathService for the given platform and runtime.
func NewPathService(p dots.Platform, rt *toolkit.Runtime) *PathService {
	return &PathService{
		Platform: p,
		Resolver: dots.NewAliasResolver(p, rt),
		runtime:  rt,
	}
}

// ConfigDir returns the dots config directory (e.g. ~/.config/dots).
func (s *PathService) ConfigDir() string {
	base, _ := s.Resolver.ResolveAlias(dots.AliasConfig)
	return filepath.Join(base, "dots")
}

// StateDir returns the dots state directory (e.g. ~/.local/state/dots).
func (s *PathService) StateDir() string {
	base, _ := s.Resolver.ResolveAlias(dots.AliasState)
	return filepath.Join(base, "dots")
}

// UserConfigFile returns the path to the user's config.yaml.
func (s *PathService) UserConfigFile() string {
	return filepath.Join(s.ConfigDir(), "config.yaml")
}

// ProfilesDir returns the directory containing profile definitions.
func (s *PathService) ProfilesDir() string {
	return filepath.Join(s.ConfigDir(), "profiles")
}

// LockfilePath returns the path to dots.lock.yaml.
func (s *PathService) LockfilePath() string {
	return filepath.Join(s.StateDir(), "dots.lock.yaml")
}

// TapsDir returns the directory where tap clones are stored.
func (s *PathService) TapsDir() string {
	return filepath.Join(s.StateDir(), "taps")
}

// MergedDir returns the directory for merged overlay output.
func (s *PathService) MergedDir() string {
	return filepath.Join(s.StateDir(), "merged")
}

// BackupsDir returns the directory for file backups.
func (s *PathService) BackupsDir() string {
	return filepath.Join(s.StateDir(), "backups")
}
