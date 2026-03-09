package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newStatusCmd(deps *Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show dots status overview",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := newDotsService(deps)
			if err != nil {
				return err
			}

			result, err := d.Status(cmd.Context())
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Platform:      %s\n", result.Platform)
			fmt.Fprintf(out, "Config:        %s\n", result.ConfigPath)
			fmt.Fprintf(out, "State:         %s\n", result.StateDir)
			fmt.Fprintf(out, "Link strategy: %s\n", result.LinkStrategy)
			fmt.Fprintf(out, "Profile:       %s\n", result.Profile)
			fmt.Fprintf(out, "Taps:          %d\n", result.TapCount)
			fmt.Fprintf(out, "Packages:      %d\n", result.PackageCount)

			return nil
		},
	}
}
