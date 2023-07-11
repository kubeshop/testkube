package commands

import (
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/cloud"
)

func NewCloudCmd() *cobra.Command {

	cmd := &cobra.Command{
		Use:     "cloud",
		Short:   "Testkube Cloud commands",
		Aliases: []string{"cl"},
		Run: func(cmd *cobra.Command, args []string) {
		},
	}

	cmd.AddCommand(cloud.NewConnectCmd())
	cmd.AddCommand(cloud.NewDisconnectCmd())
	cmd.AddCommand(cloud.NewInitCmd())
	cmd.AddCommand(cloud.NewLoginCmd())

	return cmd
}
