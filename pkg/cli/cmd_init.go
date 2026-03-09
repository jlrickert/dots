package cli

import (
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
			return d.Init(cmd.Context(), opts)
		},
	}

	cmd.Flags().StringVar(&opts.From, "from", "", "Git URL to clone as initial tap")
	cmd.Flags().StringVar(&opts.Path, "path", "", "Package path within the tap")

	return cmd
}
