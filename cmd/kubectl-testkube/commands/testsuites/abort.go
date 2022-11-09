package testsuites

import (
	"fmt"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/validator"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewAbortTestSuiteExecutionCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "testsuiteexecution <executionID>",
		Aliases: []string{"tse", "testsuites-execution", "testsuite-execution"},
		Short:   "Abort test suite execution",
		Args:    validator.ExecutionID,
		Run: func(cmd *cobra.Command, args []string) {
			executionID := args[0]

			client, _ := common.GetClient(cmd)

			err := client.AbortTestSuiteExecution(executionID)
			ui.ExitOnError(fmt.Sprintf("aborting testsuite execution %s", executionID), err)

			ui.SuccessAndExit("Succesfully aborted test suite", executionID)
		},
	}
}
