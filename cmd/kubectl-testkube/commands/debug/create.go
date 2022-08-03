package debuginfo

import (
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/debug/github"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewCreateTicketCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "create-ticket",
		Short: "Create bug ticket",
		Long:  "Create an issue of type bug in the Testkube repository",
		Run: func(cmd *cobra.Command, args []string) {
			client, _ := common.GetClient(cmd)
			debug, err := getDebugInfo(client)
			ui.ExitOnError("get debug info", err)
			ui.ExitOnError("opening GitHub ticket", github.OpenTicket(debug))
		},
	}
}
