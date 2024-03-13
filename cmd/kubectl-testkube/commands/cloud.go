package commands

import (
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/pro"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewCloudCmd() *cobra.Command {

	cmd := &cobra.Command{
		Use:        "cloud",
		Deprecated: "Use `testkube pro` instead",
		Hidden:     true,
		Short:      "[Deprecated] Testkube Cloud commands",
		Aliases:    []string{"cl"},
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			ui.Errf("You are using a deprecated command, please switch to `testkube pro` prefix.\n\n")
		},
	}

	cmd.AddCommand(pro.NewConnectCmd())
	cmd.AddCommand(pro.NewDisconnectCmd())
	cmd.AddCommand(pro.NewInitCmd())
	cmd.AddCommand(pro.NewLoginCmd())

	return cmd
}
