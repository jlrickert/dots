package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newWhichCmd(deps *Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "which <file>",
		Short: "Identify which package placed a file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := newDotsService(deps)
			if err != nil {
				return err
			}

			pkg, err := d.Which(cmd.Context(), args[0])
			if err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), pkg)
			return nil
		},
	}
}
