package cli

import "github.com/jlrickert/dots/pkg/dotsctl"

// newDotsService creates a Dots service from CLI deps.
func newDotsService(deps *Deps) (*dotsctl.Dots, error) {
	return dotsctl.NewDots(dotsctl.DotsOptions{
		Runtime:    deps.Runtime,
		ConfigPath: deps.ConfigPath,
	})
}
