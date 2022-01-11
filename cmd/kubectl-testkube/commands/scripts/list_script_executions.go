package scripts

import (
	"os"

	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewListExecutionsCmd() *cobra.Command {
	var tags []string

	cmd := &cobra.Command{
		Use:   "executions",
		Short: "List scripts executions",
		Long:  `Getting list of execution for given script name or recent executions if there is no script name passed`,
		Run: func(cmd *cobra.Command, args []string) {
			var scriptID string
			limit := 10
			if len(args) == 0 {
				scriptID = "-"
			} else if len(args) > 0 {
				scriptID = args[0]
				limit = 0
			}

			client, _ := GetClient(cmd)
			executions, err := client.ListExecutions(scriptID, limit, tags)
			ui.ExitOnError("Getting executions for script: "+scriptID, err)

			renderer := GetExecutionsListRenderer(cmd)

			err = renderer.Render(executions, os.Stdout)
			ui.ExitOnError("rendering", err)
		},
	}
	cmd.Flags().StringSliceVar(&tags, "tags", nil, "comma separated list of tags: --tags tag1,tag2,tag3")

	return cmd
}
