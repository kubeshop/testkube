package executors

import (
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewDeleteExecutorCmd() *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Gets executordetails",
		Long:  `Gets executor, you can change output format`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 1 {
				name = args[0]
			}

			if name == "" {
				ui.Failf("Please pass executor name")
			}

			client, _ := GetClient(cmd)

			err := client.DeleteExecutor(name)
			ui.ExitOnError("deleting executor: "+name, err)
			ui.Success("Executor deleted")
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "unique executor name, you can also pass it as first argument")

	return cmd
}
