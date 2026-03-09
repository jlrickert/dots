package cli

import (
	"fmt"

	"github.com/jlrickert/dots/pkg/dotsctl"
	"github.com/spf13/cobra"
)

func newListCmd(deps *Deps) *cobra.Command {
	var tap string
	var available bool

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List packages",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := newDotsService(deps)
			if err != nil {
				return err
			}

			result, err := d.List(cmd.Context(), dotsctl.ListOptions{
				Tap:       tap,
				Available: available,
			})
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			if available {
				for _, p := range result.Available {
					fmt.Fprintf(out, "%s/%s\n", p.Tap, p.Name)
				}
			} else {
				if len(result.Installed) == 0 {
					fmt.Fprintln(out, "No packages installed")
					return nil
				}
				for _, p := range result.Installed {
					fmt.Fprintf(out, "%s\t%s\t%s\n", p.Package, p.Version, p.LinkStrategy)
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&tap, "tap", "", "Filter by tap")
	cmd.Flags().BoolVar(&available, "available", false, "List available (not installed) packages")

	return cmd
}
