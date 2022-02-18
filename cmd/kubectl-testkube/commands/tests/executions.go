package tests

import (
	"os"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewListExecutionsCmd() *cobra.Command {
	var tags []string

	cmd := &cobra.Command{
		Use:     "executions [testName]",
		Aliases: []string{"el"},
		Short:   "List test executions",
		Long:    `Getting list of execution for given test name or recent executions if there is no test name passed`,
		Run: func(cmd *cobra.Command, args []string) {
			var testID string
			limit := 10
			if len(args) == 0 {
				testID = ""
			} else if len(args) > 0 {
				testID = args[0]
				limit = 0
			}

			client, _ := common.GetClient(cmd)
			executions, err := client.ListExecutions(testID, limit, tags)
			ui.ExitOnError("Getting executions for test: "+testID, err)

			renderer := GetExecutionsListRenderer(cmd)

			err = renderer.Render(executions, os.Stdout)
			ui.ExitOnError("rendering", err)
		},
	}
	cmd.Flags().StringSliceVar(&tags, "tags", nil, "comma separated list of tags: --tags tag1,tag2,tag3")

	return cmd
}
