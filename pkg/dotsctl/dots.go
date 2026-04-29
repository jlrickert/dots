package dotsctl

import (
	"fmt"

	"github.com/jlrickert/cli-toolkit/toolkit"
	"github.com/jlrickert/dots/pkg/dots"
)

// Dots is the root service struct for the dots package manager. It composes
// all sub-services and exposes high-level operations as methods. Each
// operation lives in its own dots_<op>.go file.
type Dots struct {
	Runtime *toolkit.Runtime
	Repo    dots.Repository

	PathService      *PathService
	ConfigService    *ConfigService
	WorkStateService *WorkStateService
}

// DotsOptions configures how Dots is constructed.
type DotsOptions struct {
	Runtime    *toolkit.Runtime
	ConfigPath string
	Repo       dots.Repository
}

// NewDots constructs a Dots service from the given options.
func NewDots(opts DotsOptions) (*Dots, error) {
	rt := opts.Runtime
	if rt == nil {
		return nil, fmt.Errorf("runtime is required")
	}

	platform := dots.DetectPlatform()
	pathService := NewPathService(platform, rt)

	configPath := opts.ConfigPath
	if configPath == "" {
		configPath = pathService.UserConfigFile()
	} else {
		expanded, err := toolkit.ExpandPath(rt, configPath)
		if err != nil {
			return nil, fmt.Errorf("expand config path: %w", err)
		}
		configPath = expanded
	}
	configService := NewConfigService(pathService, configPath, rt)
	workStatePath := pathService.WorkStateFile()
	workStateService := NewWorkStateService(pathService, workStatePath, rt)

	repo := opts.Repo
	if repo == nil {
		repo = dots.NewFsRepo(pathService.ConfigDir(), pathService.StateDir(), nil)
	}

	return &Dots{
		Runtime:          rt,
		Repo:             repo,
		PathService:      pathService,
		ConfigService:    configService,
		WorkStateService: workStateService,
	}, nil
}

// migrateWorkModeIfNeeded performs a one-shot, idempotent move of work_mode
// entries from config.yaml into the work state file, then clears them from
// config. Called automatically before WorkOn/WorkOff writes; safe to call
// repeatedly. State wins on conflict — newer writes already use the new path,
// so config.yaml entries only fill gaps for taps not yet in state.
//
// The config is read with cache=true so writers that have already populated
// the cache (e.g. tests, or earlier ops in the same process) see their own
// changes. Disk reads only happen on cache miss.
//
// Partial-failure recovery: if the state save succeeds but the config save
// fails, the next invocation finds the same legacy entries and re-attempts
// the migration. The state-wins gate at line below skips entries already
// migrated, so re-runs do not clobber the user's intended state.
func (d *Dots) migrateWorkModeIfNeeded() error {
	cfg, err := d.ConfigService.Config(true)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if len(cfg.WorkMode) == 0 {
		return nil
	}

	state, err := d.WorkStateService.Load(false)
	if err != nil {
		return fmt.Errorf("load work state: %w", err)
	}
	if state.Taps == nil {
		state.Taps = make(map[string]string)
	}
	for tap, path := range cfg.WorkMode {
		if _, exists := state.Taps[tap]; exists {
			// State already has an entry — newer wins; do not overwrite.
			continue
		}
		state.Taps[tap] = path
	}
	if err := d.WorkStateService.Save(state); err != nil {
		return fmt.Errorf("save work state: %w", err)
	}

	cfg.WorkMode = map[string]string{}
	if err := d.ConfigService.Save(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	return nil
}
