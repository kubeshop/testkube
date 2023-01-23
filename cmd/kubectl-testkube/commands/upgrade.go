package commands

import (
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/pkg/ui"
)

func NewUpgradeCmd() *cobra.Command {
	var options HelmUpgradeOrInstalTestkubeOptions

	cmd := &cobra.Command{
		Use:     "upgrade",
		Short:   "Upgrade Helm chart, install dependencies and run migrations",
		Aliases: []string{"update"},
		Run: func(cmd *cobra.Command, args []string) {

			hasMigrations, err := RunMigrations(cmd)
			ui.ExitOnError("Running migrations", err)
			if hasMigrations {
				ui.Success("All migrations executed successfully")
			}

			err = HelmUpgradeOrInstalTestkube(options)
			ui.ExitOnError("upgrading Testkube", err)

		},
	}

	PopulateUpgradeInstallFlags(cmd, &options)

	return cmd
}
