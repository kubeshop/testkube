package tests

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
			if len(args) == 0 {
				ui.Errf("Please pass test name as argument")
			}
			testID := args[0]

			client, _ := GetClient(cmd)
			executions, err := client.ListTestExecutions(testID, 10000, []string{})
			ui.ExitOnError("Getting executions for test: "+testID, err)

			renderer := GetExecutionsListRenderer(cmd)

			err = renderer.Render(executions, os.Stdout)
			ui.ExitOnError("rendering", err)
		},
	}
	cmd.Flags().StringSliceVar(&tags, "tags", nil, "--tags 1,2,3")

	return cmd
}
