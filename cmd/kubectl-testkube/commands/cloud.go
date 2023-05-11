package commands

import (
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/cloud"
	"github.com/spf13/cobra"
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

	return cmd
}
