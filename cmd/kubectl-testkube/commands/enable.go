package commands

import (
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/oauth"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/telemetry"
	"github.com/kubeshop/testkube/pkg/ui"
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

	cmd.AddCommand(telemetry.NewEnableTelemetryCmd())
	cmd.AddCommand(oauth.NewEnableOAuthCmd())

	return cmd
}
