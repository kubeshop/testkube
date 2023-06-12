package commands

import (
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	debuginfo "github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/debug"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/github"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewCreateTicketCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "create-ticket",
		Short: "Create bug ticket",
		Long:  "Create an issue of type bug in the Testkube repository",
		Run: func(cmd *cobra.Command, args []string) {
			client, _, err := common.GetClient(cmd)
			ui.ExitOnError("getting client", err)

			debug, err := debuginfo.GetDebugInfo(client)
			ui.ExitOnError("get debug info", err)
			ui.ExitOnError("opening GitHub ticket", github.OpenTicket(debug))
		},
	}
}
