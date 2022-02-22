package commands

import (
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/analytics"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewAnalyticsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "analytics",
		Short: "Analytics management actions",
		Run: func(cmd *cobra.Command, args []string) {
			ui.Logo()
			cmd.Usage()
		},
	}

	cmd.AddCommand(analytics.NewEnableAnalyticsCmd())
	cmd.AddCommand(analytics.NewDisableAnalyticsCmd())
	cmd.AddCommand(analytics.NewStatusAnalyticsCmd())

	return cmd
}
