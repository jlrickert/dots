package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newSearchCmd(deps *Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "search <query>",
		Short: "Search packages across all taps",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := newDotsService(deps)
			if err != nil {
				return err
			}

			results, err := d.Search(cmd.Context(), args[0])
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			if len(results) == 0 {
				fmt.Fprintf(out, "No packages matching %q\n", args[0])
				return nil
			}

			for _, r := range results {
				status := " "
				if r.Installed {
					status = "*"
				}
				if r.Description != "" {
					fmt.Fprintf(out, "%s %s/%s - %s\n", status, r.Tap, r.Name, r.Description)
				} else {
					fmt.Fprintf(out, "%s %s/%s\n", status, r.Tap, r.Name)
				}
			}
			return nil
		},
	}
}
