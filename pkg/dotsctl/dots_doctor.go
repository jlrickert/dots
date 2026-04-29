package dotsctl

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/jlrickert/dots/pkg/dots"
)

// DoctorCheck is a single diagnostic result.
type DoctorCheck struct {
	Name   string
	Status string // "ok", "warn", "error"
	Detail string
}

// Doctor runs diagnostics and returns a list of checks.
func (d *Dots) Doctor(ctx context.Context) ([]DoctorCheck, error) {
	var checks []DoctorCheck

	// Check config directory exists.
	configDir := d.PathService.ConfigDir()
	checks = append(checks, checkDir("config directory", configDir))

	// Check state directory exists.
	stateDir := d.PathService.StateDir()
	checks = append(checks, checkDir("state directory", stateDir))

	// Check config file.
	configPath := d.ConfigService.ConfigPath
	if _, err := os.Stat(configPath); err != nil {
		checks = append(checks, DoctorCheck{
			Name:   "config file",
			Status: "warn",
			Detail: fmt.Sprintf("%s not found (using defaults)", configPath),
		})
	} else {
		_, err := d.ConfigService.Config(false)
		if err != nil {
			checks = append(checks, DoctorCheck{
				Name:   "config file",
				Status: "error",
				Detail: fmt.Sprintf("parse error: %v", err),
			})
		} else {
			checks = append(checks, DoctorCheck{
				Name:   "config file",
				Status: "ok",
				Detail: configPath,
			})
		}
	}

	// Check taps.
	taps, err := d.Repo.ListTaps(ctx)
	if err != nil {
		checks = append(checks, DoctorCheck{
			Name:   "taps",
			Status: "error",
			Detail: fmt.Sprintf("failed to list taps: %v", err),
		})
	} else {
		checks = append(checks, DoctorCheck{
			Name:   "taps",
			Status: "ok",
			Detail: fmt.Sprintf("%d tap(s) registered", len(taps)),
		})
	}

	// Work-mode related checks. These all read the work state file and the
	// legacy work_mode map directly so we surface migration issues without
	// triggering an implicit migration write.
	checks = append(checks, d.checkWorkStateFile())
	checks = append(checks, d.checkWorkModeLegacy())
	checks = append(checks, d.checkWorkStateConflict())
	checks = append(checks, d.checkMergeConflictMarkers())
	checks = append(checks, d.checkWorkStateOrphan())
	checks = append(checks, d.checkWorkStatePath())

	return checks, nil
}

// checkWorkStateFile verifies the work state file is absent or parses cleanly.
func (d *Dots) checkWorkStateFile() DoctorCheck {
	path := d.PathService.WorkStateFile()
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return DoctorCheck{
				Name:   "work state file",
				Status: "ok",
				Detail: "no work state (no taps in work mode)",
			}
		}
		return DoctorCheck{
			Name:   "work state file",
			Status: "error",
			Detail: fmt.Sprintf("stat %s: %v", path, err),
		}
	}
	if _, err := dots.LoadWorkStateFile(path); err != nil {
		return DoctorCheck{
			Name:   "work state file",
			Status: "error",
			Detail: fmt.Sprintf("parse error: %v", err),
		}
	}
	return DoctorCheck{
		Name:   "work state file",
		Status: "ok",
		Detail: path,
	}
}

// checkWorkModeLegacy flags any leftover work_mode entries in config.yaml that
// have not yet been migrated to the work state file.
func (d *Dots) checkWorkModeLegacy() DoctorCheck {
	cfg, err := d.ConfigService.Config(true)
	if err != nil {
		return DoctorCheck{
			Name:   "work mode (legacy)",
			Status: "error",
			Detail: fmt.Sprintf("load config: %v", err),
		}
	}
	if cfg == nil || len(cfg.WorkMode) == 0 {
		return DoctorCheck{
			Name:   "work mode (legacy)",
			Status: "ok",
			Detail: "no legacy work_mode entries in config.yaml",
		}
	}
	taps := sortedKeys(cfg.WorkMode)
	return DoctorCheck{
		Name:   "work mode (legacy)",
		Status: "warn",
		Detail: fmt.Sprintf(
			"taps still in config.yaml work_mode: %s; run `dots work on/off` to migrate",
			strings.Join(taps, ", "),
		),
	}
}

// checkWorkStateConflict flags taps present in both legacy work_mode and the
// new work state file with conflicting paths.
func (d *Dots) checkWorkStateConflict() DoctorCheck {
	cfg, cfgErr := d.ConfigService.Config(true)
	state, stateErr := d.WorkStateService.Load(true)
	if cfgErr != nil {
		return DoctorCheck{
			Name:   "work state conflict",
			Status: "error",
			Detail: fmt.Sprintf("load config: %v", cfgErr),
		}
	}
	if stateErr != nil {
		return DoctorCheck{
			Name:   "work state conflict",
			Status: "error",
			Detail: fmt.Sprintf("load work state: %v", stateErr),
		}
	}

	if cfg == nil || state == nil || len(cfg.WorkMode) == 0 || len(state.Taps) == 0 {
		return DoctorCheck{
			Name:   "work state conflict",
			Status: "ok",
			Detail: "no conflicts between config and state",
		}
	}

	var conflicts []string
	for tap, configPath := range cfg.WorkMode {
		statePath, ok := state.Taps[tap]
		if !ok {
			continue
		}
		if configPath != statePath {
			conflicts = append(conflicts,
				fmt.Sprintf("%s (config=%s, state=%s)", tap, configPath, statePath))
		}
	}
	if len(conflicts) == 0 {
		return DoctorCheck{
			Name:   "work state conflict",
			Status: "ok",
			Detail: "no conflicts between config and state",
		}
	}
	sort.Strings(conflicts)
	return DoctorCheck{
		Name:   "work state conflict",
		Status: "error",
		Detail: "conflicting work-mode entries: " + strings.Join(conflicts, "; "),
	}
}

