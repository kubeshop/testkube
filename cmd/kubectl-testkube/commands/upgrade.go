package commands

import (
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewUpgradeCmd() *cobra.Command {
	var (
		noDashboard            bool
		noMinio                bool
		noJetstack             bool
		chart, name, namespace string
	)

	cmd := &cobra.Command{
		Use:     "upgrade",
		Short:   "Upgrade Helm chart, install dependencies and run migrations",
		Aliases: []string{"update"},
		Run: func(cmd *cobra.Command, args []string) {
			ui.Logo()

			hasMigrations, err := RunMigrations(cmd)
			ui.ExitOnError("Running migrations", err)
			if hasMigrations {
				ui.Success("All migrations executed successfully")
			}

			err = HelmUpgradeOrInstalTestkube(name, namespace, chart, noDashboard, noMinio, noJetstack)
			ui.ExitOnError("installing Testkube", err)

		},
	}

	cmd.Flags().StringVar(&chart, "chart", "kubeshop/testkube", "chart name")
	cmd.Flags().StringVar(&name, "name", "testkube", "installation name")
	cmd.Flags().StringVar(&namespace, "namespace", "testkube", "namespace where to install")

	cmd.Flags().BoolVar(&noMinio, "no-minio", false, "don't install MinIO")
	cmd.Flags().BoolVar(&noDashboard, "no-dashboard", false, "don't install dashboard")
	cmd.Flags().BoolVar(&noJetstack, "no-jetstack", false, "don't install Jetstack")

	return cmd
}
