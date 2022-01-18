package scripts

import (
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/pkg/ui"
)

func NewWatchExecutionCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "watch",
		Aliases: []string{"w"},
		Short:   "Watch logs output from executor pod",
		Long:    `Gets script execution details, until it's in success/error state, blocks until gets complete state`,
		Run: func(cmd *cobra.Command, args []string) {

			var executionID string
			if len(args) == 1 {
				executionID = args[0]
			} else {
				ui.Failf("invalid script arguments please pass execution id or script name and execution name pair")
			}

			client, _ := GetClient(cmd)
			execution, err := client.GetExecution(executionID)
			if err != nil {

				ui.Failf("execution result retrievel failed with err %s", err)
			}

			if execution.ExecutionResult.IsCompleted() {
				ui.Completed("execution is already finished")
			} else {
				watchLogs(executionID, client)
			}

		},
	}
}
