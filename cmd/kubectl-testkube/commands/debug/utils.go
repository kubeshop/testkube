package debug

import (
	"os"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/ui"
)

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
}
