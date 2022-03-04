package executors

import (
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/validator"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewDeleteExecutorCmd() *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:   "delete <executorName>",
		Short: "Delete executor",
		Long:  `Delete executor, pass name to delete`,
		Args:  validator.ExecutorName,
		Run: func(cmd *cobra.Command, args []string) {
			name = args[0]

			client, _ := common.GetClient(cmd)

			err := client.DeleteExecutor(name)
			ui.ExitOnError("deleting executor: "+name, err)

			ui.Success("Executor deleted")
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "unique executor name, you can also pass it as first argument")

	return cmd
}
