package tests

import (
	"time"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/validator"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewTestExecutionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "execution <executionID>",
		Aliases: []string{"e"},
		Short:   "Gets execution details",
		Long:    `Gets ececution details by ID`,
		Args:    validator.ExecutionID,
		Run: func(cmd *cobra.Command, args []string) {
			ui.Logo()

			startTime := time.Now()
			client, _ := common.GetClient(cmd)

			executionID := args[0]
			execution, err := client.GetTestSuiteExecution(executionID)
			ui.ExitOnError("getting recent execution data id:"+execution.Id, err)

			printTestExecutionDetails(execution, startTime)

			uiPrintTestStatus(execution)

			uiShellTestGetCommandBlock(execution.Id)
		},
	}

	return cmd
}
