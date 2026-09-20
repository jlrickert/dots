package cli

import (
	"fmt"

	"github.com/jlrickert/dots/pkg/dotsctl"
	"github.com/spf13/cobra"
)

func newInitCmd(deps *Deps) *cobra.Command {
	var opts dotsctl.InitOptions

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize the dots environment",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := newDotsService(deps)
			if err != nil {
				return err
			}
			if err := d.Init(cmd.Context(), opts); err != nil {
				return err
			}
			if opts.Apply != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "Applied profile %s\n", opts.Apply)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&opts.From, "from", "", "Git URL to clone as initial tap")
	cmd.Flags().StringVar(&opts.Path, "path", "", "Package path within the tap")
	cmd.Flags().StringVar(&opts.Name, "name", "", "Tap name for --from clone (default: \"default\"). Set this to match the tap declared in the bootstrap package's config to avoid a duplicate tap entry.")
	cmd.Flags().StringVar(&opts.Apply, "apply", "", "Profile to apply after initialization (same as 'dots profile apply')")

	return cmd
}
