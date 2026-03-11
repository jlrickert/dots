package dotsctl

import (
	"github.com/jlrickert/cli-toolkit/toolkit"
	"github.com/jlrickert/dots/pkg/dots"
	"gopkg.in/yaml.v3"
)

// ConfigService loads and merges dots configuration with caching.
type ConfigService struct {
	PathService *PathService
	ConfigPath  string
	Runtime     *toolkit.Runtime

	cached *dots.Config
}

// NewConfigService creates a ConfigService.
func NewConfigService(ps *PathService, configPath string, rt *toolkit.Runtime) *ConfigService {
	return &ConfigService{
		PathService: ps,
		ConfigPath:  configPath,
		Runtime:     rt,
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

// Save writes the current config to disk and updates the cache.
func (s *ConfigService) Save(cfg *dots.Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	if err := s.Runtime.WriteFile(s.ConfigPath, data, 0o644); err != nil {
		return err
	}
	s.cached = cfg
	return nil
}

// InvalidateCache clears the cached config.
func (s *ConfigService) InvalidateCache() {
	s.cached = nil
}
