package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/jlrickert/dots/pkg/dotsctl"

	"github.com/spf13/cobra"
)

func newProfileCmd(deps *Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profile",
		Short: "Manage profiles",
	}

	cmd.AddCommand(
		newProfileCreateCmd(deps),
		newProfileDeleteCmd(deps),
		newProfileListCmd(deps),
		newProfileShowCmd(deps),
		newProfileAddCmd(deps),
		newProfileRemoveCmd(deps),
		newProfileApplyCmd(deps),
		newProfileExportCmd(deps),
		newProfileImportCmd(deps),
	)

	return cmd
}

func newProfileCreateCmd(deps *Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := newDotsService(deps)
			if err != nil {
				return err
			}
			if err := d.ProfileCreate(cmd.Context(), args[0]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Created profile %s\n", args[0])
			return nil
		},
	}
}

func newProfileDeleteCmd(deps *Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := newDotsService(deps)
			if err != nil {
				return err
			}
			if err := d.ProfileDelete(cmd.Context(), args[0]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Deleted profile %s\n", args[0])
			return nil
		},
	}
	cmd.ValidArgsFunction = completeProfileNames(deps)
	return cmd
}

func newProfileListCmd(deps *Deps) *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List profiles",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := newDotsService(deps)
			if err != nil {
				return err
			}
			names, err := d.ProfileList(cmd.Context())
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			for _, name := range names {
				fmt.Fprintln(out, name)
			}
			return nil
		},
	}
}

func newProfileShowCmd(deps *Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <name>",
		Short: "Show profile details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := newDotsService(deps)
			if err != nil {
				return err
			}
			profile, err := d.ProfileShow(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Name:     %s\n", profile.Name)
			if len(profile.Extends) > 0 {
				fmt.Fprintf(out, "Extends:  %s\n", strings.Join(profile.Extends, ", "))
			}
			if len(profile.Exclude) > 0 {
				fmt.Fprintf(out, "Exclude:  %s\n", strings.Join(profile.Exclude, ", "))
			}
			fmt.Fprintln(out, "Packages:")
			for _, pkg := range profile.Packages {
				fmt.Fprintf(out, "  %s\n", pkg)
			}
			return nil
		},
	}
	cmd.ValidArgsFunction = completeProfileNames(deps)
	return cmd
}

func newProfileAddCmd(deps *Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <name> <tap>/<package>...",
		Short: "Add packages to a profile",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := newDotsService(deps)
			if err != nil {
				return err
			}
			if err := d.ProfileAdd(cmd.Context(), args[0], args[1:]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Added %d package(s) to profile %s\n", len(args)-1, args[0])
			return nil
		},
	}
	cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return completeProfileNames(deps)(cmd, args, toComplete)
		}
		return completeAvailablePackages(deps)(cmd, nil, toComplete)
	}
	return cmd
}

func newProfileRemoveCmd(deps *Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove <name> <tap>/<package>",
		Short: "Remove a package from a profile",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := newDotsService(deps)
			if err != nil {
				return err
			}
			if err := d.ProfileRemove(cmd.Context(), args[0], args[1]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Removed %s from profile %s\n", args[1], args[0])
			return nil
		},
	}
	cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return completeProfileNames(deps)(cmd, args, toComplete)
		}
		return completeAvailablePackages(deps)(cmd, nil, toComplete)
	}
	return cmd
}

func newProfileApplyCmd(deps *Deps) *cobra.Command {
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "apply <name>",
		Short: "Make installed packages match a profile",
		Long: `Make installed packages match a profile exactly.

Uninstalls every installed package that is not in the profile's resolved list
(plus core.always_installed), including packages installed by hand, installs
the missing ones, and records the profile as active.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := newDotsService(deps)
			if err != nil {
				return err
			}
			result, err := d.ProfileApply(cmd.Context(), dotsctl.ProfileApplyOptions{
				Name:   args[0],
				DryRun: dryRun,
			})
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			if result.DryRun {
				fmt.Fprintln(out, "Dry run — no changes made:")
			}
			for _, ref := range result.Remove {
				fmt.Fprintf(out, "  remove  %s\n", ref)
			}
			for _, ref := range result.Install {
				fmt.Fprintf(out, "  install %s\n", ref)
			}
			if !result.DryRun {
				fmt.Fprintf(out, "Applied profile %s (%d removed, %d installed, %d unchanged)\n",
					result.Profile, len(result.Remove), len(result.Install), len(result.Keep))
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print what would be removed and installed without changing anything")
	cmd.ValidArgsFunction = completeProfileNames(deps)
	return cmd
}

func newProfileExportCmd(deps *Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export <name>",
		Short: "Export profile as YAML",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := newDotsService(deps)
			if err != nil {
				return err
			}
			data, err := d.ProfileExport(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			fmt.Fprint(cmd.OutOrStdout(), string(data))
			return nil
		},
	}
	cmd.ValidArgsFunction = completeProfileNames(deps)
	return cmd
}

func newProfileImportCmd(deps *Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "import <file>",
		Short: "Import a profile from YAML file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := newDotsService(deps)
			if err != nil {
				return err
			}
			data, err := os.ReadFile(args[0])
			if err != nil {
				return err
			}
			if err := d.ProfileImport(cmd.Context(), data); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Profile imported")
			return nil
		},
	}
}
