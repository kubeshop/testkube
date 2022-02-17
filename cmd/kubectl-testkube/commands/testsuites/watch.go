package testsuites

import (
	"time"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/validator"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewWatchTestSuiteExecutionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "watch <executionID>",
		Aliases: []string{"w"},
		Short:   "Watch test",
		Long:    `Watch test by test execution ID, returns results to console`,
		Args:    validator.ExecutionID,
		Run: func(cmd *cobra.Command, args []string) {
			ui.Logo()

			client, _ := common.GetClient(cmd)
			startTime := time.Now()

			executionID := args[0]
			executionCh, err := client.WatchTestExecution(executionID)
			for execution := range executionCh {
				ui.ExitOnError("watching test execution", err)
				printExecution(execution, startTime)
			}

			execution, err := client.GetTestSuiteExecution(executionID)
			ui.ExitOnError("getting test excecution", err)
			printExecution(execution, startTime)
			ui.ExitOnError("getting recent execution data id:"+execution.Id, err)

			uiPrintExecutionStatus(execution)
			uiShellTestSuiteGetCommandBlock(execution.Id)
		},
	}

	return cmd
}
