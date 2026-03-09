package cli

import (
	"fmt"

	"github.com/jlrickert/dots/pkg/dotsctl"
	"github.com/spf13/cobra"
)

func newTapCmd(deps *Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tap",
		Short: "Manage taps (package sources)",
	}

	cmd.AddCommand(
		newTapAddCmd(deps),
		newTapRemoveCmd(deps),
		newTapListCmd(deps),
		newTapUpdateCmd(deps),
	)

	return cmd
}

func newTapAddCmd(deps *Deps) *cobra.Command {
	var branch string

	cmd := &cobra.Command{
		Use:   "add <name> <url>",
		Short: "Register a new tap",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := newDotsService(deps)
			if err != nil {
				return err
			}

			err = d.TapAdd(cmd.Context(), dotsctl.TapAddOptions{
				Name:   args[0],
				URL:    args[1],
				Branch: branch,
			})
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Added tap %s\n", args[0])
			return nil
		},
	}

	cmd.Flags().StringVar(&branch, "branch", "main", "Git branch to track")

	return cmd
}

func newTapRemoveCmd(deps *Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "remove <name>",
		Aliases: []string{"rm"},
		Short:   "Remove a registered tap",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := newDotsService(deps)
			if err != nil {
				return err
			}

			if err := d.TapRemove(cmd.Context(), args[0]); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Removed tap %s\n", args[0])
			return nil
		},
	}

	cmd.ValidArgsFunction = completeTapNames(deps)

	return cmd
}

func newTapListCmd(deps *Deps) *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List registered taps",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := newDotsService(deps)
			if err != nil {
				return err
			}

			taps, err := d.TapList(cmd.Context())
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			for _, t := range taps {
				fmt.Fprintf(out, "%s\t%s\n", t.Name, t.URL)
			}
			return nil
		},
	}
}

func newTapUpdateCmd(deps *Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update [<name>]",
		Short: "Update tap(s) to latest",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := newDotsService(deps)
			if err != nil {
				return err
			}

			name := ""
			if len(args) > 0 {
				name = args[0]
			}

			if err := d.TapUpdate(cmd.Context(), name); err != nil {
				return err
			}

			if name == "" {
				fmt.Fprintln(cmd.OutOrStdout(), "All taps updated")
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "Updated tap %s\n", name)
			}
			return nil
		},
	}

	cmd.ValidArgsFunction = completeTapNames(deps)

	return cmd
}
