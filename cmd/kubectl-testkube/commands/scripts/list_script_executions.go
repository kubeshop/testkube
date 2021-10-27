package scripts

import (
	"os"

	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewListExecutionsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "executions",
		Short: "List scripts executions",
		Long:  `Getting list of execution for given script name or recent executions if there is no script name passed`,
		Run: func(cmd *cobra.Command, args []string) {
			var scriptID string
			if len(args) == 0 {
				scriptID = "-"
			} else if len(args) > 0 {
				scriptID = args[0]
			}

			client, _ := GetClient(cmd)
			executions, err := client.ListExecutions(scriptID)
			ui.ExitOnError("Getting executions for script: "+scriptID, err)

			renderer := GetExecutionsListRenderer(cmd)

			err = renderer.Render(executions, os.Stdout)
			ui.ExitOnError("rendering", err)
		},
	}
}
