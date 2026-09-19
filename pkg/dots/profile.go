package dots

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

const (
	ProfileSchemaURL      = "https://raw.githubusercontent.com/jlrickert/dots/main/schemas/profile.json"
	ProfileSchemaModeline = "# yaml-language-server: $schema=" + ProfileSchemaURL + "\n"
)

// Profile represents a named collection of packages.
type Profile struct {
	Name string `yaml:"name"`
	// Extends lists parent profiles, resolved in order. In YAML it may be a
	// single name or a list.
	Extends  StringList `yaml:"extends,omitempty"`
	Packages []string   `yaml:"packages"`
	// Exclude drops packages inherited through Extends. It never affects
	// this profile's own Packages.
	Exclude []string `yaml:"exclude,omitempty"`
}

// StringList is a list of strings that unmarshals from either a YAML scalar
// or a sequence, and marshals a single element back to a scalar so
// single-value files stay unchanged.
type StringList []string

// UnmarshalYAML implements yaml.Unmarshaler.
func (l *StringList) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		if node.Tag == "!!null" || node.Value == "" {
			*l = nil
			return nil
		}
		*l = StringList{node.Value}
		return nil
	case yaml.SequenceNode:
		var items []string
		if err := node.Decode(&items); err != nil {
			return err
		}
		*l = items
		return nil
	default:
		return fmt.Errorf("line %d: expected a string or a list of strings", node.Line)
	}
}

// MarshalYAML implements yaml.Marshaler.
func (l StringList) MarshalYAML() (any, error) {
	if len(l) == 1 {
		return l[0], nil
	}
	return []string(l), nil
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

// MarshalProfile serializes a Profile to YAML bytes with the
// yaml-language-server schema modeline prepended.
func MarshalProfile(p *Profile) ([]byte, error) {
	data, err := yaml.Marshal(p)
	if err != nil {
		return nil, err
	}
	return append([]byte(ProfileSchemaModeline), data...), nil
}
