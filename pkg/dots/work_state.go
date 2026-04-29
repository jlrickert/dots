package dots

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

const (
	// DotsWorkStateSchemaURL is the URL for the dots work-state JSON Schema.
	DotsWorkStateSchemaURL      = "https://raw.githubusercontent.com/jlrickert/dots/main/schemas/work-state.json"
	DotsWorkStateSchemaModeline = "# yaml-language-server: $schema=" + DotsWorkStateSchemaURL + "\n"
)

// WorkState represents the host-local work-mode state stored separately from
// the user's version-controlled config. It lives at @state/dots/work.yaml and
// must never be checked into a dotfiles repo.
type WorkState struct {
	// Taps maps a tap name to the absolute local checkout path that should be
	// used in place of the internal clone for installs and re-links.
	Taps map[string]string `yaml:"taps,omitempty"`
}

// DefaultWorkState returns a zero-valued WorkState with a non-nil Taps map.
func DefaultWorkState() WorkState {
	return WorkState{Taps: make(map[string]string)}
}

// ParseWorkState parses a WorkState from YAML bytes. A YAML decode failure is
// wrapped as ErrParse so callers can use errors.Is.
func ParseWorkState(data []byte) (*WorkState, error) {
	state := DefaultWorkState()
	if err := yaml.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrParse, err)
	}
	if state.Taps == nil {
		state.Taps = make(map[string]string)
	}
	return &state, nil
}

// LoadWorkStateFile reads and parses a work state file from disk. The
// underlying os.ReadFile error is returned unwrapped on read failures so
// callers can detect absence with errors.Is(err, os.ErrNotExist) without
// confusing it with a parse failure.
func LoadWorkStateFile(path string) (*WorkState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseWorkState(data)
}
