package scripts

import (
	"os"

	"github.com/kubeshop/kubtest/pkg/ui"
	"github.com/spf13/cobra"
)

var GetScriptExecutionCmd = &cobra.Command{
	Use:   "execution",
	Short: "Gets script execution details",
	Long:  `Gets script execution details, you can change output format`,
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
		err = render.Render(scriptExecution, os.Stdout)
		ui.ExitOnError("rendering", err)
	},
}
