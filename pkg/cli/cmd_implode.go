package cli

import (
	"fmt"

	"github.com/jlrickert/dots/pkg/dotsctl"
	"github.com/spf13/cobra"
)

func newImplodeCmd(deps *Deps) *cobra.Command {
	var opts dotsctl.ImplodeOptions

	cmd := &cobra.Command{
		Use:   "implode",
		Short: "Uninstall all packages and remove dots' config and state",
		Long: "Uninstall every installed package, then remove the dots config and " +
			"state directories. The dots binary itself is not removed.",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := newDotsService(deps)
			if err != nil {
				return err
			}

			result, err := d.Implode(cmd.Context(), opts)
			out := cmd.OutOrStdout()
			if result != nil {
				for _, ref := range result.Uninstalled {
					fmt.Fprintf(out, "Uninstalled %s\n", ref)
				}
				for _, ref := range result.Failed {
					fmt.Fprintf(out, "Failed to uninstall %s\n", ref)
				}
				if result.StateDirRemoved {
					fmt.Fprintln(out, "Removed state directory")
				}
				if result.ConfigDirRemoved {
					fmt.Fprintln(out, "Removed config directory")
				}
			}
			return err
		},
	}

	cmd.Flags().BoolVarP(&opts.Yes, "yes", "y", false, "Confirm the destructive operation")

	return cmd
}
