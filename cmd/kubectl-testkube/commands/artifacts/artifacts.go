package artifacts

import (
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/validator"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/ui"
)

var (
	executionID string
)

func NewListArtifactsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "artifact <executionName>",
		Aliases: []string{"artifacts"},
		Short:   "List artifacts of the given test, test suite or test workflow execution name",
		Args:    validator.ExecutionName,
		Run: func(cmd *cobra.Command, args []string) {
			executionID = args[0]
			cmd.SilenceUsage = true
			client, _, err := common.GetClient(cmd)
			ui.ExitOnError("getting client", err)

			execution, err := client.GetExecution(executionID)
			var artifacts testkube.Artifacts
			var errArtifacts error
			if err == nil && execution.Id != "" {
				artifacts, errArtifacts = client.GetExecutionArtifacts(execution.Id)
				ui.ExitOnError("getting test artifacts", errArtifacts)
				ui.Table(artifacts, os.Stdout)
				return
			}
			tsExecution, err := client.GetTestSuiteExecution(executionID)
			if err == nil && tsExecution.Id != "" {
				artifacts, errArtifacts = client.GetTestSuiteExecutionArtifacts(tsExecution.Id)
				ui.ExitOnError("getting test suite artifacts", errArtifacts)
				ui.Table(artifacts, os.Stdout)
				return
			}
			twExecution, err := client.GetTestWorkflowExecution(executionID)
			if err == nil && twExecution.Id != "" {
				artifacts, errArtifacts = client.GetTestWorkflowExecutionArtifacts(twExecution.Id)
				ui.ExitOnError("getting test workflow artifacts", errArtifacts)
				ui.Table(artifacts, os.Stdout)
				return
			}
			if err == nil {
				err = errors.New("no test, test suite or test workflow execution was found with the following id")
			}
			ui.Fail(err)
		},
	}

	cmd.PersistentFlags().StringVarP(&executionID, "execution-id", "e", "", "ID of the execution")

	// output renderer flags
	return cmd
}
