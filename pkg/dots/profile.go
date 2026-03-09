package dots

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// Profile represents a named collection of packages.
type Profile struct {
	Name     string   `yaml:"name"`
	Extends  string   `yaml:"extends,omitempty"`
	Packages []string `yaml:"packages"`
}

// ParseProfile parses a Profile from YAML bytes.
func ParseProfile(data []byte) (*Profile, error) {
	var p Profile
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("%w: profile: %v", ErrParse, err)
	}
	if p.Name == "" {
		return nil, fmt.Errorf("%w: profile name is required", ErrParse)
	}
	return &p, nil
}

// MarshalProfile serializes a Profile to YAML bytes.
func MarshalProfile(p *Profile) ([]byte, error) {
	return yaml.Marshal(p)
}
