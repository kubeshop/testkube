package agent

import (
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewAgentDebugCmd() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "debug",
		Short: "Debug Agent info",
		Run: func(cmd *cobra.Command, args []string) {

			cfg, err := config.Load()
			ui.ExitOnError("loading config file", err)

			ui.Info("Your Testkube Cloud Context")
			ui.NL()
			common.UiPrintContext(cfg)

			client, _ := common.GetClient(cmd)
			i, err := client.GetServerInfo()
			if err != nil {
				ui.Errf("Error %v", err)
				ui.NL()
				ui.Info("Possible reasons:")
				ui.Warn("- Please check if your agent organization and environment are set correctly")
				ui.Warn("- Please check if your API token is set correctly")
				ui.NL()
			} else {
				ui.Warn("Agent correctly connected to cloud:\n")
				ui.InfoGrid(map[string]string{
					"Agent version  ": i.Version,
					"Agent namespace": i.Namespace,
				})
			}
			ui.NL()
		},
	}

	return cmd
}
