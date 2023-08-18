package commands

import (
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/validator"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/debug"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/ui"
)

// NewDebugCmd creates the 'testkube debug' command
func NewDebugCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "debug",
		Short: "Print environment information for debugging",
		Run: func(cmd *cobra.Command, args []string) {

			cfg, err := config.Load()
			ui.ExitOnError("loading config file", err)

			if cfg.ContextType == config.ContextTypeCloud {
				cfg, err := config.Load()
				ui.ExitOnError("loading config file", err)
				ui.NL()

				if cfg.ContextType != config.ContextTypeCloud {
					ui.Errf("Agent debug is only available for cloud context")
					ui.NL()
					ui.ShellCommand("Please try command below to set your context into Cloud mode", `testkube set context -o <org> -e <env> -k <api-key> `)
					ui.NL()
					return
				}

				common.UiPrintContext(cfg)

				client, _, err := common.GetClient(cmd)
				ui.ExitOnError("getting client", err)

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
			} else {
				client, _, err := common.GetClient(cmd)
				ui.ExitOnError("getting client", err)

				d, err := debug.GetDebugInfo(client)
				ui.ExitOnError("get debug info", err)

				debug.PrintDebugInfo(d)

			}
		},
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			cfg, err := config.Load()
			ui.ExitOnError("loading config", err)
			common.UiContextHeader(cmd, cfg)

			validator.PersistentPreRunVersionCheck(cmd, common.Version)
		}}

	cmd.AddCommand(debug.NewShowDebugInfoCmd())

	return cmd
}
