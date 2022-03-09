package commands

import (
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/analytics"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewEnableCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "enable <feature>",
		Aliases: []string{"on"},
		Short:   "Enable feature",
		Run: func(cmd *cobra.Command, args []string) {
			ui.Logo()
			cmd.Help()
		},
	}

	cmd.AddCommand(analytics.NewEnableAnalyticsCmd())

	return cmd
}
