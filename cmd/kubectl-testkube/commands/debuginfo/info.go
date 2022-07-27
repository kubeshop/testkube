package debuginfo

import (
	"fmt"
	"os"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewShowDebugInfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "info",
		Short: "Show debug info",
		Long:  "Getting all the necessary information to debug an issue",
		Run: func(cmd *cobra.Command, args []string) {
			// client, _ := common.GetClient(cmd)
			// debug := client.GetDebugInfo()
			debug := testkube.DebugInfo{
				ClientVersion:  "v0.0.test",
				ServerVersion:  "v0.0.test",
				ClusterVersion: "v0.0.test",
				ApiLogs:        []string{"log1", "log2", "log3"},
				OperatorLogs:   []string{"log1", "log2", "log3"},
				ExecutionLogs: map[string][]string{
					"e1": {"log1", "log2", "log3"},
					"e2": {"log1", "log2", "log3"},
					"e3": {"log1", "log2", "log3"},
				},
			}
			printDebugInfo(debug)
		},
	}
}

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
		ui.Info(fmt.Sprintf("ID: %s", id))
		ui.NL()
		for _, l := range logs {
			ui.Info(l)
		}
		ui.NL()
	}
	ui.NL()
}
