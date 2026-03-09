package cli

import (
	"fmt"
	"os"

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
		newProfileSwitchCmd(deps),
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
	return &cobra.Command{
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
	return &cobra.Command{
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
			if profile.Extends != "" {
				fmt.Fprintf(out, "Extends:  %s\n", profile.Extends)
			}
			fmt.Fprintln(out, "Packages:")
			for _, pkg := range profile.Packages {
				fmt.Fprintf(out, "  %s\n", pkg)
			}
			return nil
		},
	}
}

func newProfileAddCmd(deps *Deps) *cobra.Command {
	return &cobra.Command{
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
}

func newProfileRemoveCmd(deps *Deps) *cobra.Command {
	return &cobra.Command{
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
}

func newProfileApplyCmd(deps *Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "apply <name>",
		Short: "Install all packages in a profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := newDotsService(deps)
			if err != nil {
				return err
			}
			if err := d.ProfileApply(cmd.Context(), args[0]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Applied profile %s\n", args[0])
			return nil
		},
	}
}

func newProfileSwitchCmd(deps *Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "switch <name>",
		Short: "Switch to a different profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := newDotsService(deps)
			if err != nil {
				return err
			}
			if err := d.ProfileSwitch(cmd.Context(), args[0]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Switched to profile %s\n", args[0])
			return nil
		},
	}
}

func newProfileExportCmd(deps *Deps) *cobra.Command {
	return &cobra.Command{
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
