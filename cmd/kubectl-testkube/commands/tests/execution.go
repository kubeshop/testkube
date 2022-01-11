package tests

import (
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewExecutionCmd() *cobra.Command {
	var tags []string

	cmd := &cobra.Command{
		Use:   "execution",
		Short: "Get test execution by its ID",
		Long:  `Getting all test executions`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				ui.Errf("Please pass test execution id")
			}

			id := args[0]

			client, _ := GetClient(cmd)
			execution, err := client.GetTestExecution(id)
			ui.ExitOnError("getting test execution "+id, err)

			printTestExecutionDetails(execution)

		},
	}
	cmd.Flags().StringSliceVar(&tags, "tags", nil, "comma separated list of tags: --tags tag1,tag2,tag3")
	return cmd
}
