package testsuites

import (
	"time"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/validator"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewWatchTestSuiteExecutionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "testsuiteexecution <executionName>",
		Aliases: []string{"tse", "testsuites-execution", "testsuite-execution"},
		Short:   "Watch test suite",
		Long:    `Watch test suite by execution ID, returns results to console`,
		Args:    validator.ExecutionName,
		Run: func(cmd *cobra.Command, args []string) {

			client, _ := common.GetClient(cmd)
			startTime := time.Now()

			executionID := args[0]
			executionCh, err := client.WatchTestSuiteExecution(executionID)
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
