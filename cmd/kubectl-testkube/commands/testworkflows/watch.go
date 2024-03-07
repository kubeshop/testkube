package testworkflows

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/render"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/validator"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/testworkflows/renderer"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewWatchTestWorkflowExecutionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "testworkflowexecution <executionName>",
		Aliases: []string{"testworkflowexecutions", "twe", "tw"},
		Args:    validator.ExecutionName,
		Short:   "Watch output from test workflow execution",
		Long:    `Gets test workflow execution details, until it's in success/error state, blocks until gets complete state`,

		Run: func(cmd *cobra.Command, args []string) {
			client, _, err := common.GetClient(cmd)
			ui.ExitOnError("getting client", err)

			executionID := args[0]
			execution, err := client.GetTestWorkflowExecution(executionID)
			ui.ExitOnError("get execution failed", err)
			err = render.Obj(cmd, execution, os.Stdout, renderer.TestWorkflowExecutionRenderer)
			ui.ExitOnError("render test workflow execution", err)

			ui.NL()
			exitCode := uiWatch(execution, client)
			ui.NL()

			uiShellGetExecution(execution.Id)
			os.Exit(exitCode)
		},
	}

	return cmd
}
