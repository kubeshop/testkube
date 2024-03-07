package testworkflows

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/validator"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewAbortTestWorkflowExecutionCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "testworkflowexecution <executionName>",
		Aliases: []string{"twe", "testworkflows-execution", "testworkflow-execution"},
		Short:   "Abort test workflow execution",
		Args:    validator.ExecutionName,

		Run: func(cmd *cobra.Command, args []string) {
			executionID := args[0]

			client, _, err := common.GetClient(cmd)
			ui.ExitOnError("getting client", err)

			execution, err := client.GetTestWorkflowExecution(executionID)
			ui.ExitOnError("get execution failed", err)

			err = client.AbortTestWorkflowExecution(execution.Workflow.Name, execution.Id)
			ui.ExitOnError(fmt.Sprintf("aborting testworkflow execution %s", executionID), err)

			ui.SuccessAndExit("Succesfully aborted test workflow execution", executionID)
		},
	}
}

func NewAbortTestWorkflowExecutionsCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "testworkflowexecutions <testWorkflowName>",
		Aliases: []string{"twes", "testworkflows-executions", "testworkflow-executions"},
		Short:   "Abort all test workflow executions",
		Args:    cobra.ExactArgs(1),

		Run: func(cmd *cobra.Command, args []string) {
			testWorkflowName := args[0]

			client, _, err := common.GetClient(cmd)
			ui.ExitOnError("getting client", err)

			err = client.AbortTestWorkflowExecutions(testWorkflowName)
			ui.ExitOnError(fmt.Sprintf("aborting test workflow executions for test workflow %s", testWorkflowName), err)

			ui.SuccessAndExit("Successfully aborted all test workflow executions", testWorkflowName)
		},
	}
}
