package scripts

import (
	"os"
	"time"

	"github.com/kubeshop/kubtest/pkg/ui"
	"github.com/spf13/cobra"
)

func NewWatchScriptExecutionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "watch",
		Short: "Watch until script execution is in complete state",
		Long:  `Gets script execution details, until it's in success/error state, blocks until gets complete state`,
		Run: func(cmd *cobra.Command, args []string) {

			// get args
			// - 1 - executionID as it's unique for script executions
			// - 2 - scriptName + executionName
			var scriptID, executionID string
			if len(args) == 1 {
				scriptID = "-"
				executionID = args[0]
			} else if len(args) == 2 {
				scriptID = args[0]
				executionID = args[1]
			} else {
				ui.Failf("invalid script arguments please pass execution id or script name and execution name pair")
			}

			client, _ := GetClient(cmd)

			scriptExecution, err := client.GetExecution(scriptID, executionID)
			ui.ExitOnError("get script execution details", err)
			ui.Warn("Type          :", scriptExecution.ScriptType)
			ui.Warn("Name          :", scriptExecution.ScriptName)
			ui.Warn("Execution ID  :", scriptExecution.Execution.Id)
			ui.Warn("Execution name:", scriptExecution.Name)
			ui.BR()

			ui.Info("Watching for changes")
			for range time.Tick(time.Second) {
				scriptExecution, err := client.GetExecution(scriptID, executionID)
				ui.ExitOnError("get script execution details", err)
				render := GetRenderer(cmd)
				err = render.Watch(scriptExecution, os.Stdout)
				ui.ExitOnError("watching for changes", err)
				if scriptExecution.Execution.IsCompleted() {
					ui.Info("\nGetting results")
					render.Render(scriptExecution, os.Stdout)
					ui.Warn("Script execution completed in", scriptExecution.Execution.Duration().String())
					return
				}
			}
		},
	}
}
