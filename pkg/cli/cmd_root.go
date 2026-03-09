package cli

import (
	"fmt"
	"os"

	"github.com/jlrickert/cli-toolkit/mylog"
	"github.com/jlrickert/cli-toolkit/toolkit"
	"github.com/spf13/cobra"
)

type Deps struct {
	Root     string
	Shutdown func()
	Runtime  *toolkit.Runtime

	ConfigPath string
	LogFile    string
	LogLevel   string
	LogJSON    bool
}

func NewRootCmd(deps *Deps) *cobra.Command {
	if deps == nil {
		deps = &Deps{}
	}
	if deps.Shutdown == nil {
		deps.Shutdown = func() {}
	}

	cmd := &cobra.Command{
		Use:           "dots",
		Short:         "A brew-style dotfile package manager",
		Version:       Version,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			rt := deps.Runtime
			if rt == nil {
				return fmt.Errorf("runtime is required")
			}

			wd, err := rt.Getwd()
			if err != nil {
				return err
			}
			deps.Root = wd

			if deps.LogFile != "" || deps.LogJSON || deps.LogLevel != "" {
				var out = os.Stderr
				var f *os.File
				if deps.LogFile != "" {
					var err error
					f, err = os.OpenFile(deps.LogFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
					if err != nil {
						return err
					}
					_ = f // closed on process exit
					out = f
				}
				lg := mylog.NewLogger(mylog.LoggerConfig{
					Out:     out,
					Level:   mylog.ParseLevel(deps.LogLevel),
					JSON:    deps.LogJSON,
					Version: Version,
				})
				if err := deps.Runtime.SetLogger(lg); err != nil {
					return err
				}
			}

			return nil
		},
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
			deps.Shutdown()
			return nil
		},
	}

	cmd.PersistentFlags().StringVar(&deps.LogFile, "log-file", "", "write logs to file (default stderr)")
	cmd.PersistentFlags().StringVar(&deps.LogLevel, "log-level", "info", "minimum log level")
	cmd.PersistentFlags().BoolVar(&deps.LogJSON, "log-json", false, "output logs as JSON")
	cmd.PersistentFlags().StringVarP(&deps.ConfigPath, "config", "c", "", "path to config file")

	// Wire subcommands
	cmd.AddCommand(
		newInitCmd(deps),
		newInstallCmd(deps),
		newRemoveCmd(deps),
		newUpgradeCmd(deps),
		newReinstallCmd(deps),
		newListCmd(deps),
		newStatusCmd(deps),
		newDoctorCmd(deps),
		newTapCmd(deps),
		newSyncCmd(deps),
		newInfoCmd(deps),
		newDiffCmd(deps),
		newWhichCmd(deps),
		newProfileCmd(deps),
		newWorkCmd(deps),
	)

	return cmd
}
