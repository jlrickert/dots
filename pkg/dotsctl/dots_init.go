package dotsctl

import (
	"context"
	"fmt"

	"github.com/jlrickert/dots/pkg/dots"
	"gopkg.in/yaml.v3"
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
// When --from and --path are provided, it clones the tap, finds the dots
// package at the given path, and installs it (self-bootstrapping).
func (d *Dots) Init(ctx context.Context, opts InitOptions) error {
	// Create directories.
	configDir := d.PathService.ConfigDir()
	if err := d.Runtime.Mkdir(configDir, 0o755, true); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	stateDir := d.PathService.StateDir()
	if err := d.Runtime.Mkdir(stateDir, 0o755, true); err != nil {
		return fmt.Errorf("create state dir: %w", err)
	}

	profilesDir := d.PathService.ProfilesDir()
	if err := d.Runtime.Mkdir(profilesDir, 0o755, true); err != nil {
		return fmt.Errorf("create profiles dir: %w", err)
	}

	tapsDir := d.PathService.TapsDir()
	if err := d.Runtime.Mkdir(tapsDir, 0o755, true); err != nil {
		return fmt.Errorf("create taps dir: %w", err)
	}

	// Write default config if none exists.
	configPath := d.ConfigService.ConfigPath
	if _, err := d.Runtime.Stat(configPath, true); err != nil {
		cfg := dots.DefaultConfig()
		if opts.From != "" {
			cfg.Taps["default"] = dots.TapConfig{
				URL: opts.From,
			}
		}

		data, err := yaml.Marshal(&cfg)
		if err != nil {
			return fmt.Errorf("marshal config: %w", err)
		}
		out := append([]byte(dots.DotsConfigSchemaModeline), data...)
		if err := d.Runtime.WriteFile(configPath, out, 0o644); err != nil {
			return fmt.Errorf("write config: %w", err)
		}

		d.ConfigService.InvalidateCache()
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

		// If --path is provided, install the dots package from the tap.
		// This is the self-bootstrapping flow: the dots package manages
		// dots' own config as a regular package.
		if opts.Path != "" {
			_, err := d.Install(ctx, InstallOptions{
				Package: tapName + "/" + opts.Path,
			})
			if err != nil {
				return fmt.Errorf("install bootstrap package: %w", err)
			}
		}
	}

	return nil
}
