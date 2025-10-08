package commands

import (
	"strings"

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
			if info.Version != "" && !strings.Contains(info.Version, "invalid character") {
				ui.Info("Server Version", info.Version)
			}
			if info.Commit != "" {
				ui.Info("Commit", common.Commit)
			}
			if common.BuiltBy != "" {
				ui.Info("Built by", common.BuiltBy)
			}
			if common.Date != "" {
				ui.Info("Build date", common.Date)
			}
		},
	}

	cmd.AddCommand(agent.NewDebugAgentCmd())

	return cmd
}
