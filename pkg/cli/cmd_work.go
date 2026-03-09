package cli

import (
	"fmt"

	"github.com/jlrickert/dots/pkg/dotsctl"
	"github.com/spf13/cobra"
)

func newWorkCmd(deps *Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "work",
		Short: "Manage work mode for taps",
	}

	cmd.AddCommand(
		newWorkOnCmd(deps),
		newWorkOffCmd(deps),
		newWorkStatusCmd(deps),
		newWorkRebuildCmd(deps),
	)

	return cmd
}

func newWorkOnCmd(deps *Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "on <tap> <local-path>",
		Short: "Rewire links to a local checkout",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := newDotsService(deps)
			if err != nil {
				return err
			}
			if err := d.WorkOn(cmd.Context(), dotsctl.WorkOnOptions{
				Tap:       args[0],
				LocalPath: args[1],
			}); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Work mode enabled for tap %s -> %s\n", args[0], args[1])
			return nil
		},
	}
	cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return completeTapNames(deps)(cmd, args, toComplete)
		}
		// Second arg is a local path — let the shell do directory completion.
		return nil, cobra.ShellCompDirectiveFilterDirs
	}
	return cmd
}

func newWorkOffCmd(deps *Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "off <tap>",
		Short: "Rewire links back to internal clone",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := newDotsService(deps)
			if err != nil {
				return err
			}
			if err := d.WorkOff(cmd.Context(), args[0]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Work mode disabled for tap %s\n", args[0])
			return nil
		},
	}
	cmd.ValidArgsFunction = completeTapNames(deps)
	return cmd
}

func newWorkStatusCmd(deps *Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show work mode status",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := newDotsService(deps)
			if err != nil {
				return err
			}
			statuses, err := d.WorkStatusList(cmd.Context())
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if len(statuses) == 0 {
				fmt.Fprintln(out, "No taps in work mode")
				return nil
			}
			for _, s := range statuses {
				fmt.Fprintf(out, "%s -> %s\n", s.Tap, s.LocalPath)
			}
			return nil
		},
	}
}

func newWorkRebuildCmd(deps *Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rebuild [<tap>/<package>]",
		Short: "Re-link after local changes",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := newDotsService(deps)
			if err != nil {
				return err
			}
			pkgRef := ""
			if len(args) > 0 {
				pkgRef = args[0]
			}
			if err := d.Rebuild(cmd.Context(), pkgRef); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Rebuild complete")
			return nil
		},
	}
	cmd.ValidArgsFunction = completeInstalledPackages(deps)
	return cmd
}
