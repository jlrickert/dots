package cli

import (
	"fmt"

	"github.com/jlrickert/dots/pkg/dotsctl"
	"github.com/spf13/cobra"
)

func newRemoveCmd(deps *Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "remove <tap>/<package>",
		Aliases: []string{"rm", "uninstall"},
		Short:   "Remove an installed package",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := newDotsService(deps)
			if err != nil {
				return err
			}

			if err := d.Remove(cmd.Context(), dotsctl.RemoveOptions{Package: args[0]}); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Removed %s\n", args[0])
			return nil
		},
	}

	return cmd
}
