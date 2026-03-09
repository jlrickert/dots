package cli

import (
	"fmt"

	"github.com/jlrickert/dots/pkg/dotsctl"
	"github.com/spf13/cobra"
)

func newSyncCmd(deps *Deps) *cobra.Command {
	var all bool

	cmd := &cobra.Command{
		Use:   "sync [<tap>/<package>]",
		Short: "Sync copy-strategy packages with source",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := newDotsService(deps)
			if err != nil {
				return err
			}

			opts := dotsctl.SyncOptions{All: all}
			if len(args) > 0 {
				opts.Package = args[0]
			}

			result, err := d.Sync(cmd.Context(), opts)
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			if len(result.Updated) > 0 {
				fmt.Fprintf(out, "Updated %d file(s)\n", len(result.Updated))
				for _, f := range result.Updated {
					fmt.Fprintf(out, "  %s\n", f)
				}
			} else {
				fmt.Fprintln(out, "Everything up to date")
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "Sync all copy-strategy packages")
	cmd.ValidArgsFunction = completeInstalledPackages(deps)

	return cmd
}
