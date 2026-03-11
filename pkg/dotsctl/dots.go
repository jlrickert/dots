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

	PathService   *PathService
	ConfigService *ConfigService
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
	}
	configService := NewConfigService(pathService, configPath, rt)

	repo := opts.Repo
	if repo == nil {
		repo = dots.NewFsRepo(pathService.ConfigDir(), pathService.StateDir(), nil)
	}

	return &Dots{
		Runtime:       rt,
		Repo:          repo,
		PathService:   pathService,
		ConfigService: configService,
	}, nil
}
