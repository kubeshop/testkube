package scripts

import (
	"os"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/validator"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewGetExecutionCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "execution <executionID>",
		Aliases: []string{"e"},
		Short:   "Gets script execution details",
		Long:    `Gets script execution details, you can change output format`,
		Args:    validator.ExecutionID,
		Run: func(cmd *cobra.Command, args []string) {

			executionID := args[0]

			client, _ := common.GetClient(cmd)
			execution, err := client.GetExecution(executionID)
			ui.ExitOnError("getting script execution: "+executionID, err)

			render := GetExecutionRenderer(cmd)
			err = render.Render(execution, os.Stdout)
			ui.ExitOnError("rendering", err)
		},
	}
}
