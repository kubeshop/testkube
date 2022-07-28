package debuginfo

import (
	"fmt"
	"os"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

// NewShowDebugInfoCmd creates a new cobra command to print the debug info to the CLI
func NewShowDebugInfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "info",
		Short: "Show debug info",
		Long:  "Get all the necessary information to debug an issue in Testkube",
		Run: func(cmd *cobra.Command, args []string) {
			client, _ := common.GetClient(cmd)
			debug, err := client.GetDebugInfo()
			ui.ExitOnError("get debug info", err)

			info, err := client.GetServerInfo()
			ui.ExitOnError("get server info", err)

			debug.ClientVersion = common.Version
			debug.ServerVersion = info.Version

			printDebugInfo(debug)
		},
	}
}

// printDebugInfo prints the debugging data to the CLI
func printDebugInfo(info testkube.DebugInfo) {
	ui.Table(info, os.Stdout)
	ui.NL()

	ui.Info("API LOGS")
	ui.NL()
	for _, l := range info.ApiLogs {
		ui.Info(l)
	}
	ui.NL()

	ui.Info("OPERATOR LOGS")
	ui.NL()
	for _, l := range info.OperatorLogs {
		ui.Info(l)
	}
	ui.NL()

	ui.Info("EXECUTION LOGS")
	ui.NL()
	for id, logs := range info.ExecutionLogs {
		ui.Info(fmt.Sprintf("EXECUTION ID: %s", id))
		ui.NL()
		for _, l := range logs {
			ui.Info(l)
		}
		ui.NL()
	}
	ui.NL()
}
