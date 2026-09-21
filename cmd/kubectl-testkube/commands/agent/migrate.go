package agent

import (
	"github.com/spf13/cobra"
)

func NewMigrateAgentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "runner",
		Aliases: []string{"agent"},
		Short:   "manual migrate runner command",
		Long:    `migrate runner command will run runner migrations greater or equals current version`,
		Run: func(cmd *cobra.Command, args []string) {
			// TODO: Delete, as we don't have any migrations
		},
	}

	return cmd
}
