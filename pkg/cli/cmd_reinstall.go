package cli

import (
	"fmt"

	"github.com/jlrickert/dots/pkg/dotsctl"
	"github.com/spf13/cobra"
)

func newReinstallCmd(deps *Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reinstall <tap>/<package>",
		Short: "Reinstall a package (remove then install)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := newDotsService(deps)
			if err != nil {
				return err
			}

			result, err := d.Reinstall(cmd.Context(), dotsctl.ReinstallOptions{Package: args[0]})
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Reinstalled %s (%d files)\n", result.Package, len(result.Files))
			return nil
		},
	}

	return cmd
}
