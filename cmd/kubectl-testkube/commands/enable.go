package commands

import (
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/analytics"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/oauth"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewEnableCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "enable <feature>",
		Aliases: []string{"on"},
		Short:   "Enable feature",
		Run: func(cmd *cobra.Command, args []string) {
			err := cmd.Help()
			ui.PrintOnError("Displaying help", err)
		},
	}

	cmd.AddCommand(analytics.NewEnableAnalyticsCmd())
	cmd.AddCommand(oauth.NewEnableOAuthCmd())

	return cmd
}
