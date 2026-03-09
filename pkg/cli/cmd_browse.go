package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newBrowseCmd(deps *Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "browse <tap>",
		Short: "List packages in a tap",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := newDotsService(deps)
			if err != nil {
				return err
			}

			result, err := d.Browse(cmd.Context(), args[0])
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			if len(result.Packages) == 0 {
				fmt.Fprintf(out, "No packages in tap %s\n", args[0])
				return nil
			}

			fmt.Fprintf(out, "Packages in %s (%d):\n\n", result.Tap, len(result.Packages))
			for _, pkg := range result.Packages {
				status := "  "
				if pkg.Installed {
					status = "* "
				}
				line := fmt.Sprintf("%s%s/%s", status, result.Tap, pkg.Name)
				if pkg.Version != "" {
					line += fmt.Sprintf(" (%s)", pkg.Version)
				}
				fmt.Fprintln(out, line)

				if pkg.Description != "" {
					fmt.Fprintf(out, "    %s\n", pkg.Description)
				}
				if len(pkg.Tags) > 0 {
					fmt.Fprintf(out, "    tags: %s\n", strings.Join(pkg.Tags, ", "))
				}
				if pkg.LinkCount > 0 {
					fmt.Fprintf(out, "    %d file(s)\n", pkg.LinkCount)
				}
			}
			return nil
		},
	}

	cmd.ValidArgsFunction = completeTapNames(deps)

	return cmd
}
