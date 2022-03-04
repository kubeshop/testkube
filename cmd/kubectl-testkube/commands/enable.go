package commands

import (
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/analytics"
	"github.com/spf13/cobra"
)

func NewEnableCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "enable <feature>",
		Aliases: []string{"on"},
		Short:   "Enable feature",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	cmd.AddCommand(analytics.NewEnableAnalyticsCmd())

	return cmd
}
