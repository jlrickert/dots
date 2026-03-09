package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// completeAvailablePackages returns tap/package completions from all registered taps.
func completeAvailablePackages(deps *Deps) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		d, err := newDotsService(deps)
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
		taps, err := d.TapList(cmd.Context())
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
		var completions []string
		for _, tap := range taps {
			pkgs, err := d.Repo.ListPackages(cmd.Context(), tap.Name)
			if err != nil {
				continue
			}
			for _, pkg := range pkgs {
				completions = append(completions, fmt.Sprintf("%s/%s", tap.Name, pkg.Name))
			}
		}
		return completions, cobra.ShellCompDirectiveNoFileComp
	}
}

// completeInstalledPackages returns tap/package completions from the lockfile.
func completeInstalledPackages(deps *Deps) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		d, err := newDotsService(deps)
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
		lockfile, err := d.Repo.ReadLockfile(cmd.Context())
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		var completions []string
		for _, pkg := range lockfile.Installed {
			completions = append(completions, pkg.Package)
		}
		return completions, cobra.ShellCompDirectiveNoFileComp
	}
}

// completeTapNames returns registered tap name completions.
func completeTapNames(deps *Deps) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		d, err := newDotsService(deps)
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
		taps, err := d.TapList(cmd.Context())
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
		var completions []string
		for _, tap := range taps {
			completions = append(completions, tap.Name)
		}
		return completions, cobra.ShellCompDirectiveNoFileComp
	}
}

// completeProfileNames returns profile name completions.
func completeProfileNames(deps *Deps) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		d, err := newDotsService(deps)
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
		names, err := d.ProfileList(cmd.Context())
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return names, cobra.ShellCompDirectiveNoFileComp
	}
}
