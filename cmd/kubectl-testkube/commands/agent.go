package commands

import (
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/agent"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewAgentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Testkube Pro Agent related commands",
		Run: func(cmd *cobra.Command, args []string) {
			client, _, err := common.GetClient(cmd)
			ui.ExitOnError("getting client", err)

			info, err := client.GetServerInfo()
			if err != nil {
				info.Version = info.Version + " " + err.Error()
			}

			ui.Logo()
			ui.Info("Client Version", common.Version)
			ui.Info("Server Version", info.Version)
			ui.Info("Commit", common.Commit)
			ui.Info("Built by", common.BuiltBy)
			ui.Info("Build date", common.Date)

		},
	}

	cmd.AddCommand(agent.NewAgentDebugCmd())

	return cmd
}
