package executors

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewDeleteExecutorCmd() *cobra.Command {
	var name string
	var selectors []string

	cmd := &cobra.Command{
		Use:   "executor [executorName]",
		Short: "Delete Executor",
		Long:  `Delete Executor Resource, pass name to delete by name`,
		Run: func(cmd *cobra.Command, args []string) {
			client, _, err := common.GetClient(cmd)
			ui.ExitOnError("getting client", err)

			if len(args) > 0 {
				name = args[0]
				err := client.DeleteExecutor(name)
				ui.ExitOnError("deleting executor: "+name, err)
				ui.SuccessAndExit("Succesfully deleted executor", name)
			}

			if len(selectors) != 0 {
				selector := strings.Join(selectors, ",")
				err := client.DeleteExecutors(selector)
				ui.ExitOnError("deleting executors by labels: "+selector, err)
				ui.SuccessAndExit("Succesfully deleted executors by labels", selector)
			}

			ui.Failf("Pass Executor name or labels to delete by labels")
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "unique executor name, you can also pass it as first argument")
	cmd.Flags().StringSliceVarP(&selectors, "label", "l", nil, "label key value pair: --label key1=value1")

	return cmd
}
