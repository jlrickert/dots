package cli

import (
	"fmt"

	"github.com/jlrickert/dots/pkg/dots"
	"github.com/jlrickert/dots/pkg/dotsctl"
	"github.com/spf13/cobra"
)

func newInstallCmd(deps *Deps) *cobra.Command {
	var dryRun bool
	var strategy string

	cmd := &cobra.Command{
		Use:   "install <tap>/<package>",
		Short: "Install a dotfile package",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := newDotsService(deps)
			if err != nil {
				return err
			}

			opts := dotsctl.InstallOptions{
				Package:  args[0],
				DryRun:   dryRun,
				Strategy: dots.LinkStrategy(strategy),
			}

			result, err := d.Install(cmd.Context(), opts)
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			if result.DryRun {
				fmt.Fprintln(out, "Dry run — no changes made:")
			}
			for _, f := range result.Files {
				fmt.Fprintf(out, "  %s -> %s (%s)\n", f.Src, f.Dest, f.Method)
			}
			if !result.DryRun {
				fmt.Fprintf(out, "Installed %s (%d files)\n", result.Package, len(result.Files))
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print what would happen without writing")
	cmd.Flags().StringVar(&strategy, "strategy", "", "Override link strategy (symlink, copy, hardlink)")

	return cmd
}
