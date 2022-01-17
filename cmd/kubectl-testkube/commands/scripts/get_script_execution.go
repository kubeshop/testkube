package scripts

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/pkg/ui"
)

func NewGetExecutionCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "execution",
		Aliases: []string{"e"},
		Short:   "Gets script execution details",
		Long:    `Gets script execution details, you can change output format`,
		Run: func(cmd *cobra.Command, args []string) {

			// get args
			// - 1 - executionID as it's unique for script executions
			// - 2 - scriptName + executionName
			var scriptID, executionID string
			if len(args) == 1 {
				executionID = args[0]
			} else if len(args) == 2 {
				scriptID = args[0]
				executionID = args[1]
			} else {
				ui.Failf("invalid script arguments please pass execution id or script name and execution name pair")
			}

			client, _ := GetClient(cmd)
			execution, err := client.GetExecution(executionID)
			ui.ExitOnError("getting script execution: "+scriptID+"/"+executionID, err)

			render := GetExecutionRenderer(cmd)
			err = render.Render(execution, os.Stdout)
			ui.ExitOnError("rendering", err)
		},
	}
}
