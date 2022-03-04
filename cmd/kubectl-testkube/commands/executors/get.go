package executors

import (
	"os"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/validator"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewGetExecutorCmd() *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:   "get <executorName>",
		Short: "Gets executor details",
		Long:  `Gets executor, you can change output format`,
		Args:  validator.ExecutorName,
		Run: func(cmd *cobra.Command, args []string) {
			name := args[0]

			client, _ := common.GetClient(cmd)
			executor, err := client.GetExecutor(name)
			ui.ExitOnError("getting executor: "+name, err)

			render := GetExecutorRenderer(cmd)
			err = render.Render(executor, os.Stdout)
			ui.ExitOnError("rendering", err)
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "unique executor name, you can also pass it as argument")

	return cmd
}
