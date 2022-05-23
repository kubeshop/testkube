package commands

import (
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/analytics"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/oauth"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewDisableCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "disable <feature>",
		Aliases: []string{"off"},
		Short:   "Disable feature",
		Run: func(cmd *cobra.Command, args []string) {
			err := cmd.Help()
			ui.PrintOnError("Displaying help", err)
		},
	}

	cmd.AddCommand(analytics.NewDisableAnalyticsCmd())
	cmd.AddCommand(oauth.NewDisableOAuthCmd())

	return cmd
}
