package dotsctl

import (
	"context"
	"fmt"

	"github.com/jlrickert/dots/pkg/dots"
)

// TapAddOptions configures the tap add operation.
type TapAddOptions struct {
	Name   string
	URL    string
	Branch string
}

// TapAdd registers a new tap.
func (d *Dots) TapAdd(ctx context.Context, opts TapAddOptions) error {
	if opts.Name == "" {
		return fmt.Errorf("tap name is required")
	}
	if opts.URL == "" {
		return fmt.Errorf("tap URL is required")
	}

	tap := dots.TapInfo{
		Name:   opts.Name,
		URL:    opts.URL,
		Branch: opts.Branch,
	}

	return d.Repo.AddTap(ctx, tap)
}

// TapRemove removes a registered tap.
func (d *Dots) TapRemove(ctx context.Context, name string) error {
	if name == "" {
		return fmt.Errorf("tap name is required")
	}
	return d.Repo.RemoveTap(ctx, name)
}

// TapList returns all registered taps.
func (d *Dots) TapList(ctx context.Context) ([]dots.TapInfo, error) {
	return d.Repo.ListTaps(ctx)
}

// TapUpdate fetches the latest state for a tap.
func (d *Dots) TapUpdate(ctx context.Context, name string) error {
	if name == "" {
		// Update all taps
		taps, err := d.Repo.ListTaps(ctx)
		if err != nil {
			return err
		}
		for _, tap := range taps {
			if err := d.Repo.UpdateTap(ctx, tap.Name); err != nil {
				return fmt.Errorf("update tap %s: %w", tap.Name, err)
			}
		}
		return nil
	}
	return d.Repo.UpdateTap(ctx, name)
}
