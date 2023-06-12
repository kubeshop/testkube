package tests

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/validator"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewAbortExecutionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "execution <executionName>",
		Short: "Aborts execution of the test",
		Args:  validator.ExecutionName,
		Run: func(cmd *cobra.Command, args []string) {
			executionID := args[0]

			client, _, err := common.GetClient(cmd)
			ui.ExitOnError("getting client", err)

			err = client.AbortExecution("test", executionID)
			ui.ExitOnError(fmt.Sprintf("aborting execution %s", executionID), err)
			ui.SuccessAndExit("Succesfully aborted test", executionID)
		},
	}
}

func NewAbortExecutionsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "executions <testName>",
		Short: "Aborts all executions of the test",
		Args:  validator.ExecutionName,
		Run: func(cmd *cobra.Command, args []string) {
			testName := args[0]

			client, _, err := common.GetClient(cmd)
			ui.ExitOnError("getting client", err)

			err = client.AbortExecutions(testName)
			ui.ExitOnError(fmt.Sprintf("aborting executions of test %s", testName), err)
			ui.SuccessAndExit("Succesfully aborted all executions of the test", testName)
		},
	}
}
