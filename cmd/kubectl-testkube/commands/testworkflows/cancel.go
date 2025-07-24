package testworkflows

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/validator"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewCancelTestWorkflowExecutionCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "testworkflowexecution <executionName>",
		Aliases: []string{"twe", "testworkflows-execution", "testworkflow-execution"},
		Short:   "Cancel test workflow execution",
		Args:    validator.ExecutionName,

		Run: func(cmd *cobra.Command, args []string) {
			executionID := args[0]

			client, _, err := common.GetClient(cmd)
			ui.ExitOnError("getting client", err)

			execution, err := client.GetTestWorkflowExecution(executionID)
			ui.ExitOnError("get execution failed", err)

			// TODO: interface method was never renamed from abort to cancel
			err = client.AbortTestWorkflowExecution(execution.Workflow.Name, execution.Id)
			ui.ExitOnError(fmt.Sprintf("canceling testworkflow execution %s", executionID), err)

			ui.SuccessAndExit("Succesfully canceled test workflow execution", executionID)
		},
	}
}

func NewCancelTestWorkflowExecutionsCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "testworkflowexecutions <testWorkflowName>",
		Aliases: []string{"twes", "testworkflows-executions", "testworkflow-executions"},
		Short:   "Cancel all test workflow executions",
		Args:    cobra.ExactArgs(1),

		Run: func(cmd *cobra.Command, args []string) {
			testWorkflowName := args[0]

			client, _, err := common.GetClient(cmd)
			ui.ExitOnError("getting client", err)

			// TODO: interface method was never renamed from abort to cancel
			err = client.AbortTestWorkflowExecutions(testWorkflowName)
			ui.ExitOnError(fmt.Sprintf("canceling test workflow executions for test workflow %s", testWorkflowName), err)

			ui.SuccessAndExit("Successfully canceled all test workflow executions", testWorkflowName)
		},
	}
}
