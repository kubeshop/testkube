package agent

import (
	"github.com/spf13/cobra"
)

func NewMigrateAgentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "manual migrate agent command",
		Long:  `migrate agent command will run agent migrations greater or equals current version`,
		Run: func(cmd *cobra.Command, args []string) {
			// TODO: Delete, as we don't have any migrations
		},
	}

	return cmd
}
