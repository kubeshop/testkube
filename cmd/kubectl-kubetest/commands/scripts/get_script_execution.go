package scripts

import (
	"fmt"

	"github.com/kubeshop/kubetest/pkg/api/client"
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

		client := client.NewDefaultScriptsAPI()
		scriptExecution, err := client.GetExecution(scriptID, executionID)
		ui.ExitOnError("getting API for script completion", err)
		if scriptExecution.Execution.ErrorMessage != "" {
			ui.Errf("Script execution error")
			fmt.Println(scriptExecution.Execution.ErrorMessage)
		} else {
			ui.Info("Script execution success")

		}
		fmt.Println(scriptExecution.Execution.Output)
	},
}
