package debug

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/ui"
)

// NewShowDebugInfoCmd creates a new cobra command to print the debug info to the CLI
func NewShowDebugInfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "info",
		Short: "Show debug info",
		Long:  "Get all the necessary information to debug an issue in Testkube",
		Run: func(cmd *cobra.Command, args []string) {
			client, _, err := common.GetClient(cmd)
			ui.ExitOnError("getting client", err)

			debug, err := GetDebugInfo(client)
			ui.ExitOnError("get debug info", err)

			PrintDebugInfo(debug)
		},
	}
}

// GetDebugInfo returns information on the current Testkube environment
func GetDebugInfo(apiClient client.Client) (testkube.DebugInfo, error) {
	debug, err := apiClient.GetDebugInfo()
	if err != nil {
		return testkube.DebugInfo{}, err
	}

	info, err := apiClient.GetServerInfo()
	if err != nil {
		return testkube.DebugInfo{}, err
	}

	debug.ClientVersion = common.Version
	debug.ServerVersion = info.Version

	return debug, nil
}

// PrintDebugInfo prints the debugging data to the CLI
func PrintDebugInfo(info testkube.DebugInfo) {
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
