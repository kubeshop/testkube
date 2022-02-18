package testsuites

import (
	"time"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/validator"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewTestSuiteExecutionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "execution <executionID>",
		Aliases: []string{"e"},
		Short:   "Gets test suite execution details",
		Long:    `Gets test suite execution details by ID`,
		Args:    validator.ExecutionID,
		Run: func(cmd *cobra.Command, args []string) {
			ui.Logo()

			startTime := time.Now()
			client, _ := common.GetClient(cmd)

			executionID := args[0]
			execution, err := client.GetTestSuiteExecution(executionID)
			ui.ExitOnError("getting recent test suite execution data id:"+execution.Id, err)

			printExecution(execution, startTime)

			uiPrintExecutionStatus(execution)

			uiShellTestSuiteGetCommandBlock(execution.Id)
		},
	}

	return cmd
}
