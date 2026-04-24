package dots

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

const (
	// DotsConfigSchemaURL is the URL for the dots config JSON Schema.
	DotsConfigSchemaURL      = "https://raw.githubusercontent.com/jlrickert/dots/main/schemas/dots-config.json"
	DotsConfigSchemaModeline = "# yaml-language-server: $schema=" + DotsConfigSchemaURL + "\n"
)

// Config represents the user's dots configuration (~/.config/dots/config.yaml).
type Config struct {
	Core     CoreConfig            `yaml:"core,omitempty"`
	Git      GitConfig             `yaml:"git,omitempty"`
	Taps     map[string]TapConfig  `yaml:"taps,omitempty"`
	WorkMode map[string]string     `yaml:"work_mode,omitempty"`
	Aliases  map[string]string     `yaml:"aliases,omitempty"`
	Platform map[string]CoreConfig `yaml:"platform,omitempty"`
}

// CoreConfig holds top-level behavior settings.
type CoreConfig struct {
	ActiveProfile    string       `yaml:"active_profile,omitempty"`
	ConflictStrategy string       `yaml:"conflict_strategy,omitempty"`
	Backup           *bool        `yaml:"backup,omitempty"`
	LinkStrategy     LinkStrategy `yaml:"link_strategy,omitempty"`
}

// GitConfig holds git-related settings.
type GitConfig struct {
	DefaultBranch string `yaml:"default_branch,omitempty"`
	Protocol      string `yaml:"protocol,omitempty"`
}

// TapConfig describes a tap entry in the config file.
type TapConfig struct {
	URL        string `yaml:"url"`
	Branch     string `yaml:"branch,omitempty"`
	Provider   string `yaml:"provider,omitempty"`
	Visibility string `yaml:"visibility,omitempty"`
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	backupTrue := true
	return Config{
		Core: CoreConfig{
			ConflictStrategy: "prompt",
			Backup:           &backupTrue,
			LinkStrategy:     LinkCopy,
		},
		Git: GitConfig{
			DefaultBranch: "main",
			Protocol:      "ssh",
		},
		Taps:     make(map[string]TapConfig),
		WorkMode: make(map[string]string),
		Aliases:  make(map[string]string),
		Platform: make(map[string]CoreConfig),
	}
}

// ParseConfig parses a Config from YAML bytes.
func ParseConfig(data []byte) (*Config, error) {
	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrParse, err)
	}
	return &cfg, nil
}

// LoadConfigFile reads and parses a config file from disk.
func LoadConfigFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseConfig(data)
}

// MergeConfig merges override into base. Non-zero override fields win.
func MergeConfig(base, override *Config) *Config {
	result := *base

	if override.Core.ActiveProfile != "" {
		result.Core.ActiveProfile = override.Core.ActiveProfile
	}
	if override.Core.ConflictStrategy != "" {
		result.Core.ConflictStrategy = override.Core.ConflictStrategy
	}
	if override.Core.Backup != nil {
		result.Core.Backup = override.Core.Backup
	}
	if override.Core.LinkStrategy != "" {
		result.Core.LinkStrategy = override.Core.LinkStrategy
	}
	if override.Git.DefaultBranch != "" {
		result.Git.DefaultBranch = override.Git.DefaultBranch
	}
	if override.Git.Protocol != "" {
		result.Git.Protocol = override.Git.Protocol
	}

	if len(override.Taps) > 0 {
		if result.Taps == nil {
			result.Taps = make(map[string]TapConfig)
		}
		for k, v := range override.Taps {
			result.Taps[k] = v
		}
	}
	if len(override.WorkMode) > 0 {
		if result.WorkMode == nil {
			result.WorkMode = make(map[string]string)
		}
		for k, v := range override.WorkMode {
			result.WorkMode[k] = v
		}
	}
	if len(override.Aliases) > 0 {
		if result.Aliases == nil {
			result.Aliases = make(map[string]string)
		}
		for k, v := range override.Aliases {
			result.Aliases[k] = v
		}
	}
	if len(override.Platform) > 0 {
		if result.Platform == nil {
			result.Platform = make(map[string]CoreConfig)
		}
		for k, v := range override.Platform {
			result.Platform[k] = v
		}
	}

	return &result
}

// ResolveCorePlatform returns the effective CoreConfig after applying
// platform overrides from the config.
func (c *Config) ResolveCorePlatform(p Platform) CoreConfig {
	core := c.Core

	if osOverride, ok := c.Platform[p.OS]; ok {
		if osOverride.LinkStrategy != "" {
			core.LinkStrategy = osOverride.LinkStrategy
		}
		if osOverride.ConflictStrategy != "" {
			core.ConflictStrategy = osOverride.ConflictStrategy
		}
		if osOverride.Backup != nil {
			core.Backup = osOverride.Backup
		}
	}

	if archOverride, ok := c.Platform[p.String()]; ok {
		if archOverride.LinkStrategy != "" {
			core.LinkStrategy = archOverride.LinkStrategy
		}
		if archOverride.ConflictStrategy != "" {
			core.ConflictStrategy = archOverride.ConflictStrategy
		}
		if archOverride.Backup != nil {
			core.Backup = archOverride.Backup
		}
	}

	return core
}
