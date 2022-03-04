package commands

import (
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/analytics"
	"github.com/spf13/cobra"
)

func NewDisableCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "disable <feature>",
		Aliases: []string{"off"},
		Short:   "Disable feature",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	cmd.AddCommand(analytics.NewDisableAnalyticsCmd())

	return cmd
}
