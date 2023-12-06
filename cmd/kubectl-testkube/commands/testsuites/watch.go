package testsuites

import (
	"os"
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

			client, _, err := common.GetClient(cmd)
			ui.ExitOnError("getting client", err)

			startTime := time.Now()

			executionID := args[0]
			watchResp := client.WatchTestSuiteExecution(executionID)
			for resp := range watchResp {
				ui.ExitOnError("watching test suite execution", resp.Error)
				printExecution(resp.Execution, startTime)
			}

			execution, err := client.GetTestSuiteExecution(executionID)
			ui.ExitOnError("getting test suite excecution", err)
			printExecution(execution, startTime)
			ui.ExitOnError("getting recent execution data id:"+execution.Id, err)

			err = uiPrintExecutionStatus(client, execution)
			uiShellTestSuiteGetCommandBlock(execution.Id)
			if err != nil {
				os.Exit(1)
			}
		},
	}

	return cmd
}
