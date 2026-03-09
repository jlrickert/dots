package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newDiffCmd(deps *Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "diff <tap>/<package>",
		Short: "Show differences between source and installed files",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := newDotsService(deps)
			if err != nil {
				return err
			}

			diffs, err := d.Diff(cmd.Context(), args[0])
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			if len(diffs) == 0 {
				fmt.Fprintln(out, "No differences")
				return nil
			}

			for _, diff := range diffs {
				fmt.Fprintf(out, "%s\t%s\n", diff.Status, diff.File)
			}
			return nil
		},
	}
}
