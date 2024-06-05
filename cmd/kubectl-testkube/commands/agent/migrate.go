package agent

import (
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewMigrateAgentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "manual migrate agent command",
		Long:  `migrate agent command will run agent migrations greater or equals current version`,
		Run: func(cmd *cobra.Command, args []string) {
			hasMigrations, err := common.RunAgentMigrations(cmd)
			ui.ExitOnError("Running agent migrations", err)
			if hasMigrations {
				ui.Success("All agent migrations executed successfully")
			}
		},
	}

	return cmd
}