// checkMergeConflictMarkers scans config.yaml and work.yaml for unresolved
// git merge conflict markers. Both files are user-editable and can pick up
// conflict markers if the user merges branches by hand.
func (d *Dots) checkMergeConflictMarkers() DoctorCheck {
	candidates := []string{
		d.ConfigService.ConfigPath,
		d.PathService.WorkStateFile(),
	}
	markers := [][]byte{
		[]byte("<<<<<<< "),
		[]byte("======="),
		[]byte(">>>>>>> "),
	}
	var hits []string
	for _, path := range candidates {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		for _, m := range markers {
			if bytes.Contains(data, m) {
				hits = append(hits, path)
				break
			}
		}
	}
	if len(hits) == 0 {
		return DoctorCheck{
			Name:   "merge conflict markers",
			Status: "ok",
			Detail: "no unresolved merge conflict markers detected",
		}
	}
	return DoctorCheck{
		Name:   "merge conflict markers",
		Status: "error",
		Detail: "unresolved merge conflict markers in: " + strings.Join(hits, ", "),
	}
}

// checkWorkStateOrphan flags taps in the work state file that no longer have
// a registered tap entry in config.yaml.
func (d *Dots) checkWorkStateOrphan() DoctorCheck {
	cfg, cfgErr := d.ConfigService.Config(true)
	state, stateErr := d.WorkStateService.Load(true)
	if cfgErr != nil {
		return DoctorCheck{
			Name:   "work state orphan",
			Status: "error",
			Detail: fmt.Sprintf("load config: %v", cfgErr),
		}
	}
	if stateErr != nil {
		return DoctorCheck{
			Name:   "work state orphan",
			Status: "error",
			Detail: fmt.Sprintf("load work state: %v", stateErr),
		}
	}
	if state == nil || len(state.Taps) == 0 {
		return DoctorCheck{
			Name:   "work state orphan",
			Status: "ok",
			Detail: "no orphan tap entries in work state",
		}
	}
	registered := map[string]struct{}{}
	if cfg != nil {
		for name := range cfg.Taps {
			registered[name] = struct{}{}
		}
	}
	var orphans []string
	for tap := range state.Taps {
		if _, ok := registered[tap]; !ok {
			orphans = append(orphans, tap)
		}
	}
	if len(orphans) == 0 {
		return DoctorCheck{
			Name:   "work state orphan",
			Status: "ok",
			Detail: "no orphan tap entries in work state",
		}
	}
	sort.Strings(orphans)
	return DoctorCheck{
		Name:   "work state orphan",
		Status: "warn",
		Detail: "work state references unregistered taps: " + strings.Join(orphans, ", "),
	}
}

// checkWorkStatePath flags taps in the work state file whose recorded local
// path does not exist on disk.
func (d *Dots) checkWorkStatePath() DoctorCheck {
	state, err := d.WorkStateService.Load(true)
	if err != nil {
		return DoctorCheck{
			Name:   "work state path",
			Status: "error",
			Detail: fmt.Sprintf("load work state: %v", err),
		}
	}
	if state == nil || len(state.Taps) == 0 {
		return DoctorCheck{
			Name:   "work state path",
			Status: "ok",
			Detail: "no work state paths to check",
		}
	}
	var missing []string
	taps := sortedKeys(state.Taps)
	for _, tap := range taps {
		path := state.Taps[tap]
		if _, err := os.Lstat(path); err != nil {
			missing = append(missing, fmt.Sprintf("%s -> %s", tap, path))
		}
	}
	if len(missing) == 0 {
		return DoctorCheck{
			Name:   "work state path",
			Status: "ok",
			Detail: "all work state paths exist",
		}
	}
	return DoctorCheck{
		Name:   "work state path",
		Status: "warn",
		Detail: "missing work-mode checkout paths: " + strings.Join(missing, ", "),
	}
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func checkDir(name, path string) DoctorCheck {
	info, err := os.Stat(path)
	if err != nil {
		return DoctorCheck{
			Name:   name,
			Status: "warn",
			Detail: fmt.Sprintf("%s does not exist", path),
		}
	}
	if !info.IsDir() {
		return DoctorCheck{
			Name:   name,
			Status: "error",
			Detail: fmt.Sprintf("%s exists but is not a directory", path),
		}
	}
	return DoctorCheck{
		Name:   name,
		Status: "ok",
		Detail: path,
	}
}
