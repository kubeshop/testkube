package scripts

import (
	"fmt"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/validator"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewAbortExecutionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "abort <executionID>",
		Short: "Aborts execution of the script",
		Args:  validator.ExecutionID,
		Run: func(cmd *cobra.Command, args []string) {
			executionID := args[0]

			client, _ := common.GetClient(cmd)

			err := client.AbortExecution("script", executionID)
			ui.ExitOnError(fmt.Sprintf("aborting execution %s", executionID), err)
		},
	}
}
