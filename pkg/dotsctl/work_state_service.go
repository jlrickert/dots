package dotsctl

import (
	"errors"
	"os"

	"github.com/jlrickert/cli-toolkit/toolkit"
	"github.com/jlrickert/dots/pkg/dots"
	"gopkg.in/yaml.v3"
)

// WorkStateService loads and persists the host-local work-mode state file.
// State lives at @state/dots/work.yaml and is intentionally separate from
// the user's version-controlled config.yaml.
type WorkStateService struct {
	PathService *PathService
	Path        string
	Runtime     *toolkit.Runtime

	cached *dots.WorkState
}

// NewWorkStateService creates a WorkStateService.
func NewWorkStateService(ps *PathService, path string, rt *toolkit.Runtime) *WorkStateService {
	return &WorkStateService{
		PathService: ps,
		Path:        path,
		Runtime:     rt,
	}
}

// Load returns the current work state, falling back to DefaultWorkState() when
// the file does not yet exist. If cache is true, a previously loaded state is
// returned without re-reading from disk.
//
// Reads go through the runtime so jailed test environments see the same file
// the matching Save call wrote — the package-level dots.LoadWorkStateFile
// helper is for callers reading by absolute host path.
func (s *WorkStateService) Load(cache bool) (*dots.WorkState, error) {
	if cache && s.cached != nil {
		return s.cached, nil
	}

	data, err := s.Runtime.ReadFile(s.Path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			def := dots.DefaultWorkState()
			s.cached = &def
			return s.cached, nil
		}
		return nil, err
	}

	state, err := dots.ParseWorkState(data)
	if err != nil {
		return nil, err
	}
	s.cached = state
	return state, nil
}

// Save marshals state to YAML, prepends the schema modeline, and writes it
// atomically. The atomic write creates parent directories as needed.
func (s *WorkStateService) Save(state *dots.WorkState) error {
	data, err := yaml.Marshal(state)
	if err != nil {
		return err
	}
	out := append([]byte(dots.DotsWorkStateSchemaModeline), data...)
	if err := s.Runtime.AtomicWriteFile(s.Path, out, 0o644); err != nil {
		return err
	}
	s.cached = state
	return nil
}

// Get returns the local path recorded for tap and whether it was present.
func (s *WorkStateService) Get(tap string) (string, bool) {
	state, err := s.Load(true)
	if err != nil || state == nil {
		return "", false
	}
	path, ok := state.Taps[tap]
	return path, ok
}

// Set records path for tap and persists the updated state.
func (s *WorkStateService) Set(tap, path string) error {
	state, err := s.Load(false)
	if err != nil {
		return err
	}
	if state.Taps == nil {
		state.Taps = make(map[string]string)
	}
	state.Taps[tap] = path
	return s.Save(state)
}

// Delete removes any entry for tap from the state and persists the result.
func (s *WorkStateService) Delete(tap string) error {
	state, err := s.Load(false)
	if err != nil {
		return err
	}
	if state.Taps != nil {
		delete(state.Taps, tap)
	}
	return s.Save(state)
}

// All returns a copy of the current tap -> local path map.
func (s *WorkStateService) All() (map[string]string, error) {
	state, err := s.Load(true)
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(state.Taps))
	for k, v := range state.Taps {
		out[k] = v
	}
	return out, nil
}

// InvalidateCache clears the cached work state.
func (s *WorkStateService) InvalidateCache() {
	s.cached = nil
}
