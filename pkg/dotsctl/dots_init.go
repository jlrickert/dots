package dotsctl

import (
	"context"
	"fmt"
	"os"

	"github.com/jlrickert/dots/pkg/dots"
)

// InitOptions configures the init operation.
type InitOptions struct {
	// From is an optional Git URL to clone as the first tap.
	From string
	// Path is the package directory within the tap containing the dots config.
	Path string
}

// Init initializes the dots environment: creates config and state directories,
// writes a default config, and optionally registers an initial tap.
func (d *Dots) Init(ctx context.Context, opts InitOptions) error {
	// Create directories.
	configDir := d.PathService.ConfigDir()
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	stateDir := d.PathService.StateDir()
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		return fmt.Errorf("create state dir: %w", err)
	}

	profilesDir := d.PathService.ProfilesDir()
	if err := os.MkdirAll(profilesDir, 0o755); err != nil {
		return fmt.Errorf("create profiles dir: %w", err)
	}

	// Write default config if none exists.
	configPath := d.ConfigService.ConfigPath
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		cfg := dots.DefaultConfig()
		if opts.From != "" {
			cfg.Taps["default"] = dots.TapConfig{
				URL: opts.From,
			}
		}
		// Config writing will be handled by FsRepo in later phases.
		// For now, just ensure directories exist.
	}

	// Register the initial tap if --from is provided.
	if opts.From != "" {
		tapName := "default"
		err := d.Repo.AddTap(ctx, dots.TapInfo{
			Name: tapName,
			URL:  opts.From,
		})
		if err != nil && err != dots.ErrExist {
			return fmt.Errorf("register tap: %w", err)
		}
	}

	return nil
}
