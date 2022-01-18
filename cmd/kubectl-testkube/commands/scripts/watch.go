package scripts

import (
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/validator"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewWatchExecutionCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "watch <executionID>",
		Aliases: []string{"w"},
		Short:   "Watch logs output from executor pod",
		Long:    `Gets script execution details, until it's in success/error state, blocks until gets complete state`,
		Args:    validator.ExecutionID,
		Run: func(cmd *cobra.Command, args []string) {
			executionID := args[0]

			client, _ := common.GetClient(cmd)
			execution, err := client.GetExecution("-", executionID)
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
