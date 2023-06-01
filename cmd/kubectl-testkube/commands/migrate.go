package commands

import (
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewMigrateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "manual migrate command",
		Long:  `migrate command will run migrations greater or equals current version`,
		Run: func(cmd *cobra.Command, args []string) {
			hasMigrations, err := common.RunMigrations(cmd)
			ui.ExitOnError("Running migrations", err)
			if hasMigrations {
				ui.Success("All migrations executed successfully")
			}
		},
	}

	return cmd
}
