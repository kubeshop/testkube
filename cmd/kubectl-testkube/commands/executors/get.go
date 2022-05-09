package executors

import (
	"os"
	"strings"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common/render"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewGetExecutorCmd() *cobra.Command {
	var selectors []string

	cmd := &cobra.Command{
		Use:     "executor [executorName]",
		Aliases: []string{"executors", "er"},
		Short:   "Gets executor details",
		Long:    `Gets executor, you can change output format`,
		Run: func(cmd *cobra.Command, args []string) {
			client, _ := common.GetClient(cmd)

			if len(args) > 0 {
				name := args[0]

				executor, err := client.GetExecutor(name)
				ui.ExitOnError("getting executor: "+name, err)
				err = render.Obj(cmd, executor, os.Stdout)
				ui.ExitOnError("rendering executor", err)

			} else {
				executors, err := client.ListExecutors(strings.Join(selectors, ","))
				ui.ExitOnError("listing executors: ", err)
				err = render.List(cmd, executors, os.Stdout)
				ui.ExitOnError("rendering executors", err)
			}
		},
	}

	cmd.Flags().StringSliceVarP(&selectors, "label", "l", nil, "label key value pair: --label key1=value1")
	return cmd
}
