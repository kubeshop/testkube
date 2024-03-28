package tests

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/validator"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewWatchExecutionCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "execution <executionName>",
		Aliases: []string{"e", "executions"},
		Short:   "Watch logs output from executor pod",
		Long:    `Gets test execution details, until it's in success/error state, blocks until gets complete state`,
		Args:    validator.ExecutionName,
		Run: func(cmd *cobra.Command, args []string) {
			client, _, err := common.GetClient(cmd)
			ui.ExitOnError("getting client", err)

			executionID := args[0]
			execution, err := client.GetExecution(executionID)
			ui.ExitOnError("get execution failed", err)

			if execution.ExecutionResult.IsCompleted() {
				ui.Completed("execution is already finished")
			} else {
				info, err := client.GetServerInfo()
				ui.ExitOnError("getting server info", err)

				if info.Features != nil && info.Features.LogsV2 {
					err = watchLogsV2(execution.Id, false, client)
				} else {
					err = watchLogs(execution.Id, false, client)
				}

				ui.NL()
				uiShellGetExecution(execution.Id)
				if err != nil {
					os.Exit(1)
				}

			}

		},
	}
}
