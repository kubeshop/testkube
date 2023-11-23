package commands

import (
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/cloud"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewCloudCmd() *cobra.Command {

	cmd := &cobra.Command{
		Use:     "cloud",
		Short:   "[Deprecated] Testkube Cloud commands",
		Aliases: []string{"cl"},
		Run: func(cmd *cobra.Command, args []string) {
			ui.Warn("You are using a deprecated command, please switch to `testkube pro`.")
		},
	}

	cmd.AddCommand(cloud.NewConnectCmd())
	cmd.AddCommand(cloud.NewDisconnectCmd())
	cmd.AddCommand(cloud.NewInitCmd())
	cmd.AddCommand(cloud.NewLoginCmd())

	return cmd
}
