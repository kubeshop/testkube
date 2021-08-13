package scripts

import (
	"os"

	"github.com/kubeshop/kubetest/pkg/ui"
	"github.com/spf13/cobra"
)

var GetScriptExecutionCmd = &cobra.Command{
	Use:   "execution",
	Short: "Gets script execution details",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 2 {
			ui.Failf("invalid script arguments please pass test name and execution id")
		}

		scriptID := args[0]
		executionID := args[1]

		client := GetClient(cmd)
		scriptExecution, err := client.GetExecution(scriptID, executionID)
		ui.ExitOnError("getting API for script completion", err)

		render := GetRenderer(cmd)
		render.Render(scriptExecution, os.Stdout)
	},
}
