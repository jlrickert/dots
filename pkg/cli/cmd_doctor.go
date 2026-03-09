package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newDoctorCmd(deps *Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Run diagnostics",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := newDotsService(deps)
			if err != nil {
				return err
			}

			checks, err := d.Doctor(cmd.Context())
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			for _, c := range checks {
				icon := "OK"
				switch c.Status {
				case "warn":
					icon = "WARN"
				case "error":
					icon = "ERROR"
				}
				fmt.Fprintf(out, "[%s] %s: %s\n", icon, c.Name, c.Detail)
			}
			return nil
		},
	}
}
