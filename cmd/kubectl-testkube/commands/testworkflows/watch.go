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
	var (
		serviceName       string
		parallelStepName  string
		serviceIndex      int
		parallelStepIndex int
	)

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
			var pServiceName, pParallelStepName *string
			if cmd.Flag("service-name").Changed || cmd.Flag("service-index").Changed {
				pServiceName = &serviceName
			}
			if cmd.Flag("parallel-step-name").Changed || cmd.Flag("parallel-step-index").Changed {
				pParallelStepName = &parallelStepName
			}

			exitCode := uiWatch(execution, pServiceName, serviceIndex, pParallelStepName, parallelStepIndex, client, "")
			ui.NL()

			execution, err = client.GetTestWorkflowExecution(execution.Id)
			ui.ExitOnError("get execution failed", err)

			render.PrintTestWorkflowExecutionURIs(&execution)
			uiShellGetExecution(execution.Id)
			os.Exit(exitCode)
		},
	}

	cmd.Flags().StringVar(&serviceName, "service-name", "", "test workflow service name")
	cmd.Flags().IntVar(&serviceIndex, "service-index", 0, "test workflow service index starting from 0")
	cmd.Flags().StringVar(&parallelStepName, "parallel-step-name", "", "test workflow parallel step name or reference")
	cmd.Flags().IntVar(&parallelStepIndex, "parallel-step-index", 0, "test workflow parallel step index starting from 0")

	return cmd
}
