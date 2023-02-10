package commands

import (
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewUpgradeCmd() *cobra.Command {
	var options HelmUpgradeOrInstalTestkubeOptions

	cmd := &cobra.Command{
		Use:     "upgrade",
		Short:   "Upgrade Helm chart, install dependencies and run migrations",
		Aliases: []string{"update"},
		Run: func(cmd *cobra.Command, args []string) {

			cfg, err := config.Load()
			ui.ExitOnError("loading config file", err)
			ui.NL()

			if !options.NoConfirm {
				ui.Warn("This will upgrade Testkube to the latest version. This may take a few minutes.")
				ui.Warn("Please be sure you're on valid kubectl context before continuing!")
				ui.NL()

				currentContext, err := GetCurrentKubernetesContext()
				ui.ExitOnError("getting current context", err)
				ui.Alert("Current kubectl context:", currentContext)
				ui.NL()

				ok := ui.Confirm("Do you want to continue?")
				if !ok {
					ui.Errf("Upgrade cancelled")
					return
				}
			}

			if cfg.ContextType == config.ContextTypeCloud {
				ui.Info("Testkube Cloud agent upgrade started")
				err = HelmUpgradeOrInstalTestkubeCloud(options, cfg)
				ui.ExitOnError("Upgrading Testkube Cloud Agent", err)
			} else {
				ui.Info("Updating testkube")
				hasMigrations, err := RunMigrations(cmd)
				ui.ExitOnError("Running migrations", err)
				if hasMigrations {
					ui.Success("All migrations executed successfully")
				}

				err = HelmUpgradeOrInstalTestkube(options)
				ui.ExitOnError("Upgrading Testkube", err)
			}

		},
	}

	PopulateUpgradeInstallFlags(cmd, &options)

	return cmd
}
