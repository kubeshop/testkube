package commands

import (
	"fmt"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/internal/migrations"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewMigrateCmd() *cobra.Command {
	var namespace string
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "migrate command",
		Long:  `migrate command manages migrations`,
		Run: func(cmd *cobra.Command, args []string) {
			ui.Logo()

			client, _ := common.GetClient(cmd)
			info, err := client.GetServerInfo()
			ui.ExitOnError("getting server info", err)

			if info.Version == "" {
				ui.Failf("Can't detect cluster version")
			}

			migrator := migrations.Migrator
			ui.Info("Available migrations for", info.Version)
			migrations := migrator.GetValidMigrations(info.Version)
			if len(migrations) == 0 {
				ui.Warn("No migrations available for", info.Version)
			}

			for _, migration := range migrations {
				fmt.Printf("- %+v - %s\n", migration.Version(), migration.Info())
			}

			err = migrator.Run(info.Version)
			ui.ExitOnError("running migrations", err)
			ui.Success("All migrations executed successfully")
		},
	}

	cmd.Flags().StringVar(&namespace, "namespace", "testkube", "testkube namespace")

	return cmd
}
