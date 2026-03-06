package dotsctl

import (
	"github.com/jlrickert/dots/pkg/dots"
)

// ConfigService loads and merges dots configuration with caching.
type ConfigService struct {
	PathService *PathService
	ConfigPath  string

	cached *dots.Config
}

// NewConfigService creates a ConfigService.
func NewConfigService(ps *PathService, configPath string) *ConfigService {
	return &ConfigService{
		PathService: ps,
		ConfigPath:  configPath,
	}
}

// Config returns the effective configuration. If cache is true, a previously
// loaded config is returned without re-reading from disk.
func (s *ConfigService) Config(cache bool) (*dots.Config, error) {
	if cache && s.cached != nil {
		return s.cached, nil
	}

	cfg, err := dots.LoadConfigFile(s.ConfigPath)
	if err != nil {
		// If the config file doesn't exist, return defaults.
		def := dots.DefaultConfig()
		s.cached = &def
		return s.cached, nil
	}

	s.cached = cfg
	return cfg, nil
}

// InvalidateCache clears the cached config.
func (s *ConfigService) InvalidateCache() {
	s.cached = nil
}
