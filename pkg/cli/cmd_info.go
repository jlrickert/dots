package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newInfoCmd(deps *Deps) *cobra.Command {
	var showPlatform bool

	cmd := &cobra.Command{
		Use:   "info [<tap>/<package>]",
		Short: "Show package or platform info",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := newDotsService(deps)
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()

			if showPlatform {
				status, err := d.Status(cmd.Context())
				if err != nil {
					return err
				}
				fmt.Fprintf(out, "Platform: %s\n", status.Platform)
				return nil
			}

			if len(args) == 0 {
				return fmt.Errorf("specify a package or use --platform")
			}

			result, err := d.Info(cmd.Context(), args[0])
			if err != nil {
				return err
			}

			fmt.Fprintf(out, "Package:  %s\n", result.Package)
			fmt.Fprintf(out, "Tap:      %s\n", result.Tap)
			fmt.Fprintf(out, "Version:  %s\n", result.Version)

			if result.Manifest != nil {
				if len(result.Manifest.Links) > 0 {
					fmt.Fprintln(out, "Links:")
					for src, dest := range result.Manifest.Links {
						fmt.Fprintf(out, "  %s -> %s\n", src, dest)
					}
				}
			}

			if result.Installed != nil {
				fmt.Fprintf(out, "Status:   installed (%s)\n", result.Installed.LinkStrategy)
				fmt.Fprintf(out, "Files:    %d\n", len(result.Installed.Files))
			} else {
				fmt.Fprintln(out, "Status:   not installed")
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&showPlatform, "platform", false, "Show platform info")

	return cmd
}
