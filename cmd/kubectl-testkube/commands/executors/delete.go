package executors

import (
	"strings"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewDeleteExecutorCmd() *cobra.Command {
	var name string
	var selectors []string

	cmd := &cobra.Command{
		Use:   "executor [executorName]",
		Short: "Delete Executor",
		Long:  `Delete Executor Resource, pass name to delete by name`,
		Run: func(cmd *cobra.Command, args []string) {
			client, _ := common.GetClient(cmd)
			if len(args) > 0 {
				name = args[0]
				err := client.DeleteExecutor(name)
				ui.ExitOnError("deleting executor: "+name, err)
			} else if len(selectors) != 0 {
				selector := strings.Join(selectors, ",")
				err := client.DeleteExecutors(selector)
				ui.ExitOnError("deleting executors by labels: "+selector, err)
			} else {
				ui.Failf("Pass Executor name or labels to delete by labels ")
			}

			ui.Success("Executor deleted")
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "unique executor name, you can also pass it as first argument")
	cmd.Flags().StringSliceVarP(&selectors, "label", "l", nil, "label key value pair: --label key1=value1")

	return cmd
}
