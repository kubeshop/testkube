package debug

import (
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
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
