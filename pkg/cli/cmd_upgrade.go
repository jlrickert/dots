package cli

import (
	"fmt"

	"github.com/jlrickert/dots/pkg/dotsctl"
	"github.com/spf13/cobra"
)

func newUpgradeCmd(deps *Deps) *cobra.Command {
	var all bool

	cmd := &cobra.Command{
		Use:   "upgrade [<tap>/<package>]",
		Short: "Upgrade installed packages",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := newDotsService(deps)
			if err != nil {
				return err
			}

			opts := dotsctl.UpgradeOptions{All: all}
			if len(args) > 0 {
				opts.Package = args[0]
			}

			if err := d.Upgrade(cmd.Context(), opts); err != nil {
				return err
			}

			if all {
				fmt.Fprintln(cmd.OutOrStdout(), "All packages upgraded")
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "Upgraded %s\n", args[0])
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "Upgrade all installed packages")
	cmd.ValidArgsFunction = completeInstalledPackages(deps)

	return cmd
}
