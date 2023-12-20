package artifacts

import (
	"os"

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
		Short:   "List artifacts of the given test or test suite execution name",
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
				artifacts, errArtifacts = client.GetExecutionArtifacts(executionID)
				ui.ExitOnError("getting test artifacts ", errArtifacts)
			} else {
				_, err := client.GetTestSuiteExecution(executionID)
				ui.ExitOnError("no test or test suite execution was found with the following id", err)
				artifacts, errArtifacts = client.GetTestSuiteExecutionArtifacts(executionID)
				ui.ExitOnError("getting test suite artifacts ", errArtifacts)
			}

			ui.Table(artifacts, os.Stdout)
		},
	}

	cmd.PersistentFlags().StringVarP(&executionID, "execution-id", "e", "", "ID of the execution")

	// output renderer flags
	return cmd
}
