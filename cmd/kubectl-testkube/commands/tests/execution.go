package tests

import (
	"fmt"

	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewTestExecutionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "execution",
		Aliases: []string{"e"},
		Short:   "Gets execution details",
		Long:    `Gets ececution details by ID`,
		Run: func(cmd *cobra.Command, args []string) {
			ui.Logo()

			if len(args) == 0 {
				ui.ExitOnError("Invalid arguments", fmt.Errorf("please pass execution ID"))
			}

			client, _ := GetClient(cmd)

			executionID := args[0]
			execution, err := client.GetTestExecution(executionID)
			ui.ExitOnError("getting recent execution data id:"+execution.Id, err)

			printTestExecutionDetails(execution)

			uiPrintTestStatus(execution)

			uiShellTestGetCommandBlock(execution.Id)
		},
	}

	return cmd
}
