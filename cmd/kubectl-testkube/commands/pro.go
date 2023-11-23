package commands

import (
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/pro"
)

func NewProCmd() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "pro",
		Short: "Testkube Pro commands",
		Run: func(cmd *cobra.Command, args []string) {
		},
	}

	cmd.AddCommand(pro.NewConnectCmd())
	cmd.AddCommand(pro.NewDisconnectCmd())
	cmd.AddCommand(pro.NewInitCmd())
	cmd.AddCommand(pro.NewLoginCmd())

	return cmd
}
